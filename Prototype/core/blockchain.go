package core

import (
	"MorphDAG/config"
	"MorphDAG/core/types"
	"MorphDAG/utils"
	"bytes"
	"crypto/ecdsa"
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/wealdtech/go-merkletree"
	"log"
	"math"
	"os"
	"reflect"
	"sort"
	"sync"
)

// Blockchain implements a MorphDAG instance
type Blockchain struct {
	preChainSets ChainSets
	curChainSets map[int][]*types.Block
	DB           *leveldb.DB
	height       int
}

type ChainSets map[int]*SubChain

type SubChain struct {
	Tip     []byte
	ChainID int
}

// tempBlocks used to help slow nodes to catch newly blocks
var tempBlocks = make(map[int][]*types.Block)
var tempConnected = make(map[int]struct{})
var connected = make(map[int]struct{})
var mu sync.Mutex

// Serialize returns a serialized chain-sets
func (cs ChainSets) Serialize() []byte {
	var encoded bytes.Buffer

	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(cs)
	if err != nil {
		log.Panic(err)
	}

	return encoded.Bytes()
}

// DeserializeChainSets deserializes a chain-sets
func DeserializeChainSets(data []byte) ChainSets {
	var chainSets ChainSets

	decoder := gob.NewDecoder(bytes.NewReader(data))
	err := decoder.Decode(&chainSets)
	if err != nil {
		log.Panic(err)
	}

	return chainSets
}

// CreateBlockchain creates multiple blockchain instances
func CreateBlockchain(nodeID string, address string, chainNumber int) *Blockchain {
	dbFile := fmt.Sprintf(config.DBfile, nodeID)
	if dbExists(dbFile) {
		fmt.Println("Blockchain already exists.")
		os.Exit(1)
	}

	db, err := LoadDB(dbFile)
	if err != nil {
		log.Panic(err)
	}

	//preChainSets := make(map[int]*SubChain)
	preChainSets := make(map[int]*SubChain)
	curChainSets := make(map[int][]*types.Block)

	for i := 0; i < chainNumber; i++ {
		cbtx := types.NewCoinbaseTX(address)
		genesis := types.NewGenesisBlock(cbtx, i)
		err = StoreBlock(db, *genesis)
		if err != nil {
			log.Panic(err)
		}
		tip := genesis.BlockHash
		sc := &SubChain{tip, i}
		preChainSets[i] = sc
	}

	//err = StoreBlockHashes(db, preChainSets)
	//if err != nil {
	//	log.Panic(err)
	//}
	bc := Blockchain{preChainSets, curChainSets, db, 0}

	return &bc
}

// NewBlockchain creates a new blockchain with genesis block
//func NewBlockchain(nodeID string, height int) *Blockchain {
//	dbFile := fmt.Sprintf(config.DBfile, nodeID)
//	if dbExists(dbFile) == false {
//		fmt.Println("No existing blockchain found. Create one first.")
//		os.Exit(1)
//	}
//
//	db, err := LoadDB(dbFile)
//	if err != nil {
//		log.Panic(err)
//	}
//
//	preChainSets := make(map[int]*SubChain)
//	curChainSets := make(map[int][]*types.Block)
//
//	hashes, err := FetchBlockHashes(db)
//	if err != nil {
//		log.Panic(err)
//	}
//	preChainSets = DeserializeChainSets(hashes)
//	bc := Blockchain{preChainSets, curChainSets, db, height}
//
//	return &bc
//}

// EnterNextEpoch resets parameters for the next epoch
func (bc *Blockchain) EnterNextEpoch() {
	mu.Lock()
	defer mu.Unlock()
	for id, blk := range bc.curChainSets {
		subChain := &SubChain{blk[0].BlockHash, id}
		bc.preChainSets[id] = subChain
	}

	// update the current received blocks
	connected = tempConnected
	tempConnected = make(map[int]struct{})
	bc.curChainSets = tempBlocks
	tempBlocks = make(map[int][]*types.Block)
	bc.height++
}

// UpdateEdges updates parent edges and store blocks into the underlying database
func (bc *Blockchain) UpdateEdges() {
	// connect to the remaining tips
	var remaining [][]byte
	for id, blk := range bc.preChainSets {
		if _, ok := connected[id]; !ok {
			remaining = append(remaining, blk.Tip)
		}
	}

	counter := 0
	for _, blk2 := range bc.curChainSets {
		if counter == len(remaining) {
			break
		}
		block := blk2[0]
		block.PrevBlockHash = append(block.PrevBlockHash, remaining[counter])
		StoreBlock(bc.DB, *block)
		counter++
	}
}

