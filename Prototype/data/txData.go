package main

import (
	"MorphDAG/core/types"
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/DarcyWep/morph-txs"
	"io"
	"io/ioutil"
	"log"
	"math/big"
	"math/rand"
	"os"
	"time"
)

type Records struct {
	Rds []Record `json:"RECORDS"`
}

type Record struct {
	Hash        string      `json:"hash"`
	BlockHash   string      `json:"blockHash"`
	BlockNumber string      `json:"blockNumber"`
	Info        Transaction `json:"info"`
}

type Transfer struct {
	From  Balance `json:"from"`
	To    Balance `json:"to"`
	Type  uint8   `json:"type"`
	Nonce uint8   `json:"nonce"`
}

type Balance struct {
	Address  string  `json:"address"`
	Value    float64 `json:"value"`
	BeforeTx float64 `json:"beforeTx"`
	AfterTx  float64 `json:"afterTx"`
}

type Transaction struct {
	Count                uint8      `json:"count"`
	AccessList           string     `json:"accessList"`
	Transfer             []Transfer `json:"balance"`
	BlockHash            string     `json:"blockHash"`
	BlockNumber          string     `json:"blockNumber"`
	ChainId              string     `json:"chainId"`
	From                 string     `json:"from"`
	Gas                  string     `json:"gas"`
	GasPrice             string     `json:"gasPrice"`
	Hash                 string     `json:"hash"`
	Input                string     `json:"input"`
	MaxFeePerGas         string     `json:"maxFeePerGas"`
	MaxPriorityFeePerGas string     `json:"maxPriorityFeePerGas"`
	Nonce                string     `json:"nonce"`
	R                    string     `json:"r"`
	S                    string     `json:"s"`
	To                   string     `json:"to"`
	TransactionIndex     string     `json:"transactionIndex"`
	Type                 string     `json:"type"`
	V                    string     `json:"v"`
	Value                string     `json:"value"`
}

func main() {
	var blkNum int
	flag.IntVar(&blkNum, "b", 10000, "specify the number of blocks to fetch. defaults to 10000.")
	CreateBatchTxs("EthTxs.txt", blkNum)
}

// retrieve tx data by json file
func ReadJsonFile(file string) *Records {
	jsonFile, err := os.Open(file)
	if err != nil {
		fmt.Println("error opening json file")
		return nil
	}
	defer jsonFile.Close()

	jsonData, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		fmt.Println("error reading json file")
		return nil
	}

	var record Records
	err = json.Unmarshal(jsonData, &record)
	if err != nil {
		fmt.Println("error parsing json file")
		return nil
	}

	return &record
}

// retrieve tx data by json file
func (rds *Records) CreateBatchTxs(txNum int) []*types.Transaction {
	var txs []*types.Transaction
	records := rds.Rds[:txNum]

	for _, rd := range records {
		txInfo := rd.Info
		from := []byte(txInfo.From)
		to := []byte(txInfo.To)
		trans := txInfo.Transfer
		payload := createPayloads2(trans)
		tx := types.NewTransaction(0, from, to, payload)
		txs = append(txs, tx)
	}

	return txs
}

// CreateBatchTxs retrieves tx data by querying database and writes to file
func CreateBatchTxs(filename string, num int) {
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}
	defer file.Close()
	w := bufio.NewWriter(file)

	// open sql server connection
	morph.SetHost("202.114.7.176")
	err = morph.OpenSqlServers()
	if err != nil {
		log.Panic(err)
	}
	defer morph.CloseSqlServers()

	// starting block number
	blockNumber := 8200000

	for i := blockNumber; i < (blockNumber + num); i++ {
		mtxs := morph.GetTxsByBlockNumber(big.NewInt(int64(i)))
		for _, tx := range mtxs {
			trans := tx.Transfer
			payload := createPayloads(trans)
			t := types.NewTransaction(0, []byte(tx.From), []byte(tx.To), payload)
			txdata, _ := json.Marshal(t)
			_, err = w.WriteString(fmt.Sprintf("%s\n", string(txdata)))
			if err != nil {
				log.Printf("error: %v\n", err)
			}
		}
		w.Flush()
	}
}

// ReadEthTxsFile reads tx data from file and deserializes them (for test)
func ReadEthTxsFile(readName string) []*types.Transaction {
	var txs []*types.Transaction

	file, err := os.Open(readName)
	if err != nil {
		log.Panic("Read error: ", err)
	}
	defer file.Close()

	r := bufio.NewReader(file)

	for {
		txdata, err := r.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break
			} else {
				log.Panic(err)
			}
		}

		var tx types.Transaction
		err = json.Unmarshal(txdata, &tx)
		if err != nil {
			log.Panic(err)
		}

		txs = append(txs, &tx)
	}

	return txs
}

