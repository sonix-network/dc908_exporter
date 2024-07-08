package main

import (
	"os"
	"testing"
	"time"

	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/encoding/prototext"
)

func TestParseUpdates(t *testing.T) {
	var tests = []struct {
		fn    string
		ts    time.Time
		names []string
	}{
		{"testdata/fan.textpb", time.Date(2024, 7, 7, 19, 59, 10, 0, time.UTC), []string{
			"/openconfig-platform:components/component[name=FAN-1-33]/fan/state",
		}},
		{"testdata/optics.textpb", time.Date(2024, 7, 7, 19, 59, 6, 0, time.UTC), []string{
			"/openconfig-platform:components/component[name=TRANSCEIVER-1-1-C1]/openconfig-platform-transceiver:transceiver/physical-channels/channel[index=1]/state",
			"/openconfig-platform:components/component[name=TRANSCEIVER-1-1-C1]/openconfig-platform-transceiver:transceiver/physical-channels/channel[index=2]/state",
			"/openconfig-platform:components/component[name=TRANSCEIVER-1-1-C1]/openconfig-platform-transceiver:transceiver/physical-channels/channel[index=3]/state",
			"/openconfig-platform:components/component[name=TRANSCEIVER-1-1-C1]/openconfig-platform-transceiver:transceiver/physical-channels/channel[index=4]/state",
			"/openconfig-platform:components/component[name=TRANSCEIVER-1-1-C2]/openconfig-platform-transceiver:transceiver/physical-channels/channel[index=1]/state",
			"/openconfig-platform:components/component[name=TRANSCEIVER-1-1-C2]/openconfig-platform-transceiver:transceiver/physical-channels/channel[index=2]/state",
			"/openconfig-platform:components/component[name=TRANSCEIVER-1-1-C2]/openconfig-platform-transceiver:transceiver/physical-channels/channel[index=3]/state",
			"/openconfig-platform:components/component[name=TRANSCEIVER-1-1-C2]/openconfig-platform-transceiver:transceiver/physical-channels/channel[index=4]/state",
			"/openconfig-platform:components/component[name=TRANSCEIVER-1-1-C3]/openconfig-platform-transceiver:transceiver/physical-channels/channel[index=1]/state",
			"/openconfig-platform:components/component[name=TRANSCEIVER-1-1-C3]/openconfig-platform-transceiver:transceiver/physical-channels/channel[index=2]/state",
			"/openconfig-platform:components/component[name=TRANSCEIVER-1-1-C3]/openconfig-platform-transceiver:transceiver/physical-channels/channel[index=3]/state",
			"/openconfig-platform:components/component[name=TRANSCEIVER-1-1-C3]/openconfig-platform-transceiver:transceiver/physical-channels/channel[index=4]/state",
			"/openconfig-platform:components/component[name=TRANSCEIVER-1-1-C4]/openconfig-platform-transceiver:transceiver/physical-channels/channel[index=1]/state",
			"/openconfig-platform:components/component[name=TRANSCEIVER-1-1-C4]/openconfig-platform-transceiver:transceiver/physical-channels/channel[index=2]/state",
			"/openconfig-platform:components/component[name=TRANSCEIVER-1-1-C4]/openconfig-platform-transceiver:transceiver/physical-channels/channel[index=3]/state",
			"/openconfig-platform:components/component[name=TRANSCEIVER-1-1-C4]/openconfig-platform-transceiver:transceiver/physical-channels/channel[index=4]/state",
			"/openconfig-platform:components/component[name=OCH-1-1-L1]/openconfig-terminal-device:optical-channel/state",
			"/openconfig-platform:components/component[name=OCH-1-1-L2]/openconfig-terminal-device:optical-channel/state",
			"/openconfig-platform:components/component[name=TRANSCEIVER-1-1-L1]/openconfig-platform-transceiver:transceiver/state",
			"/openconfig-platform:components/component[name=TRANSCEIVER-1-1-L2]/openconfig-platform-transceiver:transceiver/state",
			"/openconfig-platform:components/component[name=TRANSCEIVER-1-1-C1]/openconfig-platform-transceiver:transceiver/state",
			"/openconfig-platform:components/component[name=TRANSCEIVER-1-1-C2]/openconfig-platform-transceiver:transceiver/state",
			"/openconfig-platform:components/component[name=TRANSCEIVER-1-1-C3]/openconfig-platform-transceiver:transceiver/state",
			"/openconfig-platform:components/component[name=TRANSCEIVER-1-1-C4]/openconfig-platform-transceiver:transceiver/state",
			"/openconfig-platform:components/component[name=TRANSCEIVER-1-1-L1]/state",
			"/openconfig-platform:components/component[name=TRANSCEIVER-1-1-L2]/state",
			"/openconfig-platform:components/component[name=LINECARD-1-1]/state",
			"/openconfig-platform:components/component[name=TRANSCEIVER-1-1-C1]/state",
			"/openconfig-platform:components/component[name=TRANSCEIVER-1-1-C2]/state",
			"/openconfig-platform:components/component[name=TRANSCEIVER-1-1-C3]/state",
			"/openconfig-platform:components/component[name=TRANSCEIVER-1-1-C4]/state",
		}},
	}

	for _, tt := range tests {
		t.Run(tt.fn, func(t *testing.T) {
			assert := assert.New(t)
			d, err := os.ReadFile(tt.fn)
			if err != nil {
				panic(err)
			}

			m := &gnmi.SubscribeResponse{}
			err = prototext.Unmarshal(d, m)
			if err != nil {
				panic(err)
			}

			var got []string
			WalkNotification(m.GetUpdate(), func(name string, ts *time.Time, _ string) {
				assert.Equal(*ts, tt.ts)
				got = append(got, name)
			}, nil)

			assert.Equal(got, tt.names)
		})
	}
}
