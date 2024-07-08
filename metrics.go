package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"regexp"

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
		{regexp.MustCompile(`/openconfig-platform:components/component\[name=([^,\]]+)\]/power-supply/state`), handlePowerSupply},
	}
)

type metricRegistry struct {
	r *prometheus.Registry

	fanRPM                   *prometheus.GaugeVec
	temperature              *prometheus.GaugeVec
	powerSupplyInputCurrent  *prometheus.GaugeVec
	powerSupplyInputVoltage  *prometheus.GaugeVec
	powerSupplyOutputCurrent *prometheus.GaugeVec
	powerSupplyOutputVoltage *prometheus.GaugeVec
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
	}
	m.r.MustRegister(m.fanRPM)
	m.r.MustRegister(m.temperature)
	m.r.MustRegister(m.powerSupplyInputCurrent)
	m.r.MustRegister(m.powerSupplyInputVoltage)
	m.r.MustRegister(m.powerSupplyOutputCurrent)
	m.r.MustRegister(m.powerSupplyOutputVoltage)
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
