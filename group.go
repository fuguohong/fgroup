package fgroup

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// Group go routine组
type Group struct {
	cancel func()
	ctx    context.Context

	wg sync.WaitGroup
	l  sync.Mutex

	err error

	// 最大并行数
	Parallel int
	running  int
	queue
}

// NewGroup Group构造器，传入context以便发生不可预知的panic时追踪上下文
func NewGroup(ctx context.Context) *Group {
	return &Group{ctx: ctx}
}

// NewGroupWithParallel 限制goroutine并发数，额外返回一个ctx派生的子context，在group捕获到第一个error时，子context会被取消
func NewGroupWithParallel(ctx context.Context, parallel int) (*Group, context.Context) {
	g, ctx := NewGroupWithCancel(ctx)
	g.Parallel = parallel
	return g, ctx
}

// NewGroupWithCancel 额外返回一个ctx派生的子context，在group捕获到第一个error时，子context会被取消
func NewGroupWithCancel(ctx context.Context) (*Group, context.Context) {
	childCtx, cancel := context.WithCancel(ctx)
	return &Group{cancel: cancel, ctx: childCtx}, childCtx
}

// Wait 阻塞等待所有goroutine执行完毕，返回第一个发生error或panic
func (g *Group) Wait() error {
	g.wg.Wait()
	if g.cancel != nil {
		g.cancel()
	}
	return g.err
}

// Go 在一个新的goroutine中执行func
func (g *Group) Go(f func() error) {
	if f == nil {
		return
	}
	g.l.Lock()
	isPut := g.put(f)
	if isPut {
		g.wg.Add(1)
	}
	g.l.Unlock()

	if isPut {
		g.run()
	}
}

func (g *Group) run() {
	g.l.Lock()
	if g.Parallel > 0 && g.running >= g.Parallel {
		g.l.Unlock()
		return
	}
	fn := g.pop()
	if g.err != nil || fn == nil {
		g.l.Unlock()
		return
	}

	g.running += 1
	if g.ctx != nil {
		select {
		case <-g.ctx.Done():
			g.l.Unlock()
			g.done(g.ctx.Err())
			return
		default:
		}
	}

	g.l.Unlock()

	go func() {
		var err error
		defer func() {
			g.done(err)
		}()
		defer g.recover()

		err = fn()
	}()
}

func (g *Group) done(err error) {
	g.l.Lock()
	g.running -= 1
	g.l.Unlock()

	g.wg.Done()
	if err == nil {
		g.run()
	} else {
		g.catchErr(err)
	}
}

func (g *Group) catchErr(err error) {
	g.l.Lock()
	defer g.l.Unlock()
	if err != nil && g.err == nil {
		g.err = err
		if g.cancel != nil {
			g.cancel()
		}
		l := g.abandon()
		for i := 0; i < l; i++ {
			g.wg.Done()
		}
	}
}

func (g *Group) recover() {
	e := recover()
	if e == nil {
		return
	}
	if Log != nil {
		log(g.ctx, 2, e)
	}

	var err error
	switch x := e.(type) {
	case string:
		err = errors.New(x)
	case error:
		err = x
	default:
		err = fmt.Errorf("panic: %v", e)
	}
	g.catchErr(err)
	// panic作为最高优先级的error，即使之前发生过error，也覆盖
	g.err = err
	return
}
