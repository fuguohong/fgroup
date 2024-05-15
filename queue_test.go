package fgroup

import (
	"github.com/stretchr/testify/suite"
	"testing"
)

type QueueTest struct {
	suite.Suite
}

func TestQueue(t *testing.T) {
	suite.Run(t, new(QueueTest))
}

func (t *QueueTest) TestQueue() {
	i := 0
	q := queue{}
	fn1 := func() error {
		i = 1
		return nil
	}
	fn2 := func() error {
		i = 2
		return nil
	}
	fn3 := func() error {
		i = 3
		return nil
	}

	t.True(q.pop() == nil)
	t.Equal(q.len, 0)

	q.put(fn1)
	q.put(fn2)
	t.Equal(q.len, 2)

	q.pop()()
	t.Equal(i, 1)
	t.Equal(q.len, 1)

	q.put(fn3)
	t.Equal(q.len, 2)

	q.pop()()
	q.pop()()
	t.Equal(q.len, 0)
	t.Equal(i, 3)

	t.True(q.pop() == nil)
	t.Equal(q.len, 0)

	q = queue{}
	q.put(func() error {
		return nil
	})
	t.True(q.pop() != nil)
	q.put(func() error {
		return nil
	})
	t.True(q.pop() != nil)
}

// group保证了线程安全，queue不再确保线程安全
// func (t *QueueTest) TestMultiRoutine() {
// 	for run := 0; run < 10; run++ {
// 		q := queue{}
// 		maxlen := 200
// 		result := make([]bool, maxlen)
// 		g1 := &Group{}
// 		for i := 0; i < maxlen; i++ {
// 			x := i
// 			g1.Go(func() error {
// 				q.put(func() error {
// 					result[x] = true
// 					return nil
// 				})
// 				return nil
// 			})
// 		}
//
// 		g2 := &Group{}
// 		notget := 0
// 		l := sync.Mutex{}
// 		for i := 0; i < maxlen; i++ {
// 			g1.Go(func() error {
// 				fn := q.pop()
// 				if fn == nil {
// 					l.Lock()
// 					notget += 1
// 					l.Unlock()
// 				} else {
// 					fn()
// 				}
// 				return nil
// 			})
// 		}
//
// 		g1.Wait()
// 		g2.Wait()
//
// 		t.Equal(q.len, notget)
// 		if notget > 0 {
// 			t.NotNil(q.fist)
// 			t.NotNil(q.last)
// 		}
// 		for {
// 			fn := q.pop()
// 			if fn == nil {
// 				break
// 			}
// 			fn()
// 		}
//
// 		for _, r := range result {
// 			t.True(r)
// 		}
// 	}
// }

func (t *QueueTest) TestAbandon() {
	q := &queue{}

	t.True(q.put(func() error { return nil }))

	t.False(q.isAbandon)
	t.True(q.len == 1)
	t.True(q.fist != nil)
	t.True(q.last != nil)

	t.Equal(q.abandon(), 1)
	t.Equal(q.len, 0)
	t.False(q.put(nil))
	t.Equal(q.abandon(), 0)
}
