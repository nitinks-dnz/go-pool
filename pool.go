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
	WorkerFun   func() Worker
	WorkerCount int

	ReqChan chan interface{}
	RetChan chan interface{}

	wg sync.WaitGroup
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
			WorkerFun:   wFun,
			WorkerCount: 0,

			ReqChan: make(chan interface{}, n),
			RetChan: make(chan interface{}, n),
		}
		//go poolVar.initWorkers(n)
	})
	return poolVar
}

func (p *Pool) addWorker() {
	if p.WorkerCount < 10 {
		p.WorkerCount = p.WorkerCount + 1
		p.wg.Add(1)
		go p.worker(&p.wg, p.WorkerFun())
		p.wg.Wait()
	}
	return
}

func (p *Pool) worker(wg *sync.WaitGroup, worker Worker) {
	for req := range p.ReqChan {
		p.RetChan <- worker.Process(req)
	}
	p.WorkerCount = p.WorkerCount - 1
	wg.Done()
}

func (p *Pool) Process(reqPayload interface{}) (interface{}, error) {
	p.ReqChan <- reqPayload

	go p.addWorker()

	retPayload, retOk := <-p.RetChan
	if !retOk {
		return nil, ErrWorkerClosed
	}

	return retPayload, nil
}

func (p *Pool) ProcessWithExpiry(reqPayload interface{}, timeout time.Duration) (interface{}, error) {

	tout := time.NewTimer(timeout)

	var retPayload interface{}
	var open bool

	select {
	case p.ReqChan <- reqPayload:
	case <-tout.C:
		return nil, ErrJobTimedOut
	}

	go p.addWorker()

	select {
	case retPayload, open = <-p.RetChan:
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
	close(p.ReqChan)
	close(p.RetChan)
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
