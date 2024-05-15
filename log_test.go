package fgroup

import (
	"context"
	"github.com/stretchr/testify/suite"
	"strings"
	"testing"
)

type LogTest struct {
	suite.Suite
}

func TestLog(t *testing.T) {
	suite.Run(t, new(LogTest))
}

func (t *LogTest) TestGetStack() {
	stack := getStack(0)
	stacks := strings.Split(stack, "\n")
	t.True(len(stacks) > 1)
	t.True(strings.Contains(stacks[0], "TestGetStack"))

	t.Run("多层嵌套堆栈", func() {
		var s string
		fn1 := func() { s = getStack(1) }
		fn2 := func() { fn1() }
		fn3 := func() { fn2() }
		fn3()

		ss := strings.Split(s, "\n")
		t.True(len(ss) > 2)
		t.False(strings.Contains(ss[0], "func1.1"))
		t.True(strings.Contains(ss[0], "func1.2"))
		t.True(strings.Contains(ss[1], "func1.3"))
	})

	t.Run("0深度", func() {
		defer func() {
			TraceDepth = 8
		}()
		TraceDepth = 0
		s := getStack(0)
		t.Equal(s, "")
	})

	t.Run("获取不到stack", func() {
		s := getStack(99)
		t.Equal(s, "")
	})
}

func (t *LogTest) TestLog() {
	defer func() {
		Log = nil
	}()
	var ctx context.Context
	var msg interface{}
	var stackString string

	Log = func(c context.Context, ipanic interface{}, stack string) {
		msg = ipanic
		ctx = c
		stackString = stack
	}

	parCtx := context.Background()
	parMsg := "test log"
	log(parCtx, 0, parMsg)
	stack := strings.Split(stackString, "\n")

	t.Equal(ctx, parCtx)
	t.Equal(msg, parMsg)
	t.True(strings.Contains(stack[1], "TestLog"))
}
