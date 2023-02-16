package types

import (
	"MorphDAG/utils"
	"bytes"
	"encoding/gob"
	"github.com/wealdtech/go-merkletree"
	"log"
	"time"
)

// Block represents a block in the blockchain
type Block struct {
	Timestamp      int64
	Transactions   []*Transaction
	MerkleRootHash []byte
	MerkleProof    *merkletree.Proof
	PrevBlockHash  [][]byte
	BlockHash      []byte
	TxRoot         []byte
	StateRoot      []byte
	Nonce          int
	Epoch          int
}

// NewBlock creates and returns Block
func NewBlock(transactions []*Transaction, rootHash, stateRoot []byte, height, con int) *Block {
	block := &Block{
		time.Now().Unix(),
		transactions,
		rootHash,
		new(merkletree.Proof),
		[][]byte{},
		[]byte{},
		[]byte{},
		stateRoot,
		0,
		height}
	pow := NewProofOfWork(block, con)
	nonce, hash := pow.Run(con)

	if nonce == 0 {
		return nil
	}

	block.BlockHash = hash[:]
	block.TxRoot = block.HashTransactions()
	block.Nonce = nonce

	return block
}

// NewGenesisBlock creates and returns genesis Block
func NewGenesisBlock(coinbase *Transaction, num int) *Block {
	data := utils.IntToHex(int64(num*1000 + 1000))

	return &Block{
		time.Now().UnixNano(),
		[]*Transaction{coinbase},
		[]byte{},
		new(merkletree.Proof),
		[][]byte{},
		data,
		[]byte{},
		[]byte{},
		0,
		0,
	}
}

// HashTransactions returns a hash of the transactions in the block
func (b *Block) HashTransactions() []byte {
	var transactions [][]byte

	for _, tx := range b.Transactions {
		transactions = append(transactions, tx.Serialize())
	}

	mTree, err := merkletree.New(transactions)
	if err != nil {
		log.Panic(err)
	}

	root := mTree.Root()
	return root
}

// Serialize serializes the block
func (b *Block) Serialize() []byte {
	var result bytes.Buffer
	encoder := gob.NewEncoder(&result)

	err := encoder.Encode(b)
	if err != nil {
		log.Panic(err)
	}

	return result.Bytes()
}

// DeserializeBlock deserializes a block
func DeserializeBlock(d []byte) *Block {
	var block Block

	decoder := gob.NewDecoder(bytes.NewReader(d))
	err := decoder.Decode(&block)
	if err != nil {
		log.Panic(err)
	}

	return &block
}
