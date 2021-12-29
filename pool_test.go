package go_pool

import (
	"os"
	"runtime"
	"strconv"
	"testing"
	"time"
)

func newTestPool(nRoutines int) *Pool {
	setCpuToBeUsed()
	poolVar = &Pool{
		WorkerChan: make(chan int, nRoutines),
		ReqChan:    make(chan *RequestChannel),
	}
	go poolVar.initWorker()
	return poolVar
}

func TestNumberOfCPUtoBeUsed(t *testing.T) {
	nCPU := runtime.NumCPU()
	err1 := os.Setenv("SET_CPU", strconv.Itoa(nCPU*4))
	if err1 != nil {
		t.Errorf("Error caused: %v", err1)
	}
	pool := newTestPool(1)
	defer pool.Close()
	if exp, act := nCPU, runtime.GOMAXPROCS(nCPU); exp != act {
		t.Errorf("Expected %v no of CPUs to be used, but got %v ", exp, act)
	}

	err2 := os.Setenv("SET_CPU", strconv.Itoa(nCPU/2))
	if err2 != nil {
		t.Errorf("Error caused: %v", err2)
	}
	setCpuToBeUsed()
	if exp, act := nCPU/2, runtime.GOMAXPROCS(nCPU/2); exp != act {
		t.Errorf("Expected %v no of CPUs to be used, but got %v ", exp, act)
	}
}

func TestProcessJob(t *testing.T) {
	err := os.Setenv("SET_CPU", strconv.Itoa(8))
	if err != nil {
		t.Errorf("Error caused: %v", err)
	}
	pool := newTestPool(10)
	defer pool.Close()

	for i := 0; i < 20; i++ {
		go func(itr int) {
			ret, err := pool.Process(itr, func(itr ReqPayload) RetPayload { return itr.(int) })
			if err != nil {
				t.Errorf("Error caused: %v", err)
			}
			if exp, act := itr, ret.(int); exp != act {
				t.Errorf("Wrong result: %v != %v", act, exp)
			}
		}(i)
	}

	//wait until processes are finished
	<-time.After(5 * time.Millisecond)
}

func TestProcessJobMultiFunction(t *testing.T) {
	err := os.Setenv("SET_CPU", strconv.Itoa(8))
	if err != nil {
		t.Errorf("Error caused: %v", err)
	}
	pool := newTestPool(10)
	defer pool.Close()

	go func() {
		ret, err := pool.Process(1, func(i ReqPayload) RetPayload { return i.(int) })
		if err != nil {
			t.Errorf("Error caused: %v", err)
		}
		if exp, act := 1, ret.(int); exp != act {
			t.Errorf("Wrong result: %v != %v", act, exp)
		}
	}()

	go func() {
		ret, err := pool.Process("Foo", func(i ReqPayload) RetPayload { return i.(string) })
		if err != nil {
			t.Errorf("Error caused: %v", err)
		}
		if exp, act := "Foo", ret.(string); exp != act {
			t.Errorf("Wrong result: %v != %v", act, exp)
		}
	}()

	//wait until processes are finished
	<-time.After(5 * time.Millisecond)
}

func TestProcessWithExpiryJob(t *testing.T) {
	err := os.Setenv("SET_CPU", strconv.Itoa(8))
	if err != nil {
		t.Errorf("Error caused: %v", err)
	}
	pool := newTestPool(10)
	defer pool.Close()

	for i := 0; i < 10; i++ {
		ret, err := pool.ProcessWithExpiry(i, time.Millisecond, func(i ReqPayload) RetPayload { return i.(int) })
		if err != nil {
			t.Errorf("Error caused: %v", err)
		}
		if exp, act := i, ret.(int); exp != act {
			t.Errorf("Wrong result: %v != %v", act, exp)
		}
	}
}

func TestPayloadTimedout(t *testing.T) {
	err := os.Setenv("SET_CPU", strconv.Itoa(8))
	if err != nil {
		t.Errorf("Error caused: %v", err)
	}
	pool := newTestPool(1)
	defer pool.Close()

	_, act := pool.ProcessWithExpiry(1, time.Millisecond, func(i ReqPayload) RetPayload {
		val := i.(int)
		<-time.After(2 * time.Millisecond)
		return val
	})
	if exp := ErrJobTimedOut; exp != act {
		t.Errorf("Wrong error returned: %v != %v", act, exp)
	}
}

func TestPoolSizeAdjustment(t *testing.T) {
	err := os.Setenv("SET_CPU", strconv.Itoa(8))
	if err != nil {
		t.Errorf("Error caused: %v", err)
	}
	pool := Initialize(10)
	if exp, act := 10, cap(pool.WorkerChan); exp != act {
		t.Errorf("Wrong size of pool: %v != %v", act, exp)
	}

	//Testng of pool close
	pool.Close()
	_, reqOk := <-pool.ReqChan
	if reqOk {
		t.Errorf("Pool should be closed")
	}
}
