package fgroup

import (
	"context"
	"errors"
	"fmt"
	"github.com/stretchr/testify/suite"
	"strings"
	"testing"
	"time"
)

type GroupTest struct {
	suite.Suite
}

func TestGroup(t *testing.T) {
	suite.Run(t, new(GroupTest))
}

func (t *GroupTest) TestGroup() {
	t.Run("panic 覆盖普通错误", func() {
		// 无论执行先后顺序，panic总能覆盖普通error
		for i := 0; i < 10; i++ {
			g := &Group{}
			ch := make(chan bool)
			g.Go(func() error {
				<-ch
				return errors.New("normal error")
			})
			g.Go(func() error {
				close(ch)
				time.Sleep(time.Millisecond * 50)
				panic("group panic")
				return nil
			})

			err := g.Wait()
			t.NotNil(err)
			t.Equal(err.Error(), "group panic")
			t.Equal(g.running, 0)
			t.Equal(g.len, 0)
		}

	})

	t.Run("错误阻止后续任务执行", func() {
		g := &Group{}
		i := 1
		ch := make(chan bool)
		g.Go(func() error {
			defer close(ch)
			i = 2
			return errors.New("error")
		})
		<-ch
		time.Sleep(time.Millisecond * 50)
		g.Go(func() error {
			i = 3
			panic("panic")
			return nil
		})
		err := g.Wait()
		t.Equal(i, 2)
		t.Equal(err.Error(), "error")
		t.Equal(g.running, 0)
		t.Equal(g.len, 0)
	})

	t.Run("panic阻止后续任务执行", func() {
		g := &Group{}
		i := 1
		ch := make(chan bool)
		g.Go(func() error {
			defer close(ch)
			i = 2
			panic("error")
			return nil
		})
		<-ch
		time.Sleep(time.Millisecond * 50)
		g.Go(func() error {
			i = 3
			return nil
		})
		g.Wait()
		t.Equal(i, 2)
		t.Equal(g.running, 0)
		t.Equal(g.len, 0)
	})

	t.Run("等待全部执行完", func() {
		g := &Group{}
		result := make([]int, 50)
		for i := 0; i < 50; i++ {
			x := i
			g.Go(func() error {
				result[x] = x
				return nil
			})
		}
		err := g.Wait()
		t.Nil(err)
		t.Equal(g.running, 0)
		t.Equal(g.len, 0)
		for x, r := range result {
			t.Equal(r, x)
		}
	})

	t.Run("添加nil函数", func() {
		g := &Group{}
		ch := make(chan bool)
		g.Go(func() error {
			<-ch
			return nil
		})
		g.Go(nil)
		t.Equal(g.running, 1)
		t.Equal(g.len, 0)
		close(ch)
		t.Nil(g.Wait())
		t.Equal(g.running, 0)
		t.Equal(g.len, 0)
	})
}

func (t *GroupTest) TestWithCancel() {
	g, ctx := NewGroupWithCancel(context.Background())
	g.Go(func() error {
		panic("context panic")
		return nil
	})

	err := g.Wait()
	select {
	case <-ctx.Done():
	default:
		t.Error(nil, "context not cancel")
	}

	t.NotNil(err)
	t.Equal(err.Error(), "context panic")
	t.Equal(g.running, 0)
	t.Equal(g.len, 0)
}

func (t *GroupTest) TestPanicType() {
	defer func() {
		Log = nil
	}()
	var recoverErr interface{}
	Log = func(ctx context.Context, i interface{}, stack string) {
		recoverErr = i
	}

	g := &Group{}
	strErr := "string panic"
	g.Go(func() error {
		panic(strErr)
		return nil
	})
	err := g.Wait()
	t.NotNil(err)
	t.Equal(err.Error(), "string panic")
	t.Equal(recoverErr, strErr)

	g = &Group{}
	errorErr := errors.New("error panic")
	g.Go(func() error {
		panic(errorErr)
		return nil
	})
	err = g.Wait()
	t.NotNil(err)
	t.Equal(err.Error(), "error panic")
	t.Equal(recoverErr, errorErr)

	g = &Group{}
	intErr := 9
	g.Go(func() error {
		panic(intErr)
		return nil
	})
	err = g.Wait()
	t.NotNil(err)
	t.Equal(err.Error(), "panic: 9")
	t.Equal(recoverErr, intErr)
}

