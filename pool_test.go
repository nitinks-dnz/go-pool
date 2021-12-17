package go_pool

import "testing"

func TestPoolSizeAdjustment(t *testing.T) {
	pool := Initialize(10, func(interface{}) interface{} { return "foo" })
	if exp, act := 10, len(pool.routines); exp != act {
		t.Errorf("Wrong size of pool: %v != %v", act, exp)
	}

	if exp, act := 10, pool.GetPoolSize(); exp != act {
		t.Errorf("Wrong size of pool: %v != %v", act, exp)
	}
}

func TestFuncJob(t *testing.T) {
	pool := Initialize(10, func(in interface{}) interface{} {
		intVal := in.(int)
		return intVal * 2
	})
	defer pool.Close()

	for i := 0; i < 10; i++ {
		ret := pool.Process(10)
		if exp, act := 20, ret.(int); exp != act {
			t.Errorf("Wrong result: %v != %v", act, exp)
		}
	}
}
