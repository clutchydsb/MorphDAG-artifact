package core

import (
	"MorphDAG/core/state"
	"MorphDAG/core/tp"
	"MorphDAG/core/types"
	"fmt"
	"sort"
	"time"
)

type Executor struct {
	dispatcher  *tp.Dispatcher
	state       *state.StateDB
	hotAccounts map[string]struct{}
}

// NewExecutor creates a new executor
func NewExecutor(state *state.StateDB, maxProcessors, txNum int) *Executor {
	return &Executor{
		dispatcher: tp.NewDispatcher(maxProcessors, txNum),
		state:      state,
	}
}

// Processing processes given transaction sets
func (e *Executor) Processing(txs map[string][]*types.Transaction, frequency int) []byte {
	// initialize state DB
	e.state.BatchCreateObjects(txs)

	start := time.Now()
	// prefetch the state of hot accounts into memory
	e.state.PreFetch(e.GetHotAccounts())
	// track dependency between concurrent blocks
	deps := e.TrackDependency(txs, frequency)
	// execute txs and commit state updates
	graph := tp.CreateGraph(deps, txs, e.hotAccounts)
	e.dispatcher.Run(graph, e.state, deps)
	stateRoot := e.state.Commit()
	duration := time.Since(start)
	fmt.Printf("Time of processing transactions is: %s\n", duration)

	// clear in-memory data (default interval: one epoch)
	e.state.Reset()
	return stateRoot
}

// SerialProcessing serially processes given transaction sets
func (e *Executor) SerialProcessing(txs map[string][]*types.Transaction) {
	e.state.BatchCreateObjects(txs)

	start := time.Now()
	for blk := range txs {
		for _, tx := range txs[blk] {
			msg := tx.AsMessage()
			err := tp.ApplyMessageForSerial(e.state, msg)
			if err != nil {
				panic(err)
			}
		}
	}
	e.state.Commit()
	duration := time.Since(start)
	fmt.Printf("Time of processing transactions is: %s\n", duration)

	e.state.Reset()
}

// TrackDependency tracks dependency between concurrent blocks
func (e *Executor) TrackDependency(txs map[string][]*types.Transaction, frequency int) []string {
	var deps []string
	var sorted []string
	var cold []string
	var graph = make(map[string]map[string]struct{})
	var hotAccesses = make(map[string][]string) // account -> blocks

	// if there is only one block
	if len(txs) == 1 {
		for id := range txs {
			deps = append(deps, id)
		}
		return deps
	}

	hotAccounts, accesses := AnalyzeHotAccounts(txs, frequency)
	e.UpdateHotAccounts(hotAccounts)

	// find blocks that do not access hot accounts
	for id, access := range accesses {
		isHot := false
		for acc := range access {
			if _, ok := hotAccounts[acc]; ok {
				hotAccesses[acc] = append(hotAccesses[acc], id)
				isHot = true
			}
		}
		if !isHot {
			cold = append(cold, id)
		}
	}

	sort.Strings(cold)

	// if there are no blocks accessing hot accounts
	if len(hotAccesses) == 0 {
		return cold
	}

	for _, blks := range hotAccesses {
		sort.Strings(blks)
		for i, blk := range blks {
			if _, ok := graph[blk]; !ok {
				edges := make(map[string]struct{})
				graph[blk] = edges
			}
			if i < len(blks)-1 {
				for _, con := range blks[i+1:] {
					graph[blk][con] = struct{}{}
				}
			}
		}
	}

	for blk := range graph {
		sorted = append(sorted, blk)
	}
	sort.Strings(sorted)

	for i := 0; i < len(sorted); i++ {
		if _, ok := graph[sorted[i]]; ok {
			deps = append(deps, sorted[i])
			conflicts := graph[sorted[i]]
			delete(graph, sorted[i])
			for j := i + 1; j < len(sorted); j++ {
				if _, ok2 := graph[sorted[j]]; ok2 {
					// if the block has no dependency with the current block
					if _, ok3 := conflicts[sorted[j]]; !ok3 {
						deps = append(deps, sorted[j])
						for addr := range graph[sorted[j]] {
							conflicts[addr] = struct{}{}
						}
						delete(graph, sorted[j])
					}
				}
			}
		}
	}

	// add blocks accessing cold accounts
	deps = append(deps, cold...)
	return deps
}

func (e *Executor) UpdateHotAccounts(newAccounts map[string]struct{}) {
	e.hotAccounts = newAccounts
}

func (e *Executor) GetHotAccounts() map[string]struct{} { return e.hotAccounts }
