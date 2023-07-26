package main

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"io"
)

var RawURLEncoding = base64.URLEncoding.WithPadding(-1)

func serializeString(w io.Writer, str string) {
	b := []byte(str)
	w.Write(append([]byte{byte(len(b))}, b...))
}

type Serializer interface {
	Serialize(io.Writer)
}

func (value VarUint) Serialize(w io.Writer) {
	for value >= 0x80 {
		binary.Write(w, binary.LittleEndian, byte(value)|0x80)
		value >>= 7
	}
	binary.Write(w, binary.LittleEndian, byte(value))
}

func (t TimerСmdBody) Serialize(w io.Writer) {
	t.Timestamp.Serialize(w)
}

func (d DeviceCmdBody) Serialize(w io.Writer) {
	serializeString(w, d.DevName)
	if d.DevProps != nil {
		d.DevProps.Serialize(w)
	}
}

func (s SerStrings) Serialize(w io.Writer) {
	for _, val := range s {
		serializeString(w, val)
	}
}

func (s EnvSensorProps) Serialize(w io.Writer) {
	binary.Write(w, binary.LittleEndian, s.Sensors)
	for _, val := range s.Triggers {
		binary.Write(w, binary.LittleEndian, val.Op)
		val.Value.Serialize(w)
		serializeString(w, val.Name)
	}

}

func (s EnvSensorStatusCmdBody) Serialize(w io.Writer) {
	for _, val := range s.Values {
		val.Serialize(w)
	}
}

func (f Flag) Serialize(w io.Writer) {
	if f {
		binary.Write(w, binary.LittleEndian, byte(1))
	} else {
		binary.Write(w, binary.LittleEndian, byte(0))
	}
}

func serializeCmdBody(w io.Writer, cmd byte, cmdBody Serializer) {
	switch cmd {
	case WHOISHERE, IAMHERE, GETSTATUS, STATUS, SETSTATUS, TICK:
		cmdBody.Serialize(w)
	default:
	}
}

func serializePayload(w io.Writer, payload Payload) {
	payload.Src.Serialize(w)
	payload.Dst.Serialize(w)
	payload.Serial.Serialize(w)
	binary.Write(w, binary.LittleEndian, payload.DevType)
	binary.Write(w, binary.LittleEndian, payload.Cmd)
	if payload.CmdBody != nil {
		serializeCmdBody(w, payload.Cmd, payload.CmdBody)
	}
}

func calculateCRC8(payload []byte) byte {
	СRC8_TABLE := [256]byte{0, 29, 58, 39, 116, 105, 78, 83, 232, 245, 210, 207, 156, 129, 166, 187,
		205, 208, 247, 234, 185, 164, 131, 158, 37, 56, 31, 2, 81, 76, 107, 118,
		135, 154, 189, 160, 243, 238, 201, 212, 111, 114, 85, 72, 27, 6, 33, 60,
		74, 87, 112, 109, 62, 35, 4, 25, 162, 191, 152, 133, 214, 203, 236, 241,
		19, 14, 41, 52, 103, 122, 93, 64, 251, 230, 193, 220, 143, 146, 181, 168,
		222, 195, 228, 249, 170, 183, 144, 141, 54, 43, 12, 17, 66, 95, 120, 101,
		148, 137, 174, 179, 224, 253, 218, 199, 124, 97, 70, 91, 8, 21, 50, 47,
		89, 68, 99, 126, 45, 48, 23, 10, 177, 172, 139, 150, 197, 216, 255, 226,
		38, 59, 28, 1, 82, 79, 104, 117, 206, 211, 244, 233, 186, 167, 128, 157,
		235, 246, 209, 204, 159, 130, 165, 184, 3, 30, 57, 36, 119, 106, 77, 80,
		161, 188, 155, 134, 213, 200, 239, 242, 73, 84, 115, 110, 61, 32, 7, 26,
		108, 113, 86, 75, 24, 5, 34, 63, 132, 153, 190, 163, 240, 237, 202, 215,
		53, 40, 15, 18, 65, 92, 123, 102, 221, 192, 231, 250, 169, 180, 147, 142,
		248, 229, 194, 223, 140, 145, 182, 171, 16, 13, 42, 55, 100, 121, 94, 67,
		178, 175, 136, 149, 198, 219, 252, 225, 90, 71, 96, 125, 46, 51, 20, 9,
		127, 98, 69, 88, 11, 22, 49, 44, 151, 138, 173, 176, 227, 254, 217, 196}
	var crc byte
	for _, val := range payload {
		crc = СRC8_TABLE[(crc^val)&0xff]
	}
	return crc
}

func serializePayloadsToBase64URLEncoded(payloads []Payload) string {
	if len(payloads) == 1 && payloads[0].Cmd == 0 {
		return ""
	}
	var ans []byte
	for _, payload := range payloads {
		buf := new(bytes.Buffer)
		serializePayload(buf, payload)
		ans = append(ans, byte(buf.Len()))
		ans = append(ans, buf.Bytes()...)
		ans = append(ans, calculateCRC8(buf.Bytes()))
	}

	base64Str := RawURLEncoding.EncodeToString(ans)
	return base64Str
}

func checkSrc(payload []byte, src8 byte) bool {
	return calculateCRC8(payload) == src8
}

