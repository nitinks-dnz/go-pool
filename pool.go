package go_pool

import (
	"errors"
	"os"
	"runtime"
	"strconv"
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
	//WorkerFun   func() Worker
	WorkerCount int

	ReqChan chan RequestChannel
	RetChan chan interface{}

	wg sync.WaitGroup
}

// Initialize is the public function which creates the pool
func Initialize(nRoutines int) *Pool {
	once.Do(func() {
		setCpuToBeUsed()
		poolVar = &Pool{
			WorkerCount: 0,

			ReqChan: make(chan RequestChannel, nRoutines),
			RetChan: make(chan interface{}, nRoutines),
		}
	})
	return poolVar
}

func (p *Pool) addWorker() {
	if p.WorkerCount < 10 {
		p.WorkerCount = p.WorkerCount + 1
		p.wg.Add(1)
		go p.worker(&p.wg)
		p.wg.Wait()
	}
	return
}

func (p *Pool) worker(wg *sync.WaitGroup) {
	for req := range p.ReqChan {
		p.RetChan <- req.wFun.Process(req.input)
	}
	p.WorkerCount = p.WorkerCount - 1
	wg.Done()
}

func (p *Pool) Process(input ReqPayload, f func(payload ReqPayload) RetPayload) (RetPayload, error) {
	p.ReqChan <- RequestChannel{
		input: input,
		wFun: func() WorkerFun {
			return &initWorker{
				processor: f,
			}
		}(),
	}

	go p.addWorker()

	retPayload, retOk := <-p.RetChan
	if !retOk {
		return nil, ErrWorkerClosed
	}

	return retPayload, nil
}

func (p *Pool) ProcessWithExpiry(reqPayload interface{}, timeout time.Duration, f func(payload ReqPayload) RetPayload) (RetPayload, error) {

	tout := time.NewTimer(timeout)

	var retPayload RetPayload
	var open bool

	select {
	case p.ReqChan <- RequestChannel{
		input: reqPayload,
		wFun: func() WorkerFun {
			return &initWorker{
				processor: f,
			}
		}(),
	}:
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

func setCpuToBeUsed() {
	n, err := strconv.Atoi(os.Getenv("SET_CPU"))
	if err == nil && n != 0 && n < runtime.NumCPU() {
		runtime.GOMAXPROCS(n)
	}
}
