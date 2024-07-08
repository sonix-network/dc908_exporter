package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"

	log "github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
)

type MetricCallback func(m *metricRegistry, json string, groups []string) error

var (
	matchers = []struct {
		re *regexp.Regexp
		cb MetricCallback
	}{
		{regexp.MustCompile(`/openconfig-platform:components/component\[name=([^,\]]+)\]/fan/state`), handleFan},
		{regexp.MustCompile(`/openconfig-platform:components/component\[name=([^,\]]+)\]/state`), handleTemperature},
		{regexp.MustCompile(`/openconfig-platform:components/component\[name=([^,\]]+)\]/state`), handleMemory},
		{regexp.MustCompile(`/openconfig-platform:components/component\[name=([^,\]]+)\]/cpu/openconfig-platform-cpu:utilization`), handleCPUUtilization},
		{regexp.MustCompile(`/openconfig-platform:components/component\[name=([^,\]]+)\]/power-supply/state`), handlePowerSupply},
		{regexp.MustCompile(`/openconfig-platform:components/component\[name=([^,\]]+)\]/openconfig-platform-transceiver:transceiver/physical-channels/channel\[index=([^,\]]+)\]/state`), handleGeneralLaser},
		{regexp.MustCompile(`/openconfig-platform:components/component\[name=([^,\]]+)\]/openconfig-platform-transceiver:transceiver/state`), handleGeneralLaser},
		{regexp.MustCompile(`/openconfig-platform:components/component\[name=([^,\]]+)\]/openconfig-terminal-device:optical-channel/state`), handleGeneralLaser},
		{regexp.MustCompile(`/openconfig-platform:components/component\[name=([^,\]]+)\]/openconfig-terminal-device:optical-channel/state`), handleTerminalLaser},
	}
)

type metricRegistry struct {
	r *prometheus.Registry

	fanRPM                          *prometheus.GaugeVec
	temperature                     *prometheus.GaugeVec
	memoryUtilized                  *prometheus.GaugeVec
	cpuUtilization                  *prometheus.GaugeVec
	powerSupplyInputCurrent         *prometheus.GaugeVec
	powerSupplyInputVoltage         *prometheus.GaugeVec
	powerSupplyOutputCurrent        *prometheus.GaugeVec
	powerSupplyOutputVoltage        *prometheus.GaugeVec
	laserInputPower                 *prometheus.GaugeVec
	laserBiasCurrent                *prometheus.GaugeVec
	laserOutputPower                *prometheus.GaugeVec
	laserChromaticDispersion        *prometheus.GaugeVec
	laserPolarizationDependetLoss   *prometheus.GaugeVec
	laserPolarizationModeDispersion *prometheus.GaugeVec
	laserFrequencyOffset            *prometheus.GaugeVec
}

