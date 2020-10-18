package tszlist

import (
	"container/list"
	"sync"

	"github.com/dgryski/go-tsz"
)

const defaultOverflow = 20

// List represents the safe-tszlist
type List struct {
	sync.Mutex
	l           list.List
	currBlock   *internalList
	remainBlock *internalList
	blockCap    int
	limit       int
	total       int

	fastmode bool
	front    DataPoint
}

// Option sets the List options
type Option func(*List)

// WithOverflow
func WithOverflow(n int) Option {
	return Option(func(tszList *List) {
		tszList.blockCap = n
	})
}

// WithLimit
func WithLimit(n int) Option {
	return Option(func(tszList *List) {
		tszList.limit = n
	})
}

// WithFastMode
func WithFastMode(enabled bool) Option {
	return Option(func(tszList *List) {
		tszList.fastmode = enabled
	})
}

// DataPoint
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
	l.il.PushFront(&DataPoint{Timestamp: int64(t), Value: v})
	l.frozen = l.il.Len() >= l.lcap
}

func (l *internalList) len() int {
	return l.il.Len()
}

func (l *internalList) front(n int) []*DataPoint {
	ret := make([]*DataPoint, 0)
	front := l.il.Front()

	for i := 0; i < n; i++ {
		if front == nil {
			break
		}
		ret = append(ret, front.Value.(*DataPoint))
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

// NewList
func NewList(limit int, opts ...Option) *List {
	tl := &List{limit: limit, blockCap: defaultOverflow}
	for _, opt := range opts {
		opt(tl)
	}

	tl.currBlock = &internalList{lcap: tl.blockCap}
	return tl
}

// ResetLimit
func (tl *List) ResetLimit(limit int) {
	tl.Lock()
	defer tl.Unlock()

	tl.limit = limit
	tl.removeBack()
}

// Push pushes (t, v) tuple to the list
func (tl *List) Push(t int64, v float64) {
	tl.Lock()
	defer tl.Unlock()

	tl.total++
	tl.currBlock.push(uint32(t), v)

	// if current block is frozen, then creates a new series-block
	if tl.currBlock.frozen {
		dps := reserveDps(tl.currBlock.front(tl.blockCap))

		bk := newBlock(uint32(dps[0].Timestamp))
		for i := 0; i < len(dps); i++ {
			bk.push(uint32(dps[i].Timestamp), dps[i].Value)
		}

		bk.Block.Finish()
		tl.l.PushFront(bk)
		if tl.fastmode {
			tl.remainBlock = tl.currBlock
		}
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

// Len
func (tl *List) Len() int {
	tl.Lock()
	defer tl.Unlock()

	if tl.total > tl.limit {
		return tl.limit
	}
	return tl.total
}

// Cap
func (tl *List) Cap() int {
	tl.Lock()
	defer tl.Unlock()

	return tl.total
}

// GetN
func (tl *List) GetN(n int) []*DataPoint {
	tl.Lock()
	defer tl.Unlock()

	if n > tl.limit {
		n = tl.limit
	}

	if n <= tl.blockCap {
		if tl.currBlock.len() >= n {
			return tl.currBlock.front(n)
		}
		if tl.fastmode && tl.remainBlock.len() >= n {
			return tl.remainBlock.front(n)
		}
	}

	ret := make([]*DataPoint, 0)
	ret = append(ret, tl.currBlock.front(tl.blockCap)...)
	n -= tl.currBlock.len()

	front := tl.l.Front()
	l := make([]*DataPoint, 0)
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
				l = append(l, &DataPoint{Timestamp: int64(t), Value: v})
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

func reserveDps(dps []*DataPoint) []*DataPoint {
	for i, j := 0, len(dps)-1; i < j; i, j = i+1, j-1 {
		dps[i], dps[j] = dps[j], dps[i]
	}

	return dps
}
