package tszlist

import (
	"container/list"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const factor = 3.14

func TestListPush(t *testing.T) {
	limit := 50
	l := NewList(limit)
	for i := 0; i < 70; i++ {
		l.Push(int64(i), float64(i)*factor)
	}

	assert.Equal(t, l.Len(), limit)
	assert.Equal(t, l.Cap(), limit+30)
}

func TestListGet(t *testing.T) {
	limit := 100
	l := NewList(limit)
	assert.Equal(t, len(l.GetN(1)), 0)

	for i := 1; i <= 1024; i++ {
		l.Push(int64(i), float64(i)*factor)
	}

	r := l.GetN(1)
	assert.Equal(t, r[0].Timestamp, int64(1024))
	assert.Equal(t, r[0].Value, float64(1024)*factor)

	r = l.GetN(20)
	assert.Equal(t, r[0].Timestamp, int64(1024))
	assert.Equal(t, r[0].Value, float64(1024)*factor)

	assert.Equal(t, r[19].Timestamp, int64(1005))
	assert.Equal(t, r[19].Value, float64(1005)*factor)

	r = l.GetN(100)
	assert.Equal(t, l.Len(), 100)
	assert.Equal(t, len(r), 100)

	assert.Equal(t, r[1].Timestamp, int64(1023))
	assert.Equal(t, r[1].Value, float64(1023)*factor)

	assert.Equal(t, r[99].Timestamp, int64(925))
	assert.Equal(t, r[99].Value, float64(925)*factor)

	l.ResetLimit(10)
	assert.Equal(t, l.Len(), 10)
	assert.Equal(t, len(l.GetAll()), 10)

	r = l.GetN(100)
	assert.Equal(t, len(r), 10)
	assert.Equal(t, r[9].Timestamp, int64(1015))
	assert.Equal(t, r[9].Value, float64(1015)*factor)
}

type StdList struct {
	l     list.List
	limit int
}

func NewStdList(limit int) *StdList {
	return &StdList{limit: limit}
}

func (sl *StdList) Push(t int64, v float64) {
	sl.l.PushFront(DataPoint{Timestamp: t, Value: v})

	if sl.l.Len() > sl.limit {
		sl.l.Remove(sl.l.Back())
	}
}

func (sl *StdList) GetN(limit int) []DataPoint {
	front := sl.l.Front()
	ret := make([]DataPoint, 0)
	n := 0

	for {
		if front == nil || n > limit {
			break
		}
		ret = append(ret, front.Value.(DataPoint))
		n++
		front = front.Next()
	}

	return ret
}

const listWriteCap = 20000 // 2w

func BenchmarkTszListWrite(b *testing.B) {
	l := NewList(listWriteCap)
	for i := 0; i < b.N; i++ {
		l.Push(int64(i), float64(i)*1.12)
	}
}

func BenchmarkStdListWrite(b *testing.B) {
	l := NewList(listWriteCap)
	for i := 0; i < b.N; i++ {
		l.Push(int64(i), float64(i)*1.12)
	}
}

const listReadCap = 240
const listSearch = 30

func BenchmarkTszListRead(b *testing.B) {
	l := NewList(listReadCap, WithOverflow(25))
	for i := 0; i < listReadCap; i++ {
		l.Push(int64(i), float64(i)*1.12)
	}

	for i := 0; i < b.N; i++ {
		l.GetN(int(rand.Int63n(listSearch)))
	}
}

func BenchmarkStdListRead(b *testing.B) {
	l := NewStdList(listReadCap)
	for i := 0; i < listReadCap; i++ {
		l.Push(int64(i), float64(i)*1.12)
	}

	for i := 0; i < b.N; i++ {
		l.GetN(int(rand.Int63n(listSearch)))
	}
}

const seriesCnt = 200000 // 20w
const listLimit = 20

func newStdList() {
	ls := make([]*StdList, 0, seriesCnt)

	now := time.Now().Unix()
	for i := 0; i < seriesCnt; i++ {
		l := NewStdList(listLimit)
		for j := 0; j < listLimit; j++ {
			l.Push(now, float64(j)*1.12)
			now = now + 5
		}
		ls = append(ls, l)
	}

	time.Sleep(time.Minute)
}

func newTszList() {
	ls := make([]*List, 0, seriesCnt)

	now := time.Now().Unix()
	for i := 0; i < seriesCnt; i++ {
		l := NewList(listLimit, WithOverflow(15))
		for j := 0; j < listLimit; j++ {
			l.Push(now, float64(j)*1.12)
			now = now + 5
		}
		ls = append(ls, l)
	}

	time.Sleep(time.Minute)
}
