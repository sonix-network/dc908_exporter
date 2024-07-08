package main

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"google.golang.org/protobuf/encoding/prototext"
)

func TestFanMetrics(t *testing.T) {
	var tests = []struct {
		fn string
		em string
	}{
		{"testdata/fan.textpb", `
# HELP dc908_fan_rpm Current fan speed in RPM.
# TYPE dc908_fan_rpm gauge
dc908_fan_rpm{device="FAN-1-33"} 4500
`},
		{"testdata/psu.textpb", `
# HELP dc908_power_supply_input_current_ampere Current input current on a power supply.
# TYPE dc908_power_supply_input_current_ampere gauge
dc908_power_supply_input_current_ampere{device="PSU-1-22"} 0.4620000123977661
# HELP dc908_power_supply_input_voltage Current input current on a power supply.
# TYPE dc908_power_supply_input_voltage gauge
dc908_power_supply_input_voltage{device="PSU-1-22"} 229
# HELP dc908_power_supply_output_current_ampere Current output current on a power supply.
# TYPE dc908_power_supply_output_current_ampere gauge
dc908_power_supply_output_current_ampere{device="PSU-1-22"} 1.5299999713897705
# HELP dc908_power_supply_output_voltage Current output voltage on a power supply.
# TYPE dc908_power_supply_output_voltage gauge
dc908_power_supply_output_voltage{device="PSU-1-22"} 53
# HELP dc908_temperature_celsius Current temperature of components.
# TYPE dc908_temperature_celsius gauge
dc908_temperature_celsius{device="PSU-1-22"} 30
`},
		{"testdata/mcu.textpb", `
# HELP dc908_temperature_celsius Current temperature of components.
# TYPE dc908_temperature_celsius gauge
dc908_temperature_celsius{device="MCU-1-41"} 34.7
# HELP dc908_cpu_utilization_ratio Ratio (0.0 - 1.0) of CPU utilization.
# TYPE dc908_cpu_utilization_ratio gauge
dc908_cpu_utilization_ratio{device="MCU-1-41"} 0.16
# HELP dc908_memory_utilized_bytes The number of bytes of memory currently in use by processes running on the component, not considering reserved memory that is not available for use.
# TYPE dc908_memory_utilized_bytes gauge
dc908_memory_utilized_bytes{device="MCU-1-41"} 9.08222464e+08
`},
		{"testdata/optics.textpb", `
# HELP dc908_laser_bias_current_amepere The current applied by the system to the transmit laser to achieve the output power.
# TYPE dc908_laser_bias_current_amepere gauge
dc908_laser_bias_current_amepere{device="TRANSCEIVER-1-1-C1",index=""} 0
dc908_laser_bias_current_amepere{device="TRANSCEIVER-1-1-C1",index="1"} 55.5
dc908_laser_bias_current_amepere{device="TRANSCEIVER-1-1-C1",index="2"} 55
dc908_laser_bias_current_amepere{device="TRANSCEIVER-1-1-C1",index="3"} 55
dc908_laser_bias_current_amepere{device="TRANSCEIVER-1-1-C1",index="4"} 55.5
dc908_laser_bias_current_amepere{device="TRANSCEIVER-1-1-C2",index=""} 0
dc908_laser_bias_current_amepere{device="TRANSCEIVER-1-1-C2",index="1"} 0
dc908_laser_bias_current_amepere{device="TRANSCEIVER-1-1-C2",index="2"} 0
dc908_laser_bias_current_amepere{device="TRANSCEIVER-1-1-C2",index="3"} 0
dc908_laser_bias_current_amepere{device="TRANSCEIVER-1-1-C2",index="4"} 0
dc908_laser_bias_current_amepere{device="TRANSCEIVER-1-1-C3",index=""} 0
dc908_laser_bias_current_amepere{device="TRANSCEIVER-1-1-C3",index="1"} 0
dc908_laser_bias_current_amepere{device="TRANSCEIVER-1-1-C3",index="2"} 0
dc908_laser_bias_current_amepere{device="TRANSCEIVER-1-1-C3",index="3"} 0
dc908_laser_bias_current_amepere{device="TRANSCEIVER-1-1-C3",index="4"} 0
dc908_laser_bias_current_amepere{device="TRANSCEIVER-1-1-C4",index=""} 0
dc908_laser_bias_current_amepere{device="TRANSCEIVER-1-1-C4",index="1"} 0
dc908_laser_bias_current_amepere{device="TRANSCEIVER-1-1-C4",index="2"} 0
dc908_laser_bias_current_amepere{device="TRANSCEIVER-1-1-C4",index="3"} 0
dc908_laser_bias_current_amepere{device="TRANSCEIVER-1-1-C4",index="4"} 0
dc908_laser_bias_current_amepere{device="TRANSCEIVER-1-1-L1",index=""} 188.7
dc908_laser_bias_current_amepere{device="TRANSCEIVER-1-1-L2",index=""} 226.3
# HELP dc908_laser_input_power_dbm The input optical power of a physical channel in dBm.
# TYPE dc908_laser_input_power_dbm gauge
dc908_laser_input_power_dbm{device="TRANSCEIVER-1-1-C1",index=""} 5.7
dc908_laser_input_power_dbm{device="TRANSCEIVER-1-1-C1",index="1"} -0.5
dc908_laser_input_power_dbm{device="TRANSCEIVER-1-1-C1",index="2"} -0.1
dc908_laser_input_power_dbm{device="TRANSCEIVER-1-1-C1",index="3"} -0.3
dc908_laser_input_power_dbm{device="TRANSCEIVER-1-1-C1",index="4"} -0.6
dc908_laser_input_power_dbm{device="TRANSCEIVER-1-1-C2",index=""} -60
dc908_laser_input_power_dbm{device="TRANSCEIVER-1-1-C2",index="1"} -60
dc908_laser_input_power_dbm{device="TRANSCEIVER-1-1-C2",index="2"} -60
dc908_laser_input_power_dbm{device="TRANSCEIVER-1-1-C2",index="3"} -60
dc908_laser_input_power_dbm{device="TRANSCEIVER-1-1-C2",index="4"} -60
dc908_laser_input_power_dbm{device="TRANSCEIVER-1-1-C3",index=""} -60
dc908_laser_input_power_dbm{device="TRANSCEIVER-1-1-C3",index="1"} -60
dc908_laser_input_power_dbm{device="TRANSCEIVER-1-1-C3",index="2"} -60
dc908_laser_input_power_dbm{device="TRANSCEIVER-1-1-C3",index="3"} -60
dc908_laser_input_power_dbm{device="TRANSCEIVER-1-1-C3",index="4"} -60
dc908_laser_input_power_dbm{device="TRANSCEIVER-1-1-C4",index=""} -60
dc908_laser_input_power_dbm{device="TRANSCEIVER-1-1-C4",index="1"} -60
dc908_laser_input_power_dbm{device="TRANSCEIVER-1-1-C4",index="2"} -60
dc908_laser_input_power_dbm{device="TRANSCEIVER-1-1-C4",index="3"} -60
dc908_laser_input_power_dbm{device="TRANSCEIVER-1-1-C4",index="4"} -60
dc908_laser_input_power_dbm{device="TRANSCEIVER-1-1-L1",index=""} -14.3
dc908_laser_input_power_dbm{device="TRANSCEIVER-1-1-L2",index=""} -14.5
# HELP dc908_laser_output_power_dbm The output optical power of a physical channel in dBm.
# TYPE dc908_laser_output_power_dbm gauge
dc908_laser_output_power_dbm{device="TRANSCEIVER-1-1-C1",index=""} 6.3
dc908_laser_output_power_dbm{device="TRANSCEIVER-1-1-C1",index="1"} 0
dc908_laser_output_power_dbm{device="TRANSCEIVER-1-1-C1",index="2"} 0
dc908_laser_output_power_dbm{device="TRANSCEIVER-1-1-C1",index="3"} 0.4
dc908_laser_output_power_dbm{device="TRANSCEIVER-1-1-C1",index="4"} 0.5
dc908_laser_output_power_dbm{device="TRANSCEIVER-1-1-C2",index=""} -60
dc908_laser_output_power_dbm{device="TRANSCEIVER-1-1-C2",index="1"} -60
dc908_laser_output_power_dbm{device="TRANSCEIVER-1-1-C2",index="2"} -60
dc908_laser_output_power_dbm{device="TRANSCEIVER-1-1-C2",index="3"} -60
dc908_laser_output_power_dbm{device="TRANSCEIVER-1-1-C2",index="4"} -60
dc908_laser_output_power_dbm{device="TRANSCEIVER-1-1-C3",index=""} -60
dc908_laser_output_power_dbm{device="TRANSCEIVER-1-1-C3",index="1"} -60
dc908_laser_output_power_dbm{device="TRANSCEIVER-1-1-C3",index="2"} -60
dc908_laser_output_power_dbm{device="TRANSCEIVER-1-1-C3",index="3"} -60
dc908_laser_output_power_dbm{device="TRANSCEIVER-1-1-C3",index="4"} -60
dc908_laser_output_power_dbm{device="TRANSCEIVER-1-1-C4",index=""} -60
dc908_laser_output_power_dbm{device="TRANSCEIVER-1-1-C4",index="1"} -60
dc908_laser_output_power_dbm{device="TRANSCEIVER-1-1-C4",index="2"} -60
dc908_laser_output_power_dbm{device="TRANSCEIVER-1-1-C4",index="3"} -60
dc908_laser_output_power_dbm{device="TRANSCEIVER-1-1-C4",index="4"} -60
dc908_laser_output_power_dbm{device="TRANSCEIVER-1-1-L1",index=""} 0.5
dc908_laser_output_power_dbm{device="TRANSCEIVER-1-1-L2",index=""} 0.5
# HELP dc908_temperature_celsius Current temperature of components.
# TYPE dc908_temperature_celsius gauge
dc908_temperature_celsius{device="LINECARD-1-1"} 60.2
dc908_temperature_celsius{device="TRANSCEIVER-1-1-C1"} 40.7
dc908_temperature_celsius{device="TRANSCEIVER-1-1-C2"} 38.7
dc908_temperature_celsius{device="TRANSCEIVER-1-1-C3"} 37
dc908_temperature_celsius{device="TRANSCEIVER-1-1-C4"} 35.6
dc908_temperature_celsius{device="TRANSCEIVER-1-1-L1"} 50
dc908_temperature_celsius{device="TRANSCEIVER-1-1-L2"} 50
`},
	}

	for _, tt := range tests {
		t.Run(tt.fn, func(t *testing.T) {
			d, err := os.ReadFile(tt.fn)
			if err != nil {
				panic(err)
			}

			m := &gnmi.SubscribeResponse{}
			err = prototext.Unmarshal(d, m)
			if err != nil {
				panic(err)
			}

			mr := NewMetricRegistry()

			WalkNotification(m.GetUpdate(), func(name string, _ *time.Time, j string) {
				if err := mr.Update(name, j); err != nil {
					t.Errorf("metric update: err %v", err)
				}
			}, nil)

			if err := testutil.GatherAndCompare(mr.PrometheusRegistry(), strings.NewReader(tt.em)); err != nil {
				t.Errorf("metric compare: err %v", err)
			}
		})
	}
}
