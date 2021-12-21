package go_pool

// routines wraps go routines and worker and manages the lifecycle of both of them.
type routines struct {
	worker        Worker
	interruptChan chan struct{}

	reqChan    chan<- routineRequest
	closeChan  chan struct{}
	closedChan chan struct{}
}

// routineRequest is a struct which holds the payload for worker until process is completed
type routineRequest struct {
	reqChan chan<- interface{}

	retChan       <-chan interface{}
	interruptFunc func()
}

func newRoutineWrapper(reqChan chan routineRequest, payload Worker) *routines {
	r := routines{
		worker:        payload,
		interruptChan: make(chan struct{}),
		reqChan:       reqChan,
		closeChan:     make(chan struct{}),
		closedChan:    make(chan struct{}),
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
		r.worker.Terminate()
		close(retChan)
		close(r.closedChan)
	}()

	for {
		select {
		case r.reqChan <- routineRequest{
			reqChan:       reqChan,
			retChan:       retChan,
			interruptFunc: r.interrupt,
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

func (r *routines) interrupt() {
	close(r.interruptChan)
	r.worker.Interrupt()
}
