package core

import (
	"Occam/config"
	"Occam/core/types"
	"math"
)

// CalculateConcurrency calculates resilient block concurrency according to the pending pool size
func CalculateConcurrency(scale int) int {
	var resCon int
	// coefficient after fitting
	//resCon = int(math.Ceil(0.008849*float64(scale) - 2.048))
	resCon = int(math.Ceil(0.005093*float64(scale) - 2.419))

	if resCon > config.InitialConcurrency {
		return config.InitialConcurrency
	} else if float64(resCon) < math.Ceil(float64(config.InitialConcurrency/2)) {
		return int(math.Ceil(float64(config.InitialConcurrency / 2)))
	}
	return resCon
}

// AnalyzeHotAccounts analyzes hot accounts in all concurrent blocks in the current epoch
func AnalyzeHotAccounts(txs map[string][]*types.Transaction, frequency int) (map[string]struct{}, map[string]map[string]struct{}) {
	var sum = make(map[string]int)
	var accesses = make(map[string]map[string]struct{}) // blockid -> accounts
	var hotAccounts = make(map[string]struct{})

	for id, set := range txs {
		access := make(map[string]struct{})
		for _, tx := range set {
			for acc := range tx.Payload {
				sum[acc]++
				access[acc] = struct{}{}
			}
		}
		accesses[id] = access
	}

	for acc, count := range sum {
		if count >= frequency {
			hotAccounts[acc] = struct{}{}
		}
	}

	return hotAccounts, accesses
}
