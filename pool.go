package go_pool

import (
	"errors"
	"log"
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
	WorkerChan chan int
	ReqChan    chan *RequestChannel
}

// Initialize is the public function which creates the pool
func Initialize(nRoutines int) *Pool {
	once.Do(func() {
		setCpuToBeUsed()
		poolVar = &Pool{
			WorkerChan: make(chan int, nRoutines),
			ReqChan:    make(chan *RequestChannel),
		}
		go poolVar.initWorker()
	})
	return poolVar
}

func (p *Pool) initWorker() {
	for {
		req, ok := <-p.ReqChan
		if !ok {
			return
		} else {
			go p.worker(req)
			p.WorkerChan <- len(p.WorkerChan) + 1
		}
	}
}

func (p *Pool) worker(req *RequestChannel) {
	defer func() {
		i := <-p.WorkerChan
		req.wId = i
	}()

	req.retChan <- req.wFun.Process(req.input)
}

func (p *Pool) Process(input ReqPayload, f func(payload ReqPayload) RetPayload) (RetPayload, error) {
	req := RequestChannel{
		input: input,
		wFun: func() WorkerFun {
			return &initWorker{
				processor: f,
			}
		}(),
		retChan: make(chan RetPayload),
	}
	p.ReqChan <- &req

	retPayload, retOk := <-req.retChan
	if !retOk {
		return nil, ErrWorkerClosed
	}

	log.Println("Processed job -> ", input, " by worker", req.wId)
	return retPayload, nil
}

func (p *Pool) ProcessWithExpiry(reqPayload interface{}, timeout time.Duration, f func(payload ReqPayload) RetPayload) (RetPayload, error) {
	req := RequestChannel{
		input: reqPayload,
		wFun: func() WorkerFun {
			return &initWorker{
				processor: f,
			}
		}(),
		retChan: make(chan RetPayload),
	}

	tout := time.NewTimer(timeout)

	var retPayload RetPayload
	var open bool

	select {
	case p.ReqChan <- &req:
	case <-tout.C:
		return nil, ErrJobTimedOut
	}

	select {
	case retPayload, open = <-req.retChan:
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
	close(p.WorkerChan)
}

func setCpuToBeUsed() {
	//toDo use Flag
	n, err := strconv.Atoi(os.Getenv("SET_CPU"))
	if err == nil && n != 0 && n < runtime.NumCPU() {
		runtime.GOMAXPROCS(n)
	}
}
