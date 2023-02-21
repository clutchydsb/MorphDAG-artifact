package p2p

import (
	"Occam/config"
	"Occam/core"
	"Occam/core/state"
	"Occam/core/types"
	"Occam/utils"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// Server defines the Occam server
type Server struct {
	BC              *core.Blockchain
	NodeID          string
	Address         string
	TxPool          *TxPool
	StateDB         *state.StateDB
	State           []byte
	Concurrency     int
	ProcessingQueue map[string][]*types.Transaction
	CompletedQueue  map[string][]*types.Transaction
	//StartSignalNum  int
	RunSignalNum int32
	StartOrRun   bool
}

// define the sending format of block
type block struct {
	From  string
	Block types.Block
}

// define the sending format of tx
type tx struct {
	From        string
	Transaction types.Transaction
}

var loadScale int
var rwMu sync.RWMutex

func InitializeServer(nodeID string) *Server {
	dbFile := fmt.Sprintf(config.DBfile2, nodeID)
	stateDB, err := state.NewState(dbFile, nil)
	if err != nil {
		log.Println("initialize failed")
		return nil
	}

	server := &Server{
		NodeID:          nodeID,
		Address:         nodeID,
		TxPool:          NewTxPool(),
		StateDB:         stateDB,
		State:           []byte{},
		Concurrency:     config.InitialConcurrency,
		ProcessingQueue: make(map[string][]*types.Transaction),
		CompletedQueue:  make(map[string][]*types.Transaction),
		//StartSignalNum:  0,
		RunSignalNum: 0,
		StartOrRun:   false,
	}

	return server
}

func (server *Server) CreateDAG() {
	bc := core.CreateBlockchain(server.NodeID, server.Address, server.Concurrency)
	server.BC = bc
	fmt.Println("OHIE initialized!")
}

// Run runs the Occam protocol circularly
func (server *Server) Run(cycles int) {
	defer server.BC.DB.Close()
	//server.sendStartSignal()
	//for !server.StartOrRun {
	//}
	fmt.Printf("Server %s starts\n", server.NodeID)

	// wait for 15 seconds for receiving sufficient transactions
	time.Sleep(15 * time.Second)

	for i := 0; i < cycles; i++ {
		var wg sync.WaitGroup
		start := time.Now()
		server.StartOrRun = false
		//if i > 0 {
		//	server.updateConcurrency()
		//}
		server.mineBlock(server.Concurrency)
		if i > 0 {
			// execute transactions in the last epoch
			wg.Add(1)
			go func() {
				defer wg.Done()
				server.processTxs()
			}()
		}
		var duration time.Duration
		for {
			duration = time.Since(start)
			if duration.Seconds() >= config.EpochTime {
				break
			}
		}
		// sync operation
		server.sendRunSignal()
		if i > 0 {
			wg.Wait()
		}
		for !server.StartOrRun {
		}
		loadScale = server.processTxPool()
		server.BC.UpdateEdges()
		server.BC.EnterNextEpoch()
		fmt.Printf("Epoch %d ends\n", server.BC.GetLatestHeight())
	}
}

// ProcessBlock processes received block and add it to the chain
func (server *Server) ProcessBlock(request []byte) {
	var payload block

	data := request[config.CommandLength:]
	err := json.Unmarshal(data, &payload)
	if err != nil {
		log.Panic(err)
	}

	blk := payload.Block
	fmt.Println("Recevied a new block!")

	blockHash := blk.BlockHash
	expBitNum := math.Ceil(math.Log2(float64(server.Concurrency)))
	chainID := utils.ConvertBinToDec(blockHash, int(expBitNum))
	stateRoot := server.getLatestState()
	isAdded := server.BC.AddBlock(&blk, chainID, stateRoot)

	if isAdded {
		fmt.Println("Added successfully!")
	} else {
		fmt.Println("Invalid block!")
	}
}

// ProcessTx processes the received new transaction
func (server *Server) ProcessTx(request []byte) {
	var payload tx

	data := request[config.CommandLength:]
	err := json.Unmarshal(data, &payload)
	if err != nil {
		log.Panic(err)
	}

	// store to the memory tx pool
	t := payload.Transaction
	t.SetStart()
	server.TxPool.pending.append(&t)
}

//// ProcessStartSignal processes the received start signal
//func (server *Server) ProcessStartSignal(request []byte) {
//	signal := bytesToCommand(request)
//	if len(signal) > 0 {
//		server.StartSignalNum++
//		fmt.Println(server.StartSignalNum)
//		if server.StartSignalNum == config.NodeNumber-1 {
//			server.StartOrRun = true
//		}
//	}
//}

// ProcessRunSignal processes the received run signal
func (server *Server) ProcessRunSignal(request []byte) {
	signal := bytesToCommand(request)
	if len(signal) > 0 {
		atomic.AddInt32(&server.RunSignalNum, 1)
		if atomic.LoadInt32(&server.RunSignalNum) >= config.NodeNumber-1 {
			server.StartOrRun = true
			// reset the number counter
			atomic.StoreInt32(&server.RunSignalNum, 0)
		}
	}
}

//// sendStartSignal broadcasts the start signal to the network
//func (server *Server) sendStartSignal() {
//	signal := commandToBytes("start")
//	err := SyncSignal(signal)
//	if err != nil {
//		log.Panic(err)
//	}
//}

// sendRunSignal broadcasts the run signal to the network
func (server *Server) sendRunSignal() {
	signal := commandToBytes("run")
	err := SyncSignal(signal)
	if err != nil {
		log.Panic(err)
	}
}

// mineBlock packages #size of transactions into a new block
func (server *Server) mineBlock(con int) {
	size := server.TxPool.RetrievePending()
	txs := server.TxPool.Pick(size)
	blk := server.BC.MineBlock(txs, con, server.State)
	if blk != nil {
		data := block{server.NodeID, *blk}
		bt, _ := json.Marshal(data)
		req := append(commandToBytes("block"), bt...)
		err := SyncBlock(req)
		if err != nil {
			log.Panic(err)
		}
	}
}

// updateConcurrency updates block concurrency according to the previous blocks
func (server *Server) updateConcurrency() {
	//avgSize := float64(loadScale) / float64(len(server.ProcessingQueue))
	var sum int
	preBlocks := server.BC.GetPreviousBlocks()
	for _, blk := range preBlocks {
		size := len(blk.Transactions)
		sum += size
	}
	avgSize := float64(sum) / float64(len(preBlocks))
	proportion := avgSize / float64(config.MaxBlockSize)

	if proportion >= 0.5 {
		concurrency := server.Concurrency * 2
		if concurrency < config.MaximumConcurrency {
			server.Concurrency = concurrency
		} else {
			server.Concurrency = config.MaximumConcurrency
		}
	} else {
		if server.Concurrency > 1 {
			server.Concurrency = server.Concurrency / 2
		}
	}
}

// processTxPool deletes packaged transactions in the transaction pool
func (server *Server) processTxPool() int {
	// remove duplicated blocks with the same id
	server.BC.RmDuplicated()
	server.ProcessingQueue = make(map[string][]*types.Transaction)
	blocks := server.BC.GetCurrentBlocks()

	for _, blks := range blocks {
		txs := blks[0].Transactions
		server.TxPool.DeleteTxs(txs)
	}

	load, processing := rmDuplicatedTxs(blocks)
	// mark the transactions appended to the DAG
	for id, txs := range processing {
		for _, t := range txs {
			t.SetEnd1()
		}
		server.ProcessingQueue[id] = txs
	}

	return load
}

// processTxs executes all transactions in all concurrent blocks
func (server *Server) processTxs() {
	// Process all received concurrent blocks in the previous epoch
	st := time.Now()
	server.CompletedQueue = make(map[string][]*types.Transaction)
	fmt.Printf("load scale is: %d\n", loadScale)
	executor := core.NewExecutor(server.StateDB, config.MaximumProcessors, loadScale)
	stateRoot := executor.Processing(server.ProcessingQueue)
	for id, txs := range server.ProcessingQueue {
		server.CompletedQueue[id] = txs
	}
	server.setState(stateRoot)
	end := time.Since(st)
	fmt.Printf("processing latency: %s\n", end)
}

// getLatestState gets latest state root (concurrent safe)
func (server *Server) getLatestState() []byte {
	rwMu.RLock()
	defer rwMu.RUnlock()
	stateRoot := server.State
	return stateRoot
}

// setState updates the state root (concurrent safe)
func (server *Server) setState(stateRoot []byte) {
	rwMu.Lock()
	defer rwMu.Unlock()
	server.State = stateRoot
}

// rmDuplicatedTxs deletes duplicate transactions for transaction execution
func rmDuplicatedTxs(blocks map[int][]*types.Block) (int, map[string][]*types.Transaction) {
	var sorted []int
	var load int
	var deleted = make(map[string]struct{})
	var processing = make(map[string][]*types.Transaction)

	for id := range blocks {
		sorted = append(sorted, id)
	}

	sort.Ints(sorted)

	for _, id := range sorted {
		var added []*types.Transaction
		blk := blocks[id]
		txs := blk[0].Transactions
		for _, t := range txs {
			iid := string(t.ID)
			if _, ok := deleted[iid]; !ok {
				added = append(added, t)
				deleted[iid] = struct{}{}
			}
		}
		processing[strconv.Itoa(id)] = added
		load += len(added)
	}

	return load, processing
}

func commandToBytes(command string) []byte {
	var newBytes [config.CommandLength]byte

	for i, c := range command {
		newBytes[i] = byte(c)
	}

	return newBytes[:]
}

func bytesToCommand(bytes []byte) string {
	var command []byte

	for _, b := range bytes {
		if b != 0x0 {
			command = append(command, b)
		}
	}

	return fmt.Sprintf("%s", command)
}
