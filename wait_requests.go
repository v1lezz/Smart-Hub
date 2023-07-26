package main

type waitRequest struct {
	Cmd          byte
	Address      VarUint
	wasFirstTick bool
	firstTick    VarUint
}

func CreateWaitRequest(cmd byte, address VarUint) waitRequest {
	return waitRequest{Cmd: cmd, Address: address, wasFirstTick: false, firstTick: 0}
}

func (w *waitRequest) isWhoIsAre() bool {
	return w.Cmd == 2
}

type waitRequests struct {
	requests       []waitRequest
	cntWithNotNull int
	size           int
}

func CreateWaitRequests() waitRequests {
	return waitRequests{make([]waitRequest, 0, 1), 0, 0}
}

func (w *waitRequests) Add(x waitRequest) {
	w.requests = append(w.requests, x)
	w.size++
}

func (w *waitRequests) CheckWaitRequests(timestamp VarUint) []VarUint {
	ans := make([]VarUint, 0, 1)
	cnt := 0
	for i := 0; i < w.cntWithNotNull; i++ {
		if w.requests[i].wasFirstTick && timestamp-w.requests[i].firstTick >= 300 {
			ans = append(ans, w.requests[i].Address)
			w.size--
			w.cntWithNotNull--
			cnt++
		} else {
			break
		}
	}
	w.requests = w.requests[cnt:]
	for i := w.cntWithNotNull; i < w.size; i++ {
		w.requests[i].firstTick = timestamp
		w.requests[i].wasFirstTick = true
		w.cntWithNotNull++
	}
	return ans
}
