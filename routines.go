package go_pool

type routines struct {
	worker        Worker
	interruptChan chan struct{}

	reqChan    chan<- routineRequest
	closeChan  chan struct{}
	closedChan chan struct{}
}

type routineRequest struct {
	reqChan chan<- interface{}

	retChan <-chan interface{}
}

func newRoutineWrapper(reqChan chan routineRequest, payload Worker) *routines {
	r := routines{
		worker:        payload,
		interruptChan: make(chan struct{}),
		reqChan:       reqChan,
	}

	go r.run()

	return &r
}

func (r *routines) stop() {
	close(r.closeChan)
}

func (r *routines) join() {
	<-r.closedChan
}

func (r *routines) run() {
	reqChan, retChan := make(chan interface{}), make(chan interface{})
	defer func() {
		close(retChan)
		close(r.closedChan)
	}()

	for {
		select {
		case r.reqChan <- routineRequest{
			reqChan: reqChan,
			retChan: retChan,
		}:
			select {
			case payload := <-reqChan:
				result := r.worker.Process(payload)
				select {
				case retChan <- result:
				case <-r.interruptChan:
					r.interruptChan = make(chan struct{})
				}
			case <-r.interruptChan:
				r.interruptChan = make(chan struct{})
			}
		case <-r.closeChan:
			return
		}
	}
}