// AddBlock adds the block into the specific sub-chain
func (bc *Blockchain) AddBlock(block *types.Block, chainID int, root []byte) bool {
	mu.Lock()
	defer mu.Unlock()
	isVerified, height := bc.VerifyBlock(block, root)

	if isVerified {
		if block.Epoch == height+1 {
			bc.curChainSets[chainID] = append(bc.curChainSets[chainID], block)
		} else {
			tempBlocks[chainID] = append(tempBlocks[chainID], block)
		}

		// mark the connected blocks
		for _, hash := range block.PrevBlockHash {
			bitNum := math.Ceil(math.Log2(float64(len(bc.preChainSets))))
			id := utils.ConvertBinToDec(hash, int(bitNum))
			if block.Epoch == height+1 {
				connected[id] = struct{}{}
			} else {
				tempConnected[id] = struct{}{}
			}
		}
		return true
	} else {
		return false
	}
}

// FindTransaction finds a transaction by its ID
func (bc *Blockchain) FindTransaction(ID []byte) (types.Transaction, error) {
	bci := bc.Iterator()

	for {
		blocks := bci.Previous()
		isBreak := false
		for _, blk := range blocks {
			for _, tx := range blk.Transactions {
				if bytes.Compare(tx.ID, ID) == 0 {
					return *tx, nil
				}
			}
			if !isBreak && len(blk.PrevBlockHash) == 0 {
				isBreak = true
			}
		}
		if isBreak {
			break
		}
	}

	return types.Transaction{}, errors.New("transaction is not found")
}

// Iterator returns a blockchain iterator
func (bc *Blockchain) Iterator() *Iterator {
	var currentHash = make(map[string]struct{})

	for _, blk := range bc.preChainSets {
		currentHash[string(blk.Tip)] = struct{}{}
	}

	bci := &Iterator{currentHash, bc.DB}
	return bci
}

// GetLatestHeight returns the epoch of the latest block
func (bc *Blockchain) GetLatestHeight() int { return bc.height }

// GetBlock finds a block by its hash and returns it
func (bc *Blockchain) GetBlock(blockHash []byte) (*types.Block, error) {
	blockData, err := FetchBlock(bc.DB, blockHash)
	if err != nil {
		log.Panic(err)
	}
	if blockData == nil {
		return nil, errors.New("block is not found")
	}
	block := types.DeserializeBlock(blockData)

	return block, nil
}

// GetAllBlockHashes returns a list of hashes of all the blocks in the blockchain
func (bc *Blockchain) GetAllBlockHashes() [][]byte {
	var blockSets [][]byte
	bci := bc.Iterator()

	for {
		blocks := bci.Previous()
		isBreak := false
		for _, blk := range blocks {
			blockSets = append(blockSets, blk.BlockHash)
			if !isBreak && len(blk.PrevBlockHash) == 0 {
				isBreak = true
			}
		}
		if isBreak {
			break
		}
	}

	return blockSets
}

// GetLatestHashes returns a list of block hashes in the latest epoch
func (bc *Blockchain) GetLatestHashes() [][]byte {
	var blockHashes [][]byte

	curHashes := bc.Iterator().currentHash
	for hash := range curHashes {
		blockHashes = append(blockHashes, []byte(hash))
	}

	return blockHashes
}

// GetHashesByEpoch returns a list of block hashes in a given epoch
func (bc *Blockchain) GetHashesByEpoch(epoch int) [][]byte {
	var blockHashes [][]byte
	bci := bc.Iterator()
	iterations := bc.height - epoch

	for i := 0; i < iterations; i++ {
		_ = bci.Previous()
	}

	curHashes := bci.currentHash
	for hash := range curHashes {
		blockHashes = append(blockHashes, []byte(hash))
	}

	return blockHashes
}

// GetCurrentBlocks obtains all added blocks in a new epoch
func (bc *Blockchain) GetCurrentBlocks() map[int][]*types.Block {
	return bc.curChainSets
}

// MineBlock mines a new block with the provided transactions
func (bc *Blockchain) MineBlock(transactions []*types.Transaction, con, nodeNumber int, stateRoot []byte) *types.Block {
	var latestEpoch int
	var lastHash [][]byte
	var newBlock *types.Block

	latestEpoch = bc.GetLatestHeight()
	lastHash = bc.GetLatestHashes()

	// compute the merkle root of all the block hashes
	mTree, err := merkletree.New(lastHash)
	if err != nil {
		log.Panic(err)
	}
	rootHash := mTree.Root()

	if latestEpoch == 0 || latestEpoch == 1 {
		// if this epoch is the first or second round of mining
		newBlock = types.NewBlock(transactions, rootHash, nil, latestEpoch+1, con, nodeNumber)
	} else {
		newBlock = types.NewBlock(transactions, rootHash, stateRoot, latestEpoch+1, con, nodeNumber)
	}

	// if the miner mines a valid block in an epoch
	if newBlock != nil {
		chainID, edges := bc.MappingChain(newBlock.BlockHash, con)
		// build edges to the parent blocks
		for _, e := range edges {
			connected[e] = struct{}{}
			blk := bc.preChainSets[e]
			newBlock.PrevBlockHash = append(newBlock.PrevBlockHash, blk.Tip)
		}

		// generate the merkle proof
		prevHash := newBlock.PrevBlockHash[0]
		proof, err := mTree.GenerateProof(prevHash, 0)
		if err != nil {
			log.Panic(err)
		}
		newBlock.MerkleProof = proof

		bc.curChainSets[chainID] = append(bc.curChainSets[chainID], newBlock)
	}

	return newBlock
}

