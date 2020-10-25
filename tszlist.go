package tszlist

import (
	"container/list"
	"sync"

	"github.com/dgryski/go-tsz"
)

const defaultOverflow = 30

// List represents the safe-tszlist
type List struct {
	l         list.List
	mux       sync.Mutex
	currBlock *internalList
	blockCap  int
	limit     int
	total     int
	front     DataPoint
}

// Option sets the List options
type Option func(*List)

// WithOverflow sets the overflow option for the list
// overflow is the count of the extra datapoints. In other words, the list capacity is `limit+overflow` actually
func WithOverflow(n int) Option {
	return Option(func(tszList *List) {
		tszList.blockCap = n
	})
}

// DataPoint is the (timestamp, value) tuple
type DataPoint struct {
	Timestamp int64
	Value     float64
}

type internalList struct {
	il     list.List
	lcap   int
	frozen bool
}

func (l *internalList) push(t uint32, v float64) {
	l.il.PushFront(DataPoint{Timestamp: int64(t), Value: v})
	l.frozen = l.il.Len() >= l.lcap
}

func (l *internalList) len() int {
	return l.il.Len()
}

func (l *internalList) front(n int) []DataPoint {
	ret := make([]DataPoint, 0, n)
	front := l.il.Front()

	for i := 0; i < n; i++ {
		if front == nil {
			break
		}
		ret = append(ret, front.Value.(DataPoint))
		front = front.Next()
	}
	return ret
}

type internalBlock struct {
	Block *tsz.Series
}

func (b *internalBlock) push(t uint32, v float64) {
	b.Block.Push(t, v)
}

func newBlock(t uint32) *internalBlock {
	return &internalBlock{Block: tsz.New(t)}
}

// NewList returns a new tszlist.List instance
func NewList(limit int, opts ...Option) *List {
	tl := &List{limit: limit, blockCap: defaultOverflow}
	for _, opt := range opts {
		opt(tl)
	}

	tl.currBlock = &internalList{lcap: tl.blockCap}
	return tl
}

// ResetLimit allows to reset the list capacity(limit) at runtime
func (tl *List) ResetLimit(limit int) {
	tl.mux.Lock()
	defer tl.mux.Unlock()

	tl.limit = limit
	tl.removeBack()
}

// Push pushes (t, v) tuple to the list
func (tl *List) Push(t int64, v float64) {
	tl.mux.Lock()
	defer tl.mux.Unlock()

	tl.total++
	tl.currBlock.push(uint32(t), v)

	// if current block is frozen, then creates a new series-block
	if tl.currBlock.frozen {
		dps := reserveDps(tl.currBlock.front(tl.blockCap))

		block := newBlock(uint32(dps[0].Timestamp))
		for i := 0; i < len(dps); i++ {
			block.push(uint32(dps[i].Timestamp), dps[i].Value)
		}

		block.Block.Finish()
		tl.l.PushFront(block)
		tl.currBlock = &internalList{lcap: tl.blockCap}
	}

	tl.removeBack()
}

func (tl *List) removeBack() {
	back := tl.l.Back()
	for tl.total > tl.limit+tl.blockCap && back != nil {
		tl.l.Remove(back)
		tl.total -= tl.blockCap
		back = tl.l.Back()
	}
}

// Len returns the length of the List
func (tl *List) Len() int {
	tl.mux.Lock()
	defer tl.mux.Unlock()

	if tl.total > tl.limit {
		return tl.limit
	}
	return tl.total
}

// Cap returns the capacity of the List
func (tl *List) Cap() int {
	tl.mux.Lock()
	defer tl.mux.Unlock()

	return tl.limit + tl.blockCap
}

// GetAll returns all datapoints in the List
func (tl *List) GetAll() []DataPoint {
	return tl.GetN(tl.limit)
}

// GetN returns N datapoints in the List
func (tl *List) GetN(n int) []DataPoint {
	tl.mux.Lock()
	defer tl.mux.Unlock()

	if n <= 0 {
		return nil
	}

	if n > tl.limit {
		n = tl.limit
	}

	if n <= tl.blockCap {
		if tl.currBlock.len() >= n {
			return tl.currBlock.front(n)
		}
	}

	ret := make([]DataPoint, 0, n)
	ret = append(ret, tl.currBlock.front(tl.blockCap)...)
	n -= tl.currBlock.len()

	front := tl.l.Front()
	l := make([]DataPoint, 0, tl.blockCap)
	for {
		if front == nil || n < 0 {
			break
		}

		cnt := 0
		nextBlock := front.Value.(*internalBlock)
		it := nextBlock.Block.Iter()
		for it.Next() {
			cnt++
			t, v := it.Values()
			if cnt > tl.blockCap-n {
				l = append(l, DataPoint{Timestamp: int64(t), Value: v})
			}
		}
		ret = append(ret, reserveDps(l)...)

		it = nextBlock.Block.Iter()
		front = front.Next()
		n -= tl.blockCap
		l = l[:0]
	}

	return ret
}

func reserveDps(dps []DataPoint) []DataPoint {
	for i, j := 0, len(dps)-1; i < j; i, j = i+1, j-1 {
		dps[i], dps[j] = dps[j], dps[i]
	}

	return dps
}
