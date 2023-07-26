package main

type QueueRequests struct {
	data []Payload
	size int
}

func newQueue() QueueRequests {
	return QueueRequests{data: make([]Payload, 0, 1), size: 0}
}

func (q *QueueRequests) Push(x Payload) {
	q.data = append(q.data, x)
	q.size++
}

func (q *QueueRequests) GetAndPop() Payload {
	ans := q.data[0]
	q.data = q.data[1:]
	q.size--
	return ans
}

func (q *QueueRequests) GetAllAndClear() []Payload {
	ans := q.data
	q.data = make([]Payload, 0, 1)
	q.size = 0
	return ans
}

func (q *QueueRequests) PushMoreOne(payloads []Payload) {
	q.data = append(q.data, payloads...)
	q.size += len(payloads)
}