// SignTransaction generates a signature on the transaction
func (bc *Blockchain) SignTransaction(tx *types.Transaction, privKey ecdsa.PrivateKey) {
	tx.Sign(privKey)
}

// VerifyTransaction verifies transaction input signatures
func (bc *Blockchain) VerifyTransaction(tx *types.Transaction) bool {
	return tx.VerifyTransaction()
}

// VerifyBlock verifies the validity of a new block
func (bc *Blockchain) VerifyBlock(block *types.Block, root []byte) (bool, int) {
	// first verifies the epoch
	height := bc.GetLatestHeight()
	if block.Epoch >= height {
		// verify the merkle proof
		prev := block.PrevBlockHash[0]
		isVerified, err := merkletree.VerifyProof(prev, false, block.MerkleProof, [][]byte{block.MerkleRootHash})
		if err != nil {
			log.Println(err)
		}
		if !isVerified {
			fmt.Println("verify error")
			return false, 0
		}

		// verify the state root
		if block.Epoch > 2 && block.Epoch == height+1 && !reflect.DeepEqual(block.StateRoot, root) {
			fmt.Println("state error")
			return false, 0
		}
		return true, height
	} else {
		fmt.Println("height error")
		return false, height
	}
}

// MappingChain determines which chain to be added
func (bc *Blockchain) MappingChain(blockHash []byte, con int) (int, []int) {
	var bitNumber int

	curCon := len(bc.preChainSets)
	preBitNum := math.Ceil(math.Log2(float64(curCon)))
	expBitNum := math.Ceil(math.Log2(float64(con)))
	chainID := utils.ConvertBinToDec(blockHash, int(expBitNum))

	if preBitNum > expBitNum {
		// n to m (n>m)
		bitNumber = int(expBitNum)
		edges := bc.seekEdges(blockHash, chainID, bitNumber)
		return chainID, edges
	}
	// n to m (n<=m)
	bitNumber = int(preBitNum)
	edges := bc.seekEdges2(blockHash, bitNumber)
	return chainID, edges
}

// seekEdges finds the previous blocks that the current block intends to connect with
// when the previous block concurrency is larger than the current block concurrency
func (bc *Blockchain) seekEdges(blockHash []byte, chainID, bitNumber int) []int {
	var edges []int

	if bc.height == 0 {
		for id := range bc.preChainSets {
			if id == chainID {
				edges = append(edges, id)
			}
		}
		return edges
	}

	for {
		// reduce one bit to find the same tail
		for id, blk := range bc.preChainSets {
			id2 := utils.ConvertBinToDec(blk.Tip, bitNumber)
			if id2 == chainID {
				edges = append(edges, id)
			}
		}

		if len(edges) > 0 {
			break
		}

		bitNumber--
		chainID = utils.ConvertBinToDec(blockHash, bitNumber)
	}

	return edges
}

// seekEdges finds the previous blocks that the current block intends to connect with
// when the previous block concurrency is smaller than the current block concurrency
func (bc *Blockchain) seekEdges2(blockHash []byte, bitNumber int) []int {
	var edges []int

	chainID := utils.ConvertBinToDec(blockHash, bitNumber)
	for id := range bc.preChainSets {
		if id == chainID {
			edges = append(edges, id)
		}
	}

	if len(edges) == 0 {
		for {
			bitNumber--
			chainID = utils.ConvertBinToDec(blockHash, bitNumber)
			// reduce one bit to find the same tail
			for id, blk := range bc.preChainSets {
				id2 := utils.ConvertBinToDec(blk.Tip, bitNumber)
				if id2 == chainID {
					edges = append(edges, id)
				}
			}

			if len(edges) > 0 {
				break
			}
		}
	}

	return edges
}

// RmDuplicated removes duplicate blocks with the same ID and keeps the one with the smallest ID
func (bc *Blockchain) RmDuplicated() {
	for id, set := range bc.curChainSets {
		if len(set) > 1 {
			var nonceMap = make(map[string]map[int]*types.Block)
			var sortedMap = make(map[string]*types.Block)
			var sorted []string

			for _, blk := range set {
				stringIDs := string(blk.BlockHash)
				nonce := blk.Nonce
				if _, ok := nonceMap[stringIDs]; !ok {
					temp := make(map[int]*types.Block)
					nonceMap[stringIDs] = temp
				}
				nonceMap[stringIDs][nonce] = blk
				sorted = append(sorted, stringIDs)
			}

			for hash, blks := range nonceMap {
				var nonces []int
				for nonce := range blks {
					nonces = append(nonces, nonce)
				}
				sort.Ints(nonces)
				blk := blks[nonces[0]]
				sortedMap[hash] = blk
			}

			sort.Strings(sorted)
			smallest := sortedMap[sorted[0]]
			bc.curChainSets[id] = []*types.Block{smallest}
		}
	}
}

func dbExists(dbFile string) bool {
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		return false
	}
	return true
}
