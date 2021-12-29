package go_pool

type ReqPayload interface{}
type RetPayload interface{}

// WorkerFun is an interface representing the routine agent
type WorkerFun interface {
	Process(payload ReqPayload) RetPayload
}

type initWorker struct {
	processor func(payload ReqPayload) RetPayload
}

func (w *initWorker) Process(payload ReqPayload) RetPayload {
	return w.processor(payload)
}

type RequestChannel struct {
	wFun  WorkerFun
	input ReqPayload
}
