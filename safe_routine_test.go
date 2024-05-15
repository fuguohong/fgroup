package fgroup

import (
	"context"
	"errors"
	"github.com/stretchr/testify/suite"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

type SafeRoutineTest struct {
	suite.Suite
}

func TestSafeRoutine(t *testing.T) {
	suite.Run(t, new(SafeRoutineTest))
}

func (t *SafeRoutineTest) TestGo() {
	t.Run("no panic", func() {
		ch := make(chan int)
		defer close(ch)
		Go(nil, func() {
			ch <- 1
		})
		x1 := <-ch
		t.Equal(x1, 1)
	})

	t.Run("with panic", func() {
		var i int32 = 0
		wg := sync.WaitGroup{}
		wg.Add(3)
		Go(nil, func() {
			defer wg.Done()
			atomic.AddInt32(&i, 1)
		})
		Go(nil, func() {
			defer wg.Done()
			atomic.AddInt32(&i, 2)
			panic("func 2 panic")
		})
		Go(nil, func() {
			defer wg.Done()
			panic("func 3 panic")
			atomic.AddInt32(&i, 3)
		})
		wg.Wait()
		t.Equal(i, int32(3))
	})

	t.Run("grandchild routine", func() {
		ch := make(chan int)
		Go(nil, func() {
			Go(nil, func() {
				ch <- 1
				panic("inner routine panic")
			})
		})

		x := <-ch
		t.Equal(x, 1)
	})
}

func (t *SafeRoutineTest) TestGoLog() {
	ch := make(chan bool)
	defer func() {
		Log = nil
		close(ch)
	}()
	var ctx context.Context
	var msg interface{}
	var stack string
	Log = func(c context.Context, i interface{}, s string) {
		msg = i
		ctx = c
		stack = s
		ch <- true
	}

	t.Run("Go log with context", func() {
		paramCtx := context.Background()
		Go(paramCtx, func() {
			makePanic()
		})
		<-ch

		stacks := strings.Split(stack, "\n")
		t.True(len(stacks) > 1)
		t.True(strings.Contains(stacks[1], "makePanic"))
		t.Equal(ctx, paramCtx)
		t.Equal(msg, panicErr)
	})

	t.Run("Go log without context", func() {
		Go(nil, func() {
			makePanic()
		})
		<-ch

		stacks := strings.Split(stack, "\n")
		t.True(len(stacks) > 1)
		t.True(strings.Contains(stacks[1], "makePanic"))
		t.Equal(ctx, nil)
		t.Equal(msg, panicErr)
	})
}

var panicErr = errors.New("fgh test panic")

func makePanic() {
	panic(panicErr)
}
