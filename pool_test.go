package go_pool

import (
	"runtime"
	"testing"
	"time"
)

func newTestPool(nCpus int, nRoutines int, f func(interface{}) interface{}) *Pool {
	setCpuToBeUsed(nCpus)
	return newWorker(nRoutines, func() Worker {
		return &initWorker{
			processor: f,
		}
	})
}

func newWorker(n int, payload func() Worker) *Pool {
	poolVar = &Pool{
		WorkerFun: payload,
		ReqChan:   make(chan interface{}, n),
		RetChan:   make(chan interface{}, n),
	}

	//go poolVar.initWorkers(n)

	return poolVar
}

func TestNumberOfCPUtoBeUsed(t *testing.T) {
	nCPU := runtime.NumCPU()
	pool := newTestPool(nCPU*4, 1, func(interface{}) interface{} { return "foo" })
	defer pool.Close()
	if exp, act := nCPU, runtime.GOMAXPROCS(nCPU); exp != act {
		t.Errorf("Expected %v no of CPUs to be used, but got %v ", exp, act)
	}

	setCpuToBeUsed(nCPU / 2)
	if exp, act := nCPU/2, runtime.GOMAXPROCS(nCPU/2); exp != act {
		t.Errorf("Expected %v no of CPUs to be used, but got %v ", exp, act)
	}
}

func TestProcessJob(t *testing.T) {
	pool := newTestPool(8, 10, func(f interface{}) interface{} { return f.(int) })
	defer pool.Close()

	for i := 0; i < 20; i++ {
		ret, err := pool.Process(i)
		if err != nil {
			t.Errorf("Error caused: %v", err)
		}
		if exp, act := i, ret.(int); exp != act {
			t.Errorf("Wrong result: %v != %v", act, exp)
		}
	}
}

func TestProcessWithExpiryJob(t *testing.T) {
	pool := newTestPool(8, 10, func(f interface{}) interface{} { return f.(int) })
	defer pool.Close()

	for i := 0; i < 10; i++ {
		ret, err := pool.ProcessWithExpiry(i, time.Millisecond)
		if err != nil {
			t.Errorf("Error caused: %v", err)
		}
		if exp, act := i, ret.(int); exp != act {
			t.Errorf("Wrong result: %v != %v", act, exp)
		}
	}
}

func TestPayloadTimedout(t *testing.T) {
	pool := newTestPool(8, 1, func(f interface{}) interface{} {
		val := f.(int)
		<-time.After(2 * time.Millisecond)
		return val
	})
	defer pool.Close()

	_, act := pool.ProcessWithExpiry(1, time.Millisecond)
	if exp := ErrJobTimedOut; exp != act {
		t.Errorf("Wrong error returned: %v != %v", act, exp)
	}
}

func TestPoolSizeAdjustment(t *testing.T) {
	pool := Initialize(8, 10, func(interface{}) interface{} { return "Foo" })
	if exp, act := 10, cap(pool.ReqChan); exp != act {
		t.Errorf("Wrong size of pool: %v != %v", act, exp)
	}

	//Testng of pool close
	pool.Close()
	_, reqOk := <-pool.ReqChan
	if reqOk {
		t.Errorf("Pool should be closed")
	}
}

func TestSingletonInitialization(t *testing.T) {
	pool := Initialize(8, 10, func(f interface{}) interface{} { return f.(int) })

	_, reqOk := <-pool.ReqChan
	if reqOk {
		t.Errorf("Pool should be closed")
	}
}
