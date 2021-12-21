package go_pool

import (
	"errors"
	"runtime"
	"sync"
	"time"
)

var (
	ErrPoolNotRunning = errors.New("the pool is closed")
	ErrJobNotFunc     = errors.New("init worker not given a func()")
	ErrWorkerClosed   = errors.New("no routine is active")
	ErrJobTimedOut    = errors.New("job request timed expired")
)

// Pool is a struct which contains the list of routines
type Pool struct {
	reqQueue int

	payload  func() Worker
	routines []*routines
	reqChan  chan routineRequest

	mut sync.Mutex
}

// Worker is an interface representing the routine agent
type Worker interface {
	Process(interface{}) interface{}

	BlockUntilReady()
	Interrupt()
	Terminate()
}

// Initialize is the public function which creates the pool
func Initialize(nCpus int, nRoutines int, f func(interface{}) interface{}) *Pool {
	setCpuToBeUsed(nCpus)
	return New(nRoutines, func() Worker {
		return &initWorker{
			processor: f,
		}
	})
}

func New(n int, payload func() Worker) *Pool {
	p := &Pool{
		payload: payload,
		reqChan: make(chan routineRequest),
	}
	p.SetPoolSize(n)

	return p
}

func (p *Pool) Process(reqPayload interface{}) interface{} {
	p.reqQueue = p.reqQueue + 1

	request, reqOk := <-p.reqChan
	if !reqOk {
		panic(ErrPoolNotRunning)
	}

	request.reqChan <- reqPayload

	retPayload, retOk := <-request.retChan
	if !retOk {
		panic(ErrWorkerClosed)
	}

	p.reqQueue = p.reqQueue - 1
	return retPayload
}

func (p *Pool) ProcessWithExpiry(reqPayload interface{}, timeout time.Duration) (interface{}, error) {
	p.reqQueue = p.reqQueue + 1

	tout := time.NewTimer(timeout)
	var request routineRequest
	var retPayload interface{}
	var open bool

	select {
	case request, open = <-p.reqChan:
		if !open {
			return nil, ErrPoolNotRunning
		}
	case <-tout.C:
		p.reqQueue = p.reqQueue - 1
		return nil, ErrJobTimedOut
	}

	select {
	case request.reqChan <- reqPayload:
	case <-tout.C:
		request.interruptFunc()
		p.reqQueue = p.reqQueue - 1
		return nil, ErrJobTimedOut
	}

	select {
	case retPayload, open = <-request.retChan:
		if !open {
			return nil, ErrWorkerClosed
		}
	case <-tout.C:
		request.interruptFunc()
		p.reqQueue = p.reqQueue - 1
		return nil, ErrJobTimedOut
	}

	tout.Stop()
	p.reqQueue = p.reqQueue - 1
	return retPayload, nil
}

func (p *Pool) SetPoolSize(n int) {
	p.mut.Lock()
	defer p.mut.Unlock()

	lWorkers := len(p.routines)
	if lWorkers == n {
		return
	}

	for i := lWorkers; i < n; i++ {
		p.routines = append(p.routines, newRoutineWrapper(p.reqChan, p.payload()))
	}

	for i := n; i < lWorkers; i++ {
		p.routines[i].stop()
	}

	for i := n; i < lWorkers; i++ {
		p.routines[i].join()
		p.routines[i] = nil
	}

	p.routines = p.routines[:n]
}

func (p *Pool) GetPoolSize() int {
	p.mut.Lock()
	defer p.mut.Unlock()

	return len(p.routines)
}

func (p *Pool) Close() {
	p.SetPoolSize(0)
	close(p.reqChan)
}

func (p *Pool) QueueLength() int {
	return p.reqQueue
}

func setCpuToBeUsed(n int) {
	if n < runtime.NumCPU() {
		runtime.GOMAXPROCS(n)
	}
}

type initWorker struct {
	processor func(interface{}) interface{}
}

func (w *initWorker) Process(payload interface{}) interface{} {
	return w.processor(payload)
}
func (w *initWorker) BlockUntilReady() {}
func (w *initWorker) Interrupt()       {}
func (w *initWorker) Terminate()       {}
