package fgroup

type node struct {
	fn   func() error
	next *node
}

type queue struct {
	len       int
	fist      *node
	last      *node
	isAbandon bool
}

func (q *queue) put(fn func() error) bool {

	if q.isAbandon {
		return false
	}
	q.len += 1
	n := &node{fn: fn}
	if q.fist == nil {
		q.fist = n
		q.last = n
	} else {
		q.last.next = n
		q.last = n
	}
	return true
}

func (q *queue) pop() func() error {
	if q.fist == nil {
		return nil
	}

	q.len -= 1
	fn := q.fist.fn
	q.fist = q.fist.next
	return fn
}

func (q *queue) abandon() int {
	if q.isAbandon {
		return 0
	}
	l := q.len
	q.isAbandon = true
	q.len = 0
	q.fist = nil
	q.last = nil
	return l
}