func deserializeVarUint(bin []byte, startIndex, lastIndex int) (VarUint, int) {
	var result, shift VarUint
	i := startIndex
	for i < lastIndex {
		b := bin[i]
		value := VarUint(b & 0x7F)
		result |= value << shift
		shift += 7
		i++
		if b&0x80 == 0 {
			break
		}
	}
	return result, i
}

func readString(bin []byte, startIndex, lastIndex int) (string, int, error) {
	length := int(bin[startIndex])
	startIndex++
	if startIndex+length > lastIndex {
		return "", 0, errors.New("error read string")
	}
	return string(bin[startIndex : startIndex+length]), startIndex + length, nil
}

func deserializeEnvSensorProps(bin []byte, startIndex, lastIndex int) Serializer {
	sensors := bin[startIndex]
	startIndex++
	lenghtTriggers := bin[startIndex]
	startIndex++
	triggers := make([]Trigger, 0, lenghtTriggers)
	var err error
	for startIndex < lastIndex {
		trigger := Trigger{Op: bin[startIndex]}
		startIndex++
		trigger.Value, startIndex = deserializeVarUint(bin, startIndex, lastIndex)
		trigger.Name, startIndex, err = readString(bin, startIndex, lastIndex)
		if err != nil {
			return nil
		}
		triggers = append(triggers, trigger)
	}
	return EnvSensorProps{Sensors: sensors, Triggers: triggers}

}

func deserializeDevPropsForSwitch(bin []byte, startIndex, lastIndex int) Serializer {
	length := int(bin[startIndex])
	startIndex++
	names := make(SerStrings, 0, length)
	var name string
	var err error
	for i := 0; i < length && startIndex < lastIndex; i++ {
		name, startIndex, err = readString(bin, startIndex, lastIndex)
		if err != nil {
			return nil
		}
		names = append(names, name)
	}
	return names
}

func deserializeDevice(bin []byte, devType byte, startIndex, lastIndex int) Serializer {
	devName, startIndex, err := readString(bin, startIndex, lastIndex)
	if err != nil {
		return nil
	}
	device := DeviceCmdBody{DevName: devName}
	switch devType {
	case ENVSENSOR:
		device.DevProps = deserializeEnvSensorProps(bin, startIndex, lastIndex)
	case SWITCH:
		device.DevProps = deserializeDevPropsForSwitch(bin, startIndex, lastIndex)
	case LAMP, SOCKET, CLOCK:
	default:
		return nil
	}
	return device
}

func deserializeCmdBodyStatus(bin []byte, devType byte, startIndex, lastIndex int) Serializer {
	switch devType {
	case ENVSENSOR:
		var value VarUint
		length := int(bin[startIndex])
		startIndex++
		status := EnvSensorStatusCmdBody{make([]VarUint, 0, length)}
		for i := 0; i < length; i++ {
			value, startIndex = deserializeVarUint(bin, startIndex, lastIndex)
			status.Values = append(status.Values, value)
		}
		return status
	case SWITCH, LAMP, SOCKET:
		var flag Flag
		if bin[startIndex] == 0x01 {
			flag = true
		} else {
			flag = false
		}
		startIndex++
		return flag
	default:
		return nil
	}
}

func deserializeCmdBody(bin []byte, devType, cmd byte, startIndex, lastIndex int) (Serializer, error) {
	if startIndex == lastIndex {
		return nil, nil
	}
	switch cmd {
	case WHOISHERE, IAMHERE:
		return deserializeDevice(bin, devType, startIndex, lastIndex), nil
	case STATUS:
		return deserializeCmdBodyStatus(bin, devType, startIndex, lastIndex), nil
	case TICK:
		timestamp, _ := deserializeVarUint(bin, startIndex, lastIndex)
		return TimerСmdBody{Timestamp: timestamp}, nil
	default:
		return nil, errors.New("unknown cmd")
	}
}

func deserializePayload(bin []byte, startIndex, length int) (Payload, error) {
	lastIndex := startIndex + length
	if lastIndex >= len(bin) {
		return Payload{}, errors.New("last index out of range")
	}
	src, startIndex := deserializeVarUint(bin, startIndex, lastIndex)
	dst, startIndex := deserializeVarUint(bin, startIndex, lastIndex)
	serial, startIndex := deserializeVarUint(bin, startIndex, lastIndex)
	devType := bin[startIndex]
	startIndex++
	cmd := bin[startIndex]
	startIndex++
	cmdBody, err := deserializeCmdBody(bin, devType, cmd, startIndex, lastIndex)
	if err != nil {
		return Payload{}, err
	}
	return Payload{Src: src, Dst: dst, Serial: serial, DevType: devType, Cmd: cmd, CmdBody: cmdBody}, nil

}

func deserializeFromBinaryFormToPayloads(bin []byte) []Payload {
	i := 0
	payloads := make([]Payload, 0, 1)
	for i < len(bin) {
		length := int(bin[i])
		i++
		payload, err := deserializePayload(bin, i, length)
		if err != nil {
			i += length + 1
		} else {
			binPayload := bin[i : i+length]
			i += length
			src8 := bin[i]
			i++
			if checkSrc(binPayload, src8) {
				payloads = append(payloads, payload)
			}
		}
	}
	return payloads
}

func decodeBase64ToPayloads(response []byte) []Payload {
	binaryResponse, err := base64.RawURLEncoding.DecodeString(string(response))
	if err != nil {
		return nil
	}
	payloads := deserializeFromBinaryFormToPayloads(binaryResponse)
	return payloads
}
