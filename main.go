package main

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"
)

const ( // cmd
	WHOISHERE byte = 0x01
	IAMHERE   byte = 0x02
	GETSTATUS byte = 0x03
	STATUS    byte = 0x04
	SETSTATUS byte = 0x05
	TICK      byte = 0x06
)

const ( //dev_type
	SMARTHUB  byte = 0x01
	ENVSENSOR byte = 0x02
	SWITCH    byte = 0x03
	LAMP      byte = 0x04
	SOCKET    byte = 0x05
	CLOCK     byte = 0x06
)

const ALL VarUint = 0x3FFF

var (
	statusCode204 = errors.New("status code: 204")
	errStatusCode = errors.New("error status code (need 200 or 204)")
)

type VarUint uint64

type TimerСmdBody struct {
	Timestamp VarUint `json:"timestamp"`
}

type DeviceCmdBody struct {
	DevName  string     `json:"dev_name"`
	DevProps Serializer `json:"dev_props"`
}

type EnvSensorProps struct {
	Sensors  byte      `json:"sensors"`
	Triggers []Trigger `json:"triggers"`
}

type Trigger struct {
	Op    byte    `json:"op"`
	Value VarUint `json:"value"`
	Name  string  `json:"name"`
}

type SerStrings []string

type Payload struct {
	Src     VarUint    `json:"src"`
	Dst     VarUint    `json:"dst"`
	Serial  VarUint    `json:"serial"`
	DevType byte       `json:"dev_type"`
	Cmd     byte       `json:"cmd"`
	CmdBody Serializer `json:"cmd_body"`
}

type Packet struct {
	Length  byte    `json:"length"`
	Payload Payload `json:"payload"`
	Src8    byte    `json:"crc8"`
}

type EnvSensorStatusCmdBody struct {
	Values []VarUint //температура-влажность-освещенность-загрязнение воздуха
}

type Flag bool

type Hub struct {
	Name               string
	Url                string
	Address            VarUint
	DevicesWithAddress map[VarUint]Device
	DevicesWithName    map[string]Device
	Serial             VarUint
	wr                 waitRequests
	importantRequests  QueueRequests
	requests           QueueRequests
}

type Device struct {
	DevName string
	Address VarUint
	DevType byte
	Body    Serializer
}

func newDevice(name string, address VarUint, devType byte, body Serializer) Device {
	return Device{
		DevName: name,
		Address: address,
		DevType: devType,
		Body:    body,
	}
}

func CreateHub() (Hub, error) {
	args := os.Args
	if len(args) < 3 {
		return Hub{}, errors.New("arg(s) from cmd not finded")
	}
	strAdr, err := ConvertInt(args[2], 16, 10)
	if err != nil {
		return Hub{}, errors.New("can't read address of hub")
	}
	adr, _ := strconv.Atoi(strAdr)
	return Hub{
		Name:               "HUB00",
		Url:                args[1],
		Address:            VarUint(adr),
		DevicesWithAddress: make(map[VarUint]Device),
		DevicesWithName:    make(map[string]Device),
		Serial:             0,
		wr:                 CreateWaitRequests(),
		importantRequests:  newQueue(),
		requests:           newQueue(),
	}, nil
}

func ConvertInt(val string, base, toBase int) (string, error) {
	i, err := strconv.ParseInt(val, base, 64)
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(i, toBase), nil
}

func (h *Hub) Start() error {

	h.importantRequests.Push(createWhoIsHereRequest(h))
	h.wr.Add(CreateWaitRequest(2, 1e10))
	for {
		var requests []Payload
		if h.importantRequests.size > 0 {
			requests = []Payload{h.importantRequests.GetAndPop()}
		} else {
			requests = h.requests.GetAllAndClear()
		}
		strBase64 := serializePayloadsToBase64URLEncoded(requests)
		response, err := sendPOSTRequest(h.Url, strBase64)
		if err != nil {
			return err
		}
		for _, val := range response {
			h.processingPayload(val)
		}
		if h.importantRequests.size == 0 && h.requests.size == 0 {
			h.requests.Push(Payload{})
		}
	}
}

func createWhoIsHereRequest(h *Hub) Payload {
	h.Serial++
	return Payload{
		Src:     h.Address,
		Dst:     ALL,
		Serial:  h.Serial,
		DevType: SMARTHUB,
		Cmd:     WHOISHERE,
		CmdBody: DeviceCmdBody{
			DevName: h.Name,
		},
	}
}

func sendPOSTRequest(url string, base64String string) ([]Payload, error) {
	resp, err := http.Post(url, "application/base64", bytes.NewBufferString(base64String))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case 200:
		body, _ := io.ReadAll(resp.Body)
		payloads := decodeBase64ToPayloads(body)
		return payloads, nil
	case 204:
		return nil, statusCode204
	default:
		return nil, errStatusCode
	}
}

func main() {
	hub, err := CreateHub()
	if err != nil {
		os.Exit(99)
	}
	err = hub.Start()
	if err != nil {
		if errors.Is(err, statusCode204) {
			os.Exit(0)
		} else if errors.Is(err, errStatusCode) {
			os.Exit(99)
		} else {
			os.Exit(99)
		}
	}
}
