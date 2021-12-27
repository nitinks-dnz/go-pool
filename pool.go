package go_pool

import (
	"errors"
	"runtime"
	"sync"
	"time"
)

var (
	ErrWorkerClosed = errors.New("no routine is active")
	ErrJobTimedOut  = errors.New("job request timed expired")

	once    sync.Once
	poolVar *Pool
)

// Pool is a struct which contains the list of routines
type Pool struct {
	wFun func() Worker

	reqChan chan interface{}
	retChan chan interface{}
}

// Worker is an interface representing the routine agent
type Worker interface {
	Process(interface{}) interface{}
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

func New(n int, wFun func() Worker) *Pool {
	once.Do(func() {
		poolVar = &Pool{
			wFun:    wFun,
			reqChan: make(chan interface{}, n),
			retChan: make(chan interface{}, n),
		}
		go poolVar.initWorkers(n)
	})
	return poolVar
}

func (p *Pool) initWorkers(n int) {
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go p.worker(&wg, p.wFun())
	}
	wg.Wait()
}

func (p *Pool) worker(wg *sync.WaitGroup, worker Worker) {
	for req := range p.reqChan {
		p.retChan <- worker.Process(req)
	}
	wg.Done()
}

func (p *Pool) Process(reqPayload interface{}) interface{} {

	p.reqChan <- reqPayload

	retPayload, retOk := <-p.retChan
	if !retOk {
		panic(ErrWorkerClosed)
	}

	return retPayload
}

func (p *Pool) ProcessWithExpiry(reqPayload interface{}, timeout time.Duration) (interface{}, error) {

	tout := time.NewTimer(timeout)

	var retPayload interface{}
	var open bool

	select {
	case p.reqChan <- reqPayload:
	case <-tout.C:
		return nil, ErrJobTimedOut
	}

	select {
	case retPayload, open = <-p.retChan:
		if !open {
			return nil, ErrWorkerClosed
		}
	case <-tout.C:
		return nil, ErrJobTimedOut
	}

	tout.Stop()
	return retPayload, nil
}

func (p *Pool) Close() {
	close(p.reqChan)
	close(p.retChan)
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
