package txpool

import (
	"container/heap"
	"fmt"

	"kto/transaction"
)

type nonceHeap []uint64

func (h nonceHeap) Len() int           { return len(h) }
func (h nonceHeap) Less(i, j int) bool { return h[i] < h[j] }
func (h nonceHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *nonceHeap) Push(x interface{}) {
	*h = append(*h, x.(uint64))
}

func (h *nonceHeap) Pop() interface{} {

	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

type txlist struct {
	index *nonceHeap
	items map[uint64]*transaction.Transaction
}

func NewList() *txlist {
	tx_list := &txlist{
		index: new(nonceHeap),
		items: make(map[uint64]*transaction.Transaction),
	}
	heap.Init(tx_list.index)
	fmt.Println("new List")
	return tx_list
}

func (m *txlist) Put(tx *transaction.Transaction) {
	nonce := tx.Nonce
	if _, ok := m.items[nonce]; !ok {
		fmt.Println("!ok,put")
		heap.Push(m.index, nonce)
	} else {
		if m.items[nonce].Time < tx.Time {
			return
		} else {
			m.items[nonce] = tx
		}
	}
	m.items[nonce] = tx
	fmt.Println("over,put")
}

func (m *txlist) Get(nonce uint64) *transaction.Transaction {
	if m.items[nonce] == nil {
		return nil
	}
	return m.items[nonce]
}

func (m *txlist) Remove(nonce uint64) bool {
	_, ok := m.items[nonce]
	if !ok {
		return false
	}
	for i := 0; i < m.index.Len(); i++ {
		if (*m.index)[i] == nonce {
			heap.Remove(m.index, i)
			break
		}
	}
	delete(m.items, nonce)
	return true
}