func (t *GroupTest) TestGroupLog() {
	ch := make(chan bool)
	defer func() {
		Log = nil
		close(ch)
	}()
	var msg interface{}
	var ctx context.Context
	var stack string
	Log = func(c context.Context, i interface{}, s string) {
		msg = i
		stack = s
		ctx = c
		ch <- true
	}

	t.Run("Group.Go log with context", func() {
		g := &Group{}
		g.Go(func() error {
			makePanic()
			return nil
		})
		<-ch

		stacks := strings.Split(stack, "\n")
		t.True(len(stacks) > 1)
		t.True(strings.Contains(stacks[1], "makePanic"))
		t.Equal(g.ctx, nil)
		t.Equal(ctx, nil)
		t.Equal(msg, panicErr)
	})

	t.Run("Group.Go log without context", func() {
		g := NewGroup(context.Background())
		g.Go(func() error {
			makePanic()
			return nil
		})
		<-ch

		stacks := strings.Split(stack, "\n")
		t.True(len(stacks) > 1)
		t.True(strings.Contains(stacks[1], "makePanic"))
		t.Equal(msg, panicErr)
	})
}

func (t *GroupTest) TestParallel() {
	t.Run("测试并发限制", func() {
		g, _ := NewGroupWithParallel(context.Background(), 2)
		ch := make(chan bool)
		for i := 0; i < 5; i++ {
			g.Go(func() error {
				<-ch
				return nil
			})
		}
		t.Equal(g.running, 2)
		t.Equal(g.len, 3)

		ch <- true
		time.Sleep(time.Millisecond * 50)
		t.Equal(g.running, 2)
		t.Equal(g.len, 2)

		close(ch)
		g.Wait()
		t.Equal(g.running, 0)
		t.Equal(g.len, 0)
	})

	t.Run("执行时长", func() {
		g, _ := NewGroupWithParallel(context.Background(), 2)
		start := time.Now()
		for i := 0; i < 5; i++ {
			g.Go(func() error {
				time.Sleep(time.Millisecond * 50)
				return nil
			})
		}
		t.Equal(g.running, 2)
		t.Equal(g.len, 3)
		g.Wait()
		d := time.Now().Sub(start)
		t.True(d.Milliseconds() >= 150)
		t.Equal(g.running, 0)
		t.Equal(g.len, 0)
	})

	t.Run("错误后清理等待队列", func() {
		g, _ := NewGroupWithParallel(context.Background(), 1)
		ch := make(chan bool)
		g.Go(func() error {
			<-ch
			return errors.New("error")
		})
		g.Go(func() error {
			return nil
		})
		g.Go(func() error {
			return nil
		})

		t.True(g.len == 2)
		t.True(g.running == 1)
		close(ch)
		err := g.Wait()
		t.NotNil(err)
		t.True(g.len == 0)
		t.True(g.running == 0)
	})

	t.Run("高并发错误处理", func() {
		g, _ := NewGroupWithParallel(context.Background(), 30)
		for i := 0; i <= 666; i++ {
			x := i
			g.Go(func() error {
				if x%200 == 0 {
					return fmt.Errorf("error: %d", x)
				}
				return nil
			})
		}

		wait := make(chan bool)
		go func() {
			err := g.Wait()
			t.NotNil(err)
			t.Equal(g.running, 0)
			wait <- true
		}()
		select {
		case <-wait:
		case <-time.After(time.Second):
			t.Error(nil, "wait等待超时")
		}
	})

	t.Run("正确执行", func() {
		g, _ := NewGroupWithParallel(context.Background(), 6)
		result := make([]bool, 50)
		for i := 0; i < 50; i++ {
			x := i
			g.Go(func() error {
				result[x] = true
				return nil
			})
		}
		err := g.Wait()
		t.Nil(err)
		for _, r := range result {
			t.True(r)
		}
	})
}

func (t *GroupTest) TestContextDead() {
	t.Run("测试context过期", func() {
		ctx, _ := context.WithTimeout(context.Background(), time.Millisecond*50)
		g := NewGroup(ctx)
		i := 0
		g.Go(func() error {
			i += 1
			return nil
		})
		time.Sleep(time.Millisecond * 60)
		g.Go(func() error {
			i += 1
			return nil
		})
		err := g.Wait()
		t.Equal(i, 1)
		t.Equal(g.running, 0)
		t.NotNil(err)
	})

	t.Run("测试context取消", func() {
		ctx, cancel := context.WithCancel(context.Background())
		ch := make(chan bool)
		g := NewGroup(ctx)
		i := 0
		g.Go(func() error {
			i += 1
			cancel()
			close(ch)
			return nil
		})
		<-ch
		g.Go(func() error {
			i += 1
			return nil
		})
		err := g.Wait()
		t.Equal(i, 1)
		t.Equal(g.running, 0)
		t.NotNil(err)
	})
}
