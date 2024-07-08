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
# HELP dc908_memory_utilized_bytes The number of bytes of memory currently in use by processes running on the component, not considering reserved memory that is not available for use.
# TYPE dc908_memory_utilized_bytes gauge
dc908_memory_utilized_bytes{device="MCU-1-41"} 9.08222464e+08
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
