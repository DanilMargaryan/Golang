package main

import (
	"bytes"
	"encoding/json"
	"sync"
	"testing"
)

const iterNum = 10

func BenchmarkEmptyAppend(b *testing.B) {
	for i := 0; i < b.N; i++ {
		data := make([]int, 0)
		for j := 0; j < iterNum; j++ {
			data = append(data, j)
		}
	}
}

func BenchmarkPreallocAppend(b *testing.B) {
	for i := 0; i < b.N; i++ {
		data := make([]int, 0, iterNum)
		for j := 0; j < iterNum; j++ {
			data = append(data, j)
		}
	}
}

var dataPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, 64))
	},
}

func BenchmarkAllocPool(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			data := dataPool.Get().(*bytes.Buffer)
			_ = json.NewEncoder(data).Encode(Pages)
			data.Reset()
			dataPool.Put(data)
		}
	})
}
