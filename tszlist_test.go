package tszlist

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestListPush(t *testing.T) {
	limit := 50
	l := NewList(limit)
	for i := 0; i < 70; i++ {
		l.Push(int64(i), float64(i)*3.14)
	}

	assert.Equal(t, l.Len(), limit)
	assert.Equal(t, l.Cap(), limit+20)
}

func TestListGet(t *testing.T) {
	limit := 100
	l := NewList(limit)
	assert.Equal(t, len(l.GetN(1)), 0)

	for i := 1; i <= 1024; i++ {
		l.Push(int64(i), float64(i)*3.14)
	}

	r := l.GetN(1)
	assert.Equal(t, r[0].Timestamp, int64(1024))

	r = l.GetN(20)
	assert.Equal(t, r[0].Timestamp, int64(1024))
	assert.Equal(t, r[19].Timestamp, int64(1005))

	r = l.GetN(100)
	assert.Equal(t, l.Len(), 100)
	assert.Equal(t, len(r), 100)
	assert.Equal(t, r[1].Timestamp, int64(1023))
	assert.Equal(t, r[99].Timestamp, int64(925))

	l.ResetLimit(10)
	assert.Equal(t, l.Len(), 10)

	r = l.GetN(100)
	assert.Equal(t, len(r), 10)
	assert.Equal(t, r[9].Timestamp, int64(1015))
}

func BenchmarkWrite(b *testing.B) {
	l := NewList(2 << 16)
	for i := 0; i < b.N; i++ {
		l.Push(int64(i), float64(i)*1.10)
	}
}

func BenchmarkRead(b *testing.B) {
	l := NewList(255)
	for i := 0; i < 255; i++ {
		l.Push(int64(i), float64(i)*1.10)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.GetN(int(rand.Int63n(50)))
	}
}
