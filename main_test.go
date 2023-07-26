package main

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEnvSensor(t *testing.T) {
	hub := Hub{
		Name:               "HUB00",
		Url:                "dsafd",
		Address:            VarUint(1),
		DevicesWithAddress: make(map[VarUint]Device),
		DevicesWithName:    make(map[string]Device),
		Serial:             0,
		wr:                 CreateWaitRequests(),
		importantRequests:  newQueue(),
		requests:           newQueue(),
	}
	hub.wr.Add(CreateWaitRequest(2, 1e10))
	payloads := decodeBase64ToPayloads([]byte("OAL_fwQCAghTRU5TT1IwMQ8EDGQGT1RIRVIxD7AJBk9USEVSMgCsjQYGT1RIRVIzCAAGT1RIRVI09w"))
	hub.processingPayload(Payload{
		Src:     2,
		Dst:     16383,
		Serial:  4,
		DevType: 2,
		Cmd:     2,
		CmdBody: DeviceCmdBody{
			DevName: "SENSOR01",
			DevProps: EnvSensorProps{
				Sensors: 13,
				Triggers: []Trigger{
					Trigger{
						Op:    12,
						Value: 100,
						Name:  "OTHER1",
					},
					Trigger{
						Op:    15,
						Value: 1200,
						Name:  "OTHER2",
					},
					Trigger{
						Op:    0,
						Value: 100012,
						Name:  "OTHER3",
					},
					Trigger{
						Op:    8,
						Value: 0,
						Name:  "OTHER4",
					},
				},
			},
		}})
	hub.SaveDevice("OTHER1", 100, 4, Flag(false))
	hub.SaveDevice("OTHER2", 101, 4, Flag(false))
	hub.SaveDevice("OTHER3", 102, 4, Flag(false))
	hub.SaveDevice("OTHER4", 103, 4, Flag(false))
	payloads = decodeBase64ToPayloads([]byte("EQIBBgIEBKUB4AfUjgaMjfILrw"))
	hub.processingPayload(payloads[0])
	buf := new(bytes.Buffer)
	serializePayload(buf, hub.requests.data[2])
	ser := buf.Bytes()
	assert.Equal(t, ser[len(ser)-1], byte(0x01))
}
