package types

import (
	"Occam/config"
	"Occam/utils"
	"bytes"
	"crypto/sha256"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"time"
)

// ProofOfWork represents a proof-of-work
type ProofOfWork struct {
	block  *Block
	target *big.Int
	bits   int
}

// NewProofOfWork builds and returns a ProofOfWork
func NewProofOfWork(b *Block, con int) *ProofOfWork {
	target := big.NewInt(1)
	newT := adjustTarget(con)
	target.Lsh(target, uint(256-newT))

	pow := &ProofOfWork{b, target, int(newT)}

	return pow
}

// adjustTarget adjusts the pow target according to the resilient DAG concurrency
func adjustTarget(con int) float64 {
	prob := math.Pow(2, float64(config.TargetBits))
	diff := prob / float64(con)
	newT := math.Floor(math.Log2(diff))
	return newT
}

func (pow *ProofOfWork) prepareData(nonce int) []byte {
	rand.Seed(time.Now().UnixNano())
	r := rand.Intn(100000000)

	data := bytes.Join(
		[][]byte{
			pow.block.HashTransactions(),
			utils.IntToHex(int64(r)),
			utils.IntToHex(int64(pow.bits)),
			utils.IntToHex(int64(nonce)),
		},
		[]byte{},
	)

	return data
}

// Run performs a proof-of-work
func (pow *ProofOfWork) Run() (int, []byte) {
	var hashInt big.Int
	var hash [32]byte
	nonce := 0
	rounds := int(math.Floor(math.Pow(2, float64(config.TargetBits)) / config.NodeNumber))
	find := false

	fmt.Println("Mining a new block")
	start := time.Now()
	for i := 0; i < rounds; i++ {
		data := pow.prepareData(nonce)

		hash = sha256.Sum256(data)
		//if math.Remainder(float64(nonce), 100000) == 0 {
		//	fmt.Printf("\r%x", hash)
		//}
		hashInt.SetBytes(hash[:])

		if hashInt.Cmp(pow.target) == -1 && !find {
			find = true
		} else if !(hashInt.Cmp(pow.target) == -1) && !find {
			nonce++
		}
	}

	duration := time.Since(start)

	if nonce < rounds {
		fmt.Printf("time of solving PoW: %s\n", duration)
		return nonce, hash[:]
	}

	fmt.Printf("fail to solve the PoW puzzle: %s\n", duration)
	return 0, nil
}

// Validate validates block's PoW
func (pow *ProofOfWork) Validate() bool {
	var hashInt big.Int

	data := pow.prepareData(pow.block.Nonce)
	hash := sha256.Sum256(data)
	hashInt.SetBytes(hash[:])

	isValid := hashInt.Cmp(pow.target) == -1

	return isValid
}
