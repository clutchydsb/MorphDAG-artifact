package main

import (
	"MorphDAG/core"
	"MorphDAG/core/state"
	"MorphDAG/core/tp"
	"MorphDAG/core/types"
	"MorphDAG/nezha"
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/chinuy/zipf"
	"io"
	"log"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const dbFile1 = "../data/Morph_Test"
const dbFile2 = "../data/Morph_Test2"
const dbFile3 = "../data/Morph_Test3"

func main() {
	var addrNum uint64
	var blkNum int
	var skew float64
	var frequency int
	var ratio int

	flag.Uint64Var(&addrNum, "a", 10000, "specify address number to use. defaults to 10000.")
	flag.IntVar(&blkNum, "b", 16, "specify transaction number to use. defaults to 5.")
	flag.Float64Var(&skew, "s", 1.2, "specify skew to use. defaults to 0.2.")
	flag.IntVar(&frequency, "f", 20, "specify the frequency threshold to identify hot accounts. defaults to 5.")
	flag.IntVar(&ratio, "r", 0, "specify the read-write ratio. defaults to 3.")
	flag.Parse()

	//data.CreateBatchTxs("./data/EthTxs3.txt", 30000)
	//txs := data.ReadEthTxsFile("./data/EthTxs2.txt")
	//fmt.Println(len(txs))

	//runtime.GOMAXPROCS(runtime.NumCPU())

	//file, err := os.Open("../data/EthTxs4.txt")
	file, err := os.Open("./data/EthTxs4.txt")
	if err != nil {
		log.Panic("Read error: ", err)
	}
	defer file.Close()
	r := bufio.NewReader(file)

	var ttxs = make(map[string][]*types.Transaction)
	for i := 0; i < 200; i++ {
		var txs []*types.Transaction
		for j := 0; j < 500; j++ {
			var tx types.Transaction
			txdata, err2 := r.ReadBytes('\n')
			if err2 != nil {
				if err2 == io.EOF {
					break
				}
				log.Panic(err2)
			}
			err2 = json.Unmarshal(txdata, &tx)
			if err2 != nil {
				log.Panic(err)
			}
			txs = append(txs, &tx)
		}
		ttxs[strconv.Itoa(i)] = txs
	}

	var accesses = make(map[string]int)
	var accessList []int

	for _, txs := range ttxs {
		for _, tx := range txs {
			load := tx.Data()
			for addr := range load {
				accesses[addr]++
			}
		}
	}

	for _, num := range accesses {
		accessList = append(accessList, num)
	}

	sort.Sort(sort.Reverse(sort.IntSlice(accessList)))

	var sum int
	for _, acc := range accessList {
		sum += acc
	}

	fmt.Println(accessList)

	var result []int
	var res int
	for i := 0; i < 100; i++ {
		if i > 0 {
			res = result[i-1]
		}
		if i < 99 {
			for j := i * 631; j < (i+1)*631; j++ {
				res += accessList[j]
			}
		} else {
			for j := 99 * 631; j < 63122; j++ {
				res += accessList[j]
			}
		}
		result = append(result, res)
	}

	file2, err3 := os.OpenFile("fraction.txt", os.O_RDWR|os.O_CREATE|os.O_TRUNC|os.O_APPEND, 0666)
	if err3 != nil {
		fmt.Printf("error: %v\n", err3)
	}
	defer file2.Close()
	w := bufio.NewWriter(file2)

	for _, v := range result {
		frac := 100 * (float64(v) / float64(sum))
		_, err4 := w.WriteString(fmt.Sprintf("%.2f\n", frac))
		if err4 != nil {
			log.Printf("error: %v\n", err4)
		}
	}
	w.Flush()

	//txs := CreateBatchTxs(blkNum, ratio, addrNum, skew)

	//runTestMorphDAG(ttxs, blkNum, frequency, dbFile1)
	//runTestSerial(ttxs, blkNum, dbFile2)
	//runTestNezha(ttxs, blkNum, dbFile3)
}

func runTestMorphDAG(txs map[string][]*types.Transaction, blkNum, frequency int, dbFile string) {
	statedb, _ := state.NewState(dbFile, nil)
	load := blkNum * 500
	executor := core.NewExecutor(statedb, 50000, load)
	executor.Processing(txs, frequency)
}

func runTestSerial(txs map[string][]*types.Transaction, blkNum int, dbFile string) {
	statedb, _ := state.NewState(dbFile, nil)
	load := blkNum * 500
	executor := core.NewExecutor(statedb, 1, load)
	executor.SerialProcessing(txs)
}

func runTestNezha(txs map[string][]*types.Transaction, blkNum int, dbFile string) {
	statedb, _ := state.NewState(dbFile, nil)
	statedb.BatchCreateObjects(txs)

	start := time.Now()
	var wg sync.WaitGroup
	for _, blk := range txs {
		for _, tx := range blk {
			wg.Add(1)
			go func(t *types.Transaction) {
				defer wg.Done()
				msg := t.AsMessage()
				tp.MimicConcurrentExecution(statedb, msg)
			}(tx)
		}
	}
	wg.Wait()

	input := CreateNezhaRWNodes(txs)

	var mapping = make(map[string]*types.Transaction)
	for blk := range txs {
		txSets := txs[blk]
		for _, tx := range txSets {
			id := string(tx.ID)
			mapping[id] = tx
		}
	}

	queueGraph := nezha.CreateGraph(input)
	sequence := queueGraph.QueuesSort()
	commitOrder := queueGraph.DeSS(sequence)
	tps := blkNum*500 - queueGraph.GetAbortedNums()
	fmt.Println(tps)

	var keys []int
	for seq := range commitOrder {
		keys = append(keys, int(seq))
	}
	sort.Ints(keys)

	for _, n := range keys {
		for _, group := range commitOrder[int32(n)] {
			if len(group) > 0 {
				wg.Add(1)
				node := group[0]
				tx := mapping[node.TransInfo.ID]
				go func() {
					defer wg.Done()
					msg := tx.AsMessage()
					err := tp.ApplyMessageForNezha(statedb, msg)
					if err != nil {
						panic(err)
					}
				}()
			}
		}
		wg.Wait()
	}

	statedb.Commit()
	duration := time.Since(start)
	fmt.Printf("Time of processing transactions is: %s\n", duration)
	statedb.Reset()
}

func CreateBatchTxs(blkNum, ratio int, addrNum uint64, skew float64) map[string][]*types.Transaction {
	var txs = make(map[string][]*types.Transaction)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	z := zipf.NewZipf(r, skew, addrNum)

	selectFunc := []string{"almagate", "updateBalance", "updateSaving", "sendPayment", "writeCheck", "getBalance"}

	for i := 0; i < blkNum; i++ {
		for j := i * 500; j < (i+1)*500; j++ {
			rand.Seed(time.Now().UnixNano())
			random := rand.Intn(10)

			// read-write ratio
			var function string
			if random <= ratio {
				function = selectFunc[5]
			} else {
				random2 := rand.Intn(5)
				function = selectFunc[random2]
			}

			var duplicated = make(map[string]struct{})
			var addr1, addr2, addr3, addr4 string

			addr1 = strconv.FormatUint(z.Uint64(), 10)
			duplicated[addr1] = struct{}{}

			for {
				addr2 = strconv.FormatUint(z.Uint64(), 10)
				if _, ok := duplicated[addr2]; !ok {
					duplicated[addr2] = struct{}{}
					break
				}
			}

			for {
				addr3 = strconv.FormatUint(z.Uint64(), 10)
				if _, ok := duplicated[addr3]; !ok {
					duplicated[addr3] = struct{}{}
					break
				}
			}

			for {
				addr4 = strconv.FormatUint(z.Uint64(), 10)
				if _, ok := duplicated[addr4]; !ok {
					break
				}
			}

			payload := GeneratePayload(function, addr1, addr2, addr3, addr4)
			newTx := types.NewTransaction(0, []byte("A"), []byte("K"), payload)
			txs[strconv.Itoa(i)] = append(txs[strconv.Itoa(i)], newTx)
		}
	}

	return txs
}

func CreateTxsTest() map[int64][]*types.Transaction {
	var txs = make(map[int64][]*types.Transaction)

	var payload1 = make(map[string][]*types.RWSet)
	rw1 := &types.RWSet{Label: "r", Addr: []byte("0")}
	rw2 := &types.RWSet{Label: "w", Addr: []byte("1"), Value: 3}
	payload1["A"] = append(payload1["A"], rw1, rw2)
	tx1 := types.NewTransaction(0, []byte("K"), []byte("A"), payload1)

	var payload2 = make(map[string][]*types.RWSet)
	rw1 = &types.RWSet{Label: "r", Addr: []byte("4")}
	rw2 = &types.RWSet{Label: "w", Addr: []byte("5"), Value: 3}
	payload2["C"] = append(payload2["C"], rw1)
	payload2["D"] = append(payload2["D"], rw2)
	tx2 := types.NewTransaction(0, []byte("K"), []byte("A"), payload2)

	txs[1] = append(txs[1], tx1, tx2)

	var payload3 = make(map[string][]*types.RWSet)
	rw1 = &types.RWSet{Label: "r", Addr: []byte("3")}
	rw2 = &types.RWSet{Label: "w", Addr: []byte("7"), Value: 5}
	payload3["C"] = append(payload3["C"], rw1)
	payload3["B"] = append(payload3["B"], rw2)
	tx3 := types.NewTransaction(0, []byte("K"), []byte("A"), payload3)

	var payload4 = make(map[string][]*types.RWSet)
	rw1 = &types.RWSet{Label: "r", Addr: []byte("5")}
	rw2 = &types.RWSet{Label: "w", Addr: []byte("7"), Value: 4}
	payload4["B"] = append(payload4["B"], rw1, rw2)
	tx4 := types.NewTransaction(0, []byte("K"), []byte("A"), payload4)

	txs[2] = append(txs[2], tx3, tx4)

	var payload5 = make(map[string][]*types.RWSet)
	rw1 = &types.RWSet{Label: "r", Addr: []byte("5")}
	rw2 = &types.RWSet{Label: "w", Addr: []byte("5"), Value: 7}
	payload5["D"] = append(payload3["D"], rw1, rw2)
	tx5 := types.NewTransaction(0, []byte("K"), []byte("A"), payload5)

	var payload6 = make(map[string][]*types.RWSet)
	rw1 = &types.RWSet{Label: "r", Addr: []byte("3")}
	payload6["A"] = append(payload6["A"], rw1)
	tx6 := types.NewTransaction(0, []byte("K"), []byte("A"), payload6)

	var payload7 = make(map[string][]*types.RWSet)
	rw1 = &types.RWSet{Label: "w", Addr: []byte("1"), Value: 2}
	rw2 = &types.RWSet{Label: "w", Addr: []byte("3"), Value: 2}
	payload7["A"] = append(payload7["A"], rw1)
	payload7["B"] = append(payload7["B"], rw2)
	tx7 := types.NewTransaction(0, []byte("K"), []byte("A"), payload7)

	txs[3] = append(txs[3], tx5, tx6, tx7)

	var payload8 = make(map[string][]*types.RWSet)
	rw1 = &types.RWSet{Label: "r", Addr: []byte("2")}
	rw2 = &types.RWSet{Label: "w", Addr: []byte("2"), Value: 3}
	payload8["E"] = append(payload8["E"], rw1)
	payload8["E"] = append(payload8["E"], rw2)
	tx8 := types.NewTransaction(0, []byte("K"), []byte("A"), payload8)

	var payload9 = make(map[string][]*types.RWSet)
	rw1 = &types.RWSet{Label: "r", Addr: []byte("4")}
	rw2 = &types.RWSet{Label: "w", Addr: []byte("7"), Value: -10}
	payload9["E"] = append(payload9["E"], rw1)
	payload9["F"] = append(payload9["F"], rw2)
	tx9 := types.NewTransaction(0, []byte("K"), []byte("A"), payload9)

	var payload10 = make(map[string][]*types.RWSet)
	rw1 = &types.RWSet{Label: "w", Addr: []byte("1"), Value: 10}
	rw2 = &types.RWSet{Label: "w", Addr: []byte("1"), Value: -5}
	payload10["G"] = append(payload10["G"], rw1)
	payload10["H"] = append(payload10["H"], rw2)
	tx10 := types.NewTransaction(0, []byte("K"), []byte("A"), payload10)

	txs[4] = append(txs[4], tx8, tx9, tx10)
	return txs
}

func GeneratePayload(funcName string, addr1, addr2, addr3, addr4 string) types.Payload {
	var payload = make(map[string][]*types.RWSet)
	switch funcName {
	// addr1 & addr3 --> savingStore, addr2 & addr4 --> checkingStore
	case "almagate":
		rw1 := &types.RWSet{Label: "r", Addr: []byte("0")}
		payload[addr1] = append(payload[addr1], rw1)
		rw2 := &types.RWSet{Label: "w", Addr: []byte("0"), Value: 1000}
		payload[addr2] = append(payload[addr2], rw2)
		rw3 := &types.RWSet{Label: "r", Addr: []byte("0")}
		payload[addr4] = append(payload[addr4], rw3)
		rw4 := &types.RWSet{Label: "iw", Addr: []byte("0"), Value: 10}
		payload[addr3] = append(payload[addr3], rw4)
	case "getBalance":
		rw1 := &types.RWSet{Label: "r", Addr: []byte("0")}
		payload[addr1] = append(payload[addr1], rw1)
		rw2 := &types.RWSet{Label: "r", Addr: []byte("0")}
		payload[addr2] = append(payload[addr2], rw2)
	case "updateBalance":
		rw1 := &types.RWSet{Label: "iw", Addr: []byte("0"), Value: 10}
		payload[addr2] = append(payload[addr2], rw1)
	case "updateSaving":
		rw1 := &types.RWSet{Label: "iw", Addr: []byte("0"), Value: 10}
		payload[addr1] = append(payload[addr1], rw1)
	case "sendPayment":
		rw1 := &types.RWSet{Label: "r", Addr: []byte("0")}
		rw2 := &types.RWSet{Label: "w", Addr: []byte("0"), Value: -10}
		payload[addr2] = append(payload[addr2], rw1, rw2)
		rw3 := &types.RWSet{Label: "iw", Addr: []byte("0"), Value: 10}
		payload[addr4] = append(payload[addr4], rw3)
	case "writeCheck":
		rw1 := &types.RWSet{Label: "r", Addr: []byte("0")}
		payload[addr1] = append(payload[addr1], rw1)
		rw2 := &types.RWSet{Label: "iw", Addr: []byte("0"), Value: 5}
		payload[addr2] = append(payload[addr2], rw2)
	default:
		fmt.Println("Invalid inputs")
		return nil
	}
	return payload
}

func CreateNezhaRWNodes(txs map[string][]*types.Transaction) [][]*nezha.RWNode {
	var input [][]*nezha.RWNode
	for blk := range txs {
		txSets := txs[blk]
		for _, tx := range txSets {
			var rAddr, wAddr, rValue, wValue [][]byte
			id := string(tx.ID)
			ts := tx.Header.Timestamp
			payload := tx.Data()
			for addr := range payload {
				rwSet := payload[addr]
				for _, rw := range rwSet {
					if strings.Compare(rw.Label, "r") == 0 && !isExist(rAddr, []byte(addr)) {
						rAddr = append(rAddr, []byte(addr))
						rValue = append(rValue, []byte("10"))
					} else if strings.Compare(rw.Label, "w") == 0 && !isExist(wAddr, []byte(addr)) {
						wAddr = append(wAddr, []byte(addr))
						wValue = append(wValue, []byte("10"))
					} else if strings.Compare(rw.Label, "iw") == 0 && !isExist(wAddr, []byte(addr)) {
						wAddr = append(wAddr, []byte(addr))
						wValue = append(wValue, []byte("10"))
					}
				}
			}
			rwNodes := nezha.CreateRWNode(id, uint32(ts), rAddr, rValue, wAddr, wValue)
			input = append(input, rwNodes)
		}
	}
	return input
}

func isExist(data [][]byte, addr []byte) bool {
	for _, d := range data {
		if strings.Compare(string(addr), string(d)) == 0 {
			return true
		}
	}
	return false
}
