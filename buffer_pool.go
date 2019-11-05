package logo

import (
	"bytes"
	"container/list"
	"sync"
)

type BufferPool struct {
	minCapacity uint32
	maxCapacity uint32
	curCapacity uint32
	freeList    *list.List
	mu          sync.Mutex
}

func NewBufferPool(initCapacity, maxCapacity uint32) *BufferPool {
	pool := &BufferPool{
		minCapacity: initCapacity,
		maxCapacity: maxCapacity,
		freeList:    list.New(),
		curCapacity: initCapacity,
	}
	var i uint32 = 1
	for ; i <= initCapacity; i++ {
		buffer := new(bytes.Buffer)
		pool.freeList.PushBack(buffer)
	}
	return pool
}

func (pool *BufferPool) Get() (buffer *bytes.Buffer) {
retry:
	pool.mu.Lock()
	defer pool.mu.Unlock()
	if e := pool.freeList.Front(); e != nil {
		buffer = e.Value.(*bytes.Buffer)
		pool.freeList.Remove(e)
		return
	}
	if pool.curCapacity < pool.maxCapacity {
		pool.curCapacity++
		buffer = new(bytes.Buffer)
		return
	}
	pool.mu.Unlock()
	goto retry
	return
}

func (pool *BufferPool) Return(buffer *bytes.Buffer) {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	buffer.Reset()
	pool.freeList.PushBack(buffer)
}
