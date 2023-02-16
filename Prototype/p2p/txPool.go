package p2p

import (
	"MorphDAG/config"
	"MorphDAG/core/types"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"
)

type TxPool struct {
	pending *pendingPool
	queue   map[string]*types.Transaction
}

func NewTxPool() *TxPool {
	pool := &TxPool{
		pending: newPending(),
		queue:   make(map[string]*types.Transaction),
	}

	return pool
}

// GetScale gets the current workload scale
func (tp *TxPool) GetScale() int {
	return tp.pending.len()
}

// RetrievePending retrieves txs from the pending pool and adds them into the queue pool
func (tp *TxPool) RetrievePending() {
	pendingTxs := tp.pending.empty()
	fmt.Printf("tx pool size: %d\n", len(pendingTxs))
	for _, t := range pendingTxs {
		id := string(t.ID)
		tp.queue[id] = t
	}
}

// Pick randomly picks #size txs into a new block
func (tp *TxPool) Pick(size int) []*types.Transaction {
	var selectedTxs = make(map[int]struct{})
	var ids []string
	var txs []*types.Transaction

	for id := range tp.queue {
		ids = append(ids, id)
	}

	for i := 0; i < size; i++ {
		for {
			rand.Seed(time.Now().UnixNano())
			random := rand.Intn(len(ids))
			if _, ok := selectedTxs[random]; !ok {
				selectedTxs[random] = struct{}{}
				break
			}
		}
	}

	for key := range selectedTxs {
		id := ids[key]
		txs = append(txs, tp.queue[id])
	}

	return txs
}

// DeleteTxs deletes txs in other concurrent blocks
func (tp *TxPool) DeleteTxs(txs []*types.Transaction) []*types.Transaction {
	var deleted []*types.Transaction

	for _, del := range txs {
		id := string(del.ID)
		if _, ok := tp.queue[id]; ok {
			deleted = append(deleted, tp.queue[id])
			delete(tp.queue, id)
		}
	}

	return deleted
}

type pendingPool struct {
	mu sync.Mutex
	//txs map[string]*types.Transaction
	txs []*types.Transaction
}

func (pp *pendingPool) append(tx *types.Transaction) {
	//id := string(tx.ID)
	//pp.txs[id] = tx
	pp.mu.Lock()
	defer pp.mu.Unlock()
	if !pp.existInPending(tx) {
		pp.txs = append(pp.txs, tx)
	}
}

func (pp *pendingPool) batchAppend(txs []*types.Transaction) {
	for _, t := range txs {
		pp.append(t)
	}
}

func (pp *pendingPool) pop() *types.Transaction {
	pp.mu.Lock()
	defer pp.mu.Unlock()
	last := pp.txs[len(pp.txs)-1]
	pp.txs = pp.txs[:len(pp.txs)-1]
	return last
}

func (pp *pendingPool) delete(tx *types.Transaction) {
	pp.mu.Lock()
	defer pp.mu.Unlock()
	for i := range pp.txs {
		if pp.txs[i] == tx {
			pp.txs = append(pp.txs[:i], pp.txs[i+1:]...)
			break
		}
	}
	//id := string(tx.ID)
	//delete(pp.txs, id)
}

func (pp *pendingPool) len() int {
	pp.mu.Lock()
	defer pp.mu.Unlock()
	return len(pp.txs)
}

func (pp *pendingPool) swap(i, j int) {
	pp.mu.Lock()
	defer pp.mu.Unlock()
	pp.txs[i], pp.txs[j] = pp.txs[j], pp.txs[i]
}

func (pp *pendingPool) isFull() bool {
	return len(pp.txs) == config.MaximumPoolSize
}

func (pp *pendingPool) isEmpty() bool {
	return len(pp.txs) == 0
}

func (pp *pendingPool) empty() []*types.Transaction {
	pp.mu.Lock()
	defer pp.mu.Unlock()

	var txs []*types.Transaction
	if !pp.isEmpty() {
		//for _, t := range pp.txs {
		//	txs = append(txs, t)
		//}
		txs = pp.txs
		pp.txs = make([]*types.Transaction, 0, config.MaximumPoolSize)
	}

	return txs
}

func (pp *pendingPool) existInPending(tx *types.Transaction) bool {
	id := string(tx.ID)
	for _, t := range pp.txs {
		if strings.Compare(id, string(t.ID)) == 0 {
			return true
		}
	}
	return false
}

func newPending() *pendingPool {
	return &pendingPool{
		txs: make([]*types.Transaction, 0, config.MaximumPoolSize),
		//txs: make(map[string]*types.Transaction),
	}
}
