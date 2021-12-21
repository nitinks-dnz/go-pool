package go_pool

import (
	"runtime"
	"testing"
	"time"
)

func TestPoolSizeAdjustment(t *testing.T) {
	pool := Initialize(8, 10, func(interface{}) interface{} { return "foo" })
	if exp, act := 10, len(pool.routines); exp != act {
		t.Errorf("Wrong size of pool: %v != %v", act, exp)
	}

	//Testing of Set and Get pool size
	pool.SetPoolSize(0)
	if exp, act := 0, pool.GetPoolSize(); exp != act {
		t.Errorf("Wrong size of pool: %v != %v", act, exp)
	}

	pool.SetPoolSize(9)
	if exp, act := 9, pool.GetPoolSize(); exp != act {
		t.Errorf("Wrong size of pool: %v != %v", act, exp)
	}

	//Testng of pool close
	pool.Close()
	if exp, act := 0, pool.GetPoolSize(); exp != act {
		t.Errorf("Wrong size of pool: %v != %v", act, exp)
	}

}

func TestProcessJob(t *testing.T) {
	pool := Initialize(8, 10, func(f interface{}) interface{} { return f.(int) })
	defer pool.Close()

	for i := 0; i < 10; i++ {
		ret := pool.Process(i)
		if exp, act := i, ret.(int); exp != act {
			t.Errorf("Wrong result: %v != %v", act, exp)
		}
	}
}

func TestProcessWithExpiryJob(t *testing.T) {
	pool := Initialize(8, 10, func(f interface{}) interface{} { return f.(int) })
	defer pool.Close()

	for i := 0; i < 10; i++ {
		ret, err := pool.ProcessWithExpiry(i, time.Duration(time.Millisecond))
		if err != nil {
			t.Errorf("Error caused: %v", err)
		}
		if exp, act := i, ret.(int); exp != act {
			t.Errorf("Wrong result: %v != %v", act, exp)
		}
	}
}

func TestPayloadTimedout(t *testing.T) {
	pool := Initialize(8, 1, func(f interface{}) interface{} {
		val := f.(int)
		<-time.After(2 * time.Millisecond)
		return val
	})
	defer pool.Close()

	_, act := pool.ProcessWithExpiry(1, time.Duration(time.Millisecond))
	if exp := ErrJobTimedOut; exp != act {
		t.Errorf("Wrong error returned: %v != %v", act, exp)
	}
}

func TestProcessAfterPoolClose(t *testing.T) {
	pool := Initialize(8, 1, func(f interface{}) interface{} { return f.(int) })
	pool.Close()

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Process after Stop() did not panic")
		}
	}()

	pool.Process(1)
}
func TestQueueLength(t *testing.T) {
	pool := Initialize(8, 1, func(f interface{}) interface{} {
		val := f.(int)
		<-time.After(2 * time.Millisecond)
		return val
	})
	defer pool.Close()

	befQ := pool.QueueLength()
	if exp, act := 0, befQ; exp != act {
		t.Errorf("Expected Queue length: %v, but got: %v", exp, act)
	}

	go func() {
		pool.Process(1)
	}()
	time.Sleep(time.Millisecond)
	if exp, act := 1, pool.QueueLength(); exp != act {
		t.Errorf("Expected Queue length: %v, but got: %v", exp, act)
	}
}

func TestNumberOfCPUtoBeUsed(t *testing.T) {
	nCPU := runtime.NumCPU()
	pool := Initialize(nCPU*4, 1, func(interface{}) interface{} { return "foo" })
	defer pool.Close()
	if exp, act := nCPU, runtime.GOMAXPROCS(nCPU); exp != act {
		t.Errorf("Expected %v no of CPUs to be used, but got %v ", exp, act)
	}

	setCpuToBeUsed(nCPU / 2)
	if exp, act := nCPU/2, runtime.GOMAXPROCS(nCPU/2); exp != act {
		t.Errorf("Expected %v no of CPUs to be used, but got %v ", exp, act)
	}
}
