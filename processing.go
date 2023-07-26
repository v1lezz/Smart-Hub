package main

func (h *Hub) processingPayload(payload Payload) {
	switch payload.Cmd {
	case WHOISHERE:
		h.Serial++
		h.importantRequests.Push(Payload{
			Src:     h.Address,
			Dst:     payload.Src,
			Serial:  h.Serial,
			DevType: SMARTHUB,
			Cmd:     IAMHERE,
			CmdBody: DeviceCmdBody{
				DevName: h.Name,
			},
		})
		h.Serial++
		h.requests.Push(Payload{
			Src:     h.Address,
			Dst:     payload.Src,
			Serial:  h.Serial,
			DevType: payload.DevType,
			Cmd:     GETSTATUS,
			CmdBody: DeviceCmdBody{
				DevName: h.Name,
			},
		})
		cmdBody, ok := payload.CmdBody.(DeviceCmdBody)
		if !ok {
			return
		}
		h.SaveDevice(cmdBody.DevName, payload.Src, payload.DevType, cmdBody.DevProps)
		h.wr.Add(CreateWaitRequest(4, payload.Src))
	case IAMHERE:
		if h.wr.size > 0 && h.wr.requests[0].isWhoIsAre() {
			cmdBody, ok := payload.CmdBody.(DeviceCmdBody)
			if !ok {
				return
			}
			h.SaveDevice(cmdBody.DevName, payload.Src, payload.DevType, cmdBody.DevProps)
			h.Serial++
			h.requests.Push(Payload{
				Src:     h.Address,
				Dst:     payload.Src,
				Serial:  h.Serial,
				DevType: payload.DevType,
				Cmd:     GETSTATUS,
			})
			h.wr.Add(CreateWaitRequest(4, payload.Src))
		}
	case STATUS:
		switch payload.DevType {
		case ENVSENSOR:
			device, ok := h.DevicesWithAddress[payload.Src]
			if ok {
				cmdBody, ok := payload.CmdBody.(EnvSensorStatusCmdBody)
				if !ok {
					return
				}
				i := byte(0)
				typeSensor := byte(0)
				props, ok := device.Body.(EnvSensorProps)
				if !ok {
					return
				}
				for b := byte(1); b <= 8; b *= 2 {
					if h.processingStatusSensor(props, b, cmdBody.Values[i], typeSensor) {
						i++
					}
					typeSensor++
				}
			}
		case SWITCH:
			device, ok := h.DevicesWithAddress[payload.Src]
			if ok {
				cmdBody, ok := payload.CmdBody.(Flag)
				if !ok {
					return
				}
				devices, ok := device.Body.(SerStrings)
				if !ok {
					return
				}
				h.processingStatusSwitch(devices, cmdBody)
			}
		case LAMP, SOCKET:
			h.SaveStatus(payload.Src, payload.CmdBody)
		}
		h.DeleteFromWR(payload.Cmd, payload.Src)
	case TICK:
		if t, ok := payload.CmdBody.(TimerСmdBody); ok {
			addresses := h.wr.CheckWaitRequests(t.Timestamp)
			h.DeleteDevices(addresses)
		} else {
			return
		}
	}
}

func (h *Hub) processingStatusSensor(props EnvSensorProps, b byte, value VarUint, xType byte) bool {
	if props.Sensors&b == 0 {
		return false
	}

	for _, trigger := range props.Triggers {
		typeSensor := (trigger.Op & 12) / 4
		if typeSensor != xType {
			continue
		}
		border := trigger.Value
		comp := (trigger.Op & 2) / 2
		dev, ok := h.DevicesWithName[trigger.Name]
		if !ok {
			continue
		}

		setStatus := Flag(false)
		if trigger.Op&1 == 1 {
			setStatus = true
		}
		if (comp == 1 && value > border) || (comp == 0 && value < border) {
			h.Serial++
			h.requests.Push(Payload{
				Src:     h.Address,
				Dst:     dev.Address,
				Serial:  h.Serial,
				DevType: dev.DevType,
				Cmd:     SETSTATUS,
				CmdBody: setStatus,
			})
			h.wr.Add(CreateWaitRequest(STATUS, dev.Address))
		}
	}
	return true
}

func (h *Hub) processingStatusSwitch(props SerStrings, setStatus Flag) {
	for _, devName := range props {
		device, ok := h.DevicesWithName[devName]
		if !ok {
			continue
		}
		h.Serial++
		h.requests.Push(Payload{
			Src:     h.Address,
			Dst:     device.Address,
			Serial:  h.Serial,
			DevType: device.DevType,
			Cmd:     SETSTATUS,
			CmdBody: setStatus,
		})
		h.wr.Add(CreateWaitRequest(STATUS, device.Address))

	}
}

func (h *Hub) SaveStatus(address VarUint, status Serializer) {
	device, ok := h.DevicesWithAddress[address]
	if !ok {
		return
	}
	device.Body = status
	h.DevicesWithAddress[address] = device
	h.DevicesWithName[device.DevName] = device
}

func (h *Hub) DeleteFromWR(cmd byte, address VarUint) {
	for i := 0; i < len(h.wr.requests); i++ {
		if cmd == h.wr.requests[i].Cmd && address != h.wr.requests[i].Address {
			h.wr.requests = append(h.wr.requests[:i], h.wr.requests[i+1:]...)
			h.wr.size--
			break
		}
	}
}

func (h *Hub) SaveDevice(name string, address VarUint, devType byte, body Serializer) {
	dev := newDevice(name, address, devType, body)
	if val, ok := h.DevicesWithAddress[address]; ok {
		delete(h.DevicesWithName, val.DevName)
	}
	h.DevicesWithName[name] = dev
	h.DevicesWithAddress[address] = dev
}

func (h *Hub) DeleteDevices(addresses []VarUint) {
	for _, val := range addresses {
		name := h.DevicesWithAddress[val].DevName
		delete(h.DevicesWithAddress, val)
		delete(h.DevicesWithName, name)
	}
}