func createPayloads(trans []*morph.MorphTransfer) types.Payload {
	var payload = make(map[string][]*types.RWSet)

	for _, tran := range trans {
		switch tran.Type {
		case 1:
			rand.Seed(time.Now().UnixNano())
			random := rand.Intn(3)
			if random == 0 {
				// read-write (withdraw)
				r := rand.Intn(20)
				rw1 := &types.RWSet{Label: "r", Addr: []byte("1")}
				rw2 := &types.RWSet{Label: "w", Addr: []byte("1"), Value: int64(-r)}
				addr1 := tran.To
				payload[addr1] = append(payload[addr1], rw1, rw2)
				rw3 := &types.RWSet{Label: "iw", Addr: []byte("0"), Value: int64(r)}
				addr2 := tran.From
				payload[addr2] = append(payload[addr2], rw3)
			} else if random == 1 {
				// only read (query)
				rw := &types.RWSet{Label: "r", Addr: []byte("0")}
				addr := tran.To
				payload[addr] = append(payload[addr], rw)
			} else {
				// incremental write
				r := rand.Intn(20)
				rw := &types.RWSet{Label: "iw", Addr: []byte("0"), Value: int64(r)}
				addr := tran.To
				payload[addr] = append(payload[addr], rw)
			}
		case 2:
			// create contract
			rw := &types.RWSet{Label: "w", Addr: []byte("0"), Value: int64(200000)}
			addr := tran.To
			payload[addr] = append(payload[addr], rw)
		case 3:
			// transfer transaction fee to the miner
			r := rand.Intn(10)
			rw := &types.RWSet{Label: "iw", Addr: []byte("0"), Value: int64(r)}
			addr := tran.To
			payload[addr] = append(payload[addr], rw)
		case 4:
			// return gas fee to the caller
			r := rand.Intn(5)
			rw := &types.RWSet{Label: "iw", Addr: []byte("1"), Value: int64(r)}
			addr := tran.To
			payload[addr] = append(payload[addr], rw)
		case 5:
			// withhold transaction fee from the caller
			r := rand.Intn(10)
			rw1 := &types.RWSet{Label: "r", Addr: []byte("0")}
			rw2 := &types.RWSet{Label: "w", Addr: []byte("0"), Value: int64(-r)}
			addr := tran.From
			payload[addr] = append(payload[addr], rw1, rw2)
		case 6:
			// destroy the contract
			rw := &types.RWSet{Label: "w", Addr: []byte("2"), Value: int64(0)}
			addr := tran.To
			payload[addr] = append(payload[addr], rw)
		case 7, 8:
			// mining reward
			r := rand.Intn(100)
			rw := &types.RWSet{Label: "iw", Addr: []byte("1"), Value: int64(r)}
			addr := tran.To
			payload[addr] = append(payload[addr], rw)
		default:
			continue
		}
	}

	return payload
}

func createPayloads2(trans []Transfer) types.Payload {
	var payload = make(map[string][]*types.RWSet)

	for _, tran := range trans {
		switch tran.Type {
		case 1:
			rand.Seed(time.Now().UnixNano())
			random := rand.Intn(3)
			if random == 0 {
				// only read
				rw := &types.RWSet{Label: "r", Addr: []byte("0")}
				addr := tran.To.Address
				payload[addr] = append(payload[addr], rw)
			} else if random == 1 {
				// only write
				r := rand.Intn(20)
				rw := &types.RWSet{Label: "w", Addr: []byte("0"), Value: int64(r)}
				addr := tran.To.Address
				payload[addr] = append(payload[addr], rw)
			} else {
				// read-write
				rw1 := &types.RWSet{Label: "r", Addr: []byte("0")}
				r := rand.Intn(20)
				rw2 := &types.RWSet{Label: "w", Addr: []byte("0"), Value: int64(-r)}
				addr := tran.To.Address
				payload[addr] = append(payload[addr], rw1, rw2)
			}
		case 2:
			rw := &types.RWSet{Label: "w", Addr: []byte("1"), Value: 0}
			addr := tran.To.Address
			payload[addr] = append(payload[addr], rw)
		case 3:
			r := rand.Intn(10)
			rw := &types.RWSet{Label: "w", Addr: []byte("1"), Value: int64(r)}
			addr := tran.To.Address
			payload[addr] = append(payload[addr], rw)
		case 4:
			r := rand.Intn(5)
			rw := &types.RWSet{Label: "w", Addr: []byte("1"), Value: int64(r)}
			addr := tran.To.Address
			payload[addr] = append(payload[addr], rw)
		case 5:
			rw1 := &types.RWSet{Label: "r", Addr: []byte("1")}
			r := rand.Intn(10)
			rw2 := &types.RWSet{Label: "w", Addr: []byte("1"), Value: int64(-r)}
			addr := tran.From.Address
			payload[addr] = append(payload[addr], rw1, rw2)
		case 7, 8:
			r := rand.Intn(100)
			rw := &types.RWSet{Label: "w", Addr: []byte("0"), Value: int64(r)}
			addr := tran.To.Address
			payload[addr] = append(payload[addr], rw)
		default:
			continue
		}
	}

	return payload
}