func NewMetricRegistry() *metricRegistry {
	m := &metricRegistry{
		r: prometheus.NewPedanticRegistry(),
		fanRPM: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "dc908_fan_rpm",
			Help: "Current fan speed in RPM.",
		},
			[]string{"device"}),
		temperature: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "dc908_temperature_celsius",
			Help: "Current temperature of components.",
		},
			[]string{"device"}),
		memoryUtilized: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "dc908_memory_utilized_bytes",
			Help: "The number of bytes of memory currently in use by processes running on the component, not considering reserved memory that is not available for use.",
		},
			[]string{"device"}),
		cpuUtilization: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "dc908_cpu_utilization_ratio",
			Help: "Ratio (0.0 - 1.0) of CPU utilization.",
		},
			[]string{"device"}),
		powerSupplyInputCurrent: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "dc908_power_supply_input_current_ampere",
			Help: "Current input current on a power supply.",
		},
			[]string{"device"}),
		powerSupplyInputVoltage: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "dc908_power_supply_input_voltage",
			Help: "Current input current on a power supply.",
		},
			[]string{"device"}),
		powerSupplyOutputCurrent: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "dc908_power_supply_output_current_ampere",
			Help: "Current output current on a power supply.",
		},
			[]string{"device"}),
		powerSupplyOutputVoltage: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "dc908_power_supply_output_voltage",
			Help: "Current output voltage on a power supply.",
		},
			[]string{"device"}),

		laserInputPower: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "dc908_laser_input_power_dbm",
			Help: "The input optical power of a physical channel in dBm.",
		},
			[]string{"device", "index"}),
		laserBiasCurrent: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "dc908_laser_bias_current_amepere",
			Help: "The current applied by the system to the transmit laser to achieve the output power.",
		},
			[]string{"device", "index"}),
		laserOutputPower: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "dc908_laser_output_power_dbm",
			Help: "The output optical power of a physical channel in dBm.",
		},
			[]string{"device", "index"}),
		laserChromaticDispersion: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "dc908_laser_chromatic_dispersion_ps_nm",
			Help: "Chromatic Dispersion of an optical channel in picoseconds / nanometer (ps/nm).",
		},
			[]string{"device"}),
		laserPolarizationDependetLoss: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "dc908_laser_polarization_dependent_loss_db",
			Help: "Polarization Dependent Loss of an optical channel in dB.",
		},
			[]string{"device"}),
		laserPolarizationModeDispersion: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "dc908_laser_polarization_mode_dispersion_ps",
			Help: "Polarization Mode Dispersion of an optical channel in picoseconds (ps).",
		},
			[]string{"device"}),
		// TODO: If we figure out what this really is, improve the help string.
		laserFrequencyOffset: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "dc908_laser_frequency_offset_hertz",
			Help: "Frequency offset from reference frequency.",
		},
			[]string{"device"}),
	}
	m.r.MustRegister(m.fanRPM)
	m.r.MustRegister(m.temperature)
	m.r.MustRegister(m.memoryUtilized)
	m.r.MustRegister(m.cpuUtilization)
	m.r.MustRegister(m.powerSupplyInputCurrent)
	m.r.MustRegister(m.powerSupplyInputVoltage)
	m.r.MustRegister(m.powerSupplyOutputCurrent)
	m.r.MustRegister(m.powerSupplyOutputVoltage)
	m.r.MustRegister(m.laserInputPower)
	m.r.MustRegister(m.laserBiasCurrent)
	m.r.MustRegister(m.laserOutputPower)
	m.r.MustRegister(m.laserChromaticDispersion)
	m.r.MustRegister(m.laserPolarizationDependetLoss)
	m.r.MustRegister(m.laserPolarizationModeDispersion)
	m.r.MustRegister(m.laserFrequencyOffset)
	return m
}

func (m *metricRegistry) PrometheusRegistry() *prometheus.Registry {
	return m.r
}

func (m *metricRegistry) Update(name string, json string) error {
	for _, mm := range matchers {
		match := mm.re.FindStringSubmatch(name)
		if match == nil {
			continue
		}
		if err := mm.cb(m, json, match[1:]); err != nil {
			return err
		}
	}
	return nil
}

func handleFan(m *metricRegistry, j string, groups []string) error {
	name := groups[0]
	val := struct {
		Speed uint64
	}{}

	if err := json.Unmarshal([]byte(j), &val); err != nil {
		return fmt.Errorf("failed to parse fan metric: %v", err)
	}
	log.V(2).Infof("New fan metric for %q: %+v", name, val)
	m.fanRPM.With(prometheus.Labels{"device": name}).Set(float64(val.Speed))
	return nil
}

func handleTemperature(m *metricRegistry, j string, groups []string) error {
	name := groups[0]
	val := struct {
		Temperature struct {
			Instant float64
		}
	}{}

	if err := json.Unmarshal([]byte(j), &val); err != nil {
		return fmt.Errorf("failed to parse temperature metric: %v", err)
	}
	log.V(2).Infof("New temperature metric for %q: %+v", name, val)
	m.temperature.With(prometheus.Labels{"device": name}).Set(float64(val.Temperature.Instant))
	return nil
}

func handleMemory(m *metricRegistry, j string, groups []string) error {
	name := groups[0]
	val := struct {
		Memory struct {
			Utilized *string
		}
	}{}

	if err := json.Unmarshal([]byte(j), &val); err != nil {
		return fmt.Errorf("failed to parse memory metric: %v", err)
	}
	if val.Memory.Utilized == nil {
		return nil
	}
	log.V(2).Infof("New memory metric for %q: %+v", name, val)
	memUtil, err := strconv.Atoi(*val.Memory.Utilized)
	if err != nil {
		return err
	}
	m.memoryUtilized.With(prometheus.Labels{"device": name}).Set(float64(memUtil))
	return nil
}

func handleCPUUtilization(m *metricRegistry, j string, groups []string) error {
	name := groups[0]
	val := struct {
		State struct {
			Instant float64
		}
	}{}

	if err := json.Unmarshal([]byte(j), &val); err != nil {
		return fmt.Errorf("failed to parse cpu utilization metric: %v", err)
	}
	log.V(2).Infof("New CPU utilization metric for %q: %+v", name, val)
	m.cpuUtilization.With(prometheus.Labels{"device": name}).Set(float64(val.State.Instant) / 100.0)
	return nil
}

