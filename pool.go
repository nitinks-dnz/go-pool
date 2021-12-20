package go_pool

import (
	"errors"
	"runtime"
	"sync"
	"time"
)

var (
	ErrPoolNotRunning = errors.New("the pool is not running")
	ErrJobNotFunc     = errors.New("generic worker not given a func()")
	ErrWorkerClosed   = errors.New("worker was closed")
	ErrJobTimedOut    = errors.New("job request timed out")
)

type Pool struct {
	reqQueue int

	payload  func() Worker
	routines []*routines
	reqChan  chan routineRequest

	mut sync.Mutex
}

type Worker interface {
	// Process will synchronously perform a job and return the result.
	Process(interface{}) interface{}

	BlockUntilReady()
	Interrupt()
	Terminate()
}

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

func (p *Pool) Process(payload interface{}) interface{} {
	p.reqQueue = p.reqQueue + 1

	request, reqOk := <-p.reqChan
	if !reqOk {
		panic(ErrPoolNotRunning)
	}

	request.reqChan <- payload

	payload, retOk := <-request.retChan
	if !retOk {
		panic(ErrWorkerClosed)
	}

	p.reqQueue = p.reqQueue - 1
	return payload
}

func (p *Pool) ProcessWithExpiry(payload interface{}, timeout time.Duration) (interface{}, error) {
	p.reqQueue = p.reqQueue + 1

	tout := time.NewTimer(timeout)
	var request routineRequest
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
	case request.reqChan <- payload:
	case <-tout.C:
		request.interruptFunc()
		p.reqQueue = p.reqQueue - 1
		return nil, ErrJobTimedOut
	}

	select {
	case payload, open = <-request.retChan:
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
	return payload, nil
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

func (p *Pool) QueueLength() int {
	return p.reqQueue
}

func (p *Pool) SetPoolSize(n int) {
	p.mut.Lock()
	defer p.mut.Unlock()

	lWorkers := len(p.routines)
	if lWorkers == n {
		return
	}

	// Add extra workers if N > len(workers)
	for i := lWorkers; i < n; i++ {
		p.routines = append(p.routines, newRoutineWrapper(p.reqChan, p.payload()))
	}

	// Asynchronously stop all workers > N
	for i := n; i < lWorkers; i++ {
		p.routines[i].stop()
	}

	// Synchronously wait for all workers > N to stop
	for i := n; i < lWorkers; i++ {
		p.routines[i].join()
		p.routines[i] = nil
	}

	// Remove stopped workers from slice
	p.routines = p.routines[:n]
}

// GetSize returns the current size of the pool.
func (p *Pool) GetPoolSize() int {
	p.mut.Lock()
	defer p.mut.Unlock()

	return len(p.routines)
}

// Close will terminate all workers and close the job channel of this Pool.
func (p *Pool) Close() {
	p.SetPoolSize(0)
	close(p.reqChan)
}

func setCpuToBeUsed(n int) {
	if n < runtime.NumCPU() {
		runtime.GOMAXPROCS(n)
	}
}
