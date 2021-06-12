package eventloop

import (
	"container/heap"
	"math/rand"
	"testing"
)

type IntHeap []int

func (h IntHeap) Len() int           { return len(h) }
func (h IntHeap) Less(i, j int) bool { return h[i] < h[j] }
func (h IntHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *IntHeap) Push(x interface{}) {
	*h = append(*h, x.(int))
}

func (h *IntHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

func testData() IntHeap {
	n := 10000
	testData := IntHeap(make([]int, 0, n))
	for i := 0; i < n; i++ {
		testData = append(testData, rand.Intn(10000))
	}

	return testData
}

func BenchmarkHeap(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data := testData()
		heap.Init(&data)
		heap.Pop(&data)
	}
}

func min(data IntHeap) int {
	ret := data[0]
	for _, v := range data {
		if v < ret {
			ret = v
		}
	}

	return ret
}

func BenchmarkMin(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data := testData()
		min(data)
	}
}
