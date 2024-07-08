# dc908\_exporter

[![Go Report Card](https://goreportcard.com/badge/github.com/sonix-network/dc908_exporter)](https://goreportcard.com/report/github.com/sonix-network/dc908_exporter)

Simple collector designed to take the gNMI dialout stream from a **single** [Huawei OptiXtrans DC908](https://e.huawei.com/en/products/optical-transmission/dc908)
and export the real-time updates as native Prometheus metrics.

The Huawei OptiXtrans DC908 is a optical-electrical Wavelength Division Multiplexing (WDM) transmission device designed for Data Center Interconnect (DCI).

Caveats as of right now:

 - Only a single dialout device is supported per exporter
 - Metrics do not become stale, in the absence of gNMI updates the last value is exported indefinitely

## Usage

Configuration on the DC908:

```
# Enable metrics collection
system-view
grpc
protocol no-tls
pm
start statistics-task sdh_15minute_default now
stop statistics-task sdh_15minute_default at 2087-12-31 00:00:00
return

# Define what to send and to where
system-view
telemetry
sensor-group my-sensors
sensor-path /openconfig-platform:components/component/cpu/openconfig-platform-cpu:utilization
sensor-path /openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/physical-channels/channel/state
sensor-path /openconfig-platform:components/component/openconfig-terminal-device:optical-channel/state
sensor-path /openconfig-platform:components/component/openconfig-platform-transceiver:transceiver/state
sensor-path /openconfig-platform:components/component/power-supply/state
sensor-path /openconfig-interfaces:interfaces/interface/state/counters
sensor-path /openconfig-platform:components/component/fan/state
sensor-path /openconfig-platform:components/component/state
destination-group exporter
destination ipv4-address 1.2.3.4 port 8888
subscription my-subscription
add sensor-group my-sensors sample-interval 5000 suppress-redundant false heartbeat-interval 10
add destination-group exporter
protocol grpc encoding json_ietf
return
```

Prometheus configuration like any other standard exporter.

## Example metrics

```
# HELP dc908_fan_rpm Current fan speed in RPM.
# TYPE dc908_fan_rpm gauge
dc908_fan_rpm{device="FAN-1-31"} 4250
dc908_fan_rpm{device="FAN-1-32"} 4300
dc908_fan_rpm{device="FAN-1-33"} 4200
# HELP dc908_power_supply_input_current_ampere Current input current on a power supply.
# TYPE dc908_power_supply_input_current_ampere gauge
dc908_power_supply_input_current_ampere{device="PSU-1-21"} 0.3970000147819519
dc908_power_supply_input_current_ampere{device="PSU-1-22"} 0.4620000123977661
# HELP dc908_power_supply_input_voltage Current input current on a power supply.
# TYPE dc908_power_supply_input_voltage gauge
dc908_power_supply_input_voltage{device="PSU-1-21"} 229
dc908_power_supply_input_voltage{device="PSU-1-22"} 229
# HELP dc908_power_supply_output_current_ampere Current output current on a power supply.
# TYPE dc908_power_supply_output_current_ampere gauge
dc908_power_supply_output_current_ampere{device="PSU-1-21"} 1.2599999904632568
dc908_power_supply_output_current_ampere{device="PSU-1-22"} 1.5299999713897705
# HELP dc908_power_supply_output_voltage Current output voltage on a power supply.
# TYPE dc908_power_supply_output_voltage gauge
dc908_power_supply_output_voltage{device="PSU-1-21"} 53
dc908_power_supply_output_voltage{device="PSU-1-22"} 53
# HELP dc908_temperature_celsius Current temperature of components.
# TYPE dc908_temperature_celsius gauge
dc908_temperature_celsius{device="LINECARD-1-1"} 60.1
dc908_temperature_celsius{device="MCU-1-41"} 34.3
dc908_temperature_celsius{device="PANEL-1-40"} 25.8
dc908_temperature_celsius{device="PSU-1-21"} 29
dc908_temperature_celsius{device="PSU-1-22"} 30
dc908_temperature_celsius{device="TRANSCEIVER-1-1-C1"} 40.7
dc908_temperature_celsius{device="TRANSCEIVER-1-1-C2"} 38.7
dc908_temperature_celsius{device="TRANSCEIVER-1-1-C3"} 36.8
dc908_temperature_celsius{device="TRANSCEIVER-1-1-C4"} 35.2
dc908_temperature_celsius{device="TRANSCEIVER-1-1-L1"} 50
dc908_temperature_celsius{device="TRANSCEIVER-1-1-L2"} 50
```