func binaryFloat32ToFloat(b []byte) float64 {
	var pi float32
	buf := bytes.NewReader(b)
	err := binary.Read(buf, binary.BigEndian, &pi)
	if err != nil {
		return math.NaN()
	}
	return float64(pi)
}

func handlePowerSupply(m *metricRegistry, j string, groups []string) error {
	name := groups[0]
	valRaw := struct {
		InputCurrent  []byte `json:"input-current"`
		InputVoltage  []byte `json:"input-voltage"`
		OutputCurrent []byte `json:"output-current"`
		OutputVoltage []byte `json:"output-voltage"`
	}{}

	if err := json.Unmarshal([]byte(j), &valRaw); err != nil {
		return fmt.Errorf("failed to parse power supply metric: %v", err)
	}

	val := struct {
		InputCurrent  float64
		InputVoltage  float64
		OutputCurrent float64
		OutputVoltage float64
	}{
		InputCurrent:  binaryFloat32ToFloat(valRaw.InputCurrent),
		InputVoltage:  binaryFloat32ToFloat(valRaw.InputVoltage),
		OutputCurrent: binaryFloat32ToFloat(valRaw.OutputCurrent),
		OutputVoltage: binaryFloat32ToFloat(valRaw.OutputVoltage),
	}

	log.V(2).Infof("New power supply metric for %q: %+v", name, val)
	m.powerSupplyInputCurrent.With(prometheus.Labels{"device": name}).Set(val.InputCurrent)
	m.powerSupplyInputVoltage.With(prometheus.Labels{"device": name}).Set(val.InputVoltage)
	m.powerSupplyOutputCurrent.With(prometheus.Labels{"device": name}).Set(val.OutputCurrent)
	m.powerSupplyOutputVoltage.With(prometheus.Labels{"device": name}).Set(val.OutputVoltage)
	return nil
}

func handleGeneralLaser(m *metricRegistry, j string, groups []string) error {
	name := groups[0]
	labels := prometheus.Labels{"device": name, "index": ""}
	if len(groups) > 1 {
		index := groups[1]
		labels = prometheus.Labels{"device": name, "index": index}
	}
	val := struct {
		InputPower struct {
			Instant float64
		} `json:"input-power"`
		LaserBiasCurrent struct {
			Instant float64
		} `json:"laser-bias-current"`
		OutputPower struct {
			Instant float64
		} `json:"output-power"`
	}{}

	if err := json.Unmarshal([]byte(j), &val); err != nil {
		return fmt.Errorf("failed to parse general laser metric: %v", err)
	}
	log.V(2).Infof("New general laser metric for %v, %+v", labels, val)
	m.laserInputPower.With(labels).Set(val.InputPower.Instant)
	m.laserBiasCurrent.With(labels).Set(val.LaserBiasCurrent.Instant / 1000.0)
	m.laserOutputPower.With(labels).Set(val.OutputPower.Instant)
	return nil
}

func handleTerminalLaser(m *metricRegistry, j string, groups []string) error {
	name := groups[0]
	labels := prometheus.Labels{"device": name}
	val := struct {
		ChromaticDispersion struct {
			Instant float64
		} `json:"chromatic-dispersion"`
		PolarizationDependentLoss struct {
			Instant float64
		} `json:"polarization-dependent-loss"`
		PolarizationModeDispersion struct {
			Instant float64
		} `json:"polarization-mode-dispersion"`
		LaserFrequencyOffset string `json:"laser-freq-offset"`
	}{}

	if err := json.Unmarshal([]byte(j), &val); err != nil {
		return fmt.Errorf("failed to parse terminal laser metric: %v", err)
	}
	log.V(2).Infof("New terminal laser metric for %v, %+v", labels, val)

	freqOff, err := strconv.Atoi(val.LaserFrequencyOffset)
	if err != nil {
		return err
	}
	m.laserChromaticDispersion.With(labels).Set(val.ChromaticDispersion.Instant)
	m.laserPolarizationDependetLoss.With(labels).Set(val.PolarizationDependentLoss.Instant)
	m.laserPolarizationModeDispersion.With(labels).Set(val.PolarizationModeDispersion.Instant)
	m.laserFrequencyOffset.With(labels).Set(float64(freqOff) * 1000 * 1000)
	return nil
}
