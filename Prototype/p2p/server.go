package p2p

import (
	"MorphDAG/config"
	"MorphDAG/core"
	"MorphDAG/core/state"
	"MorphDAG/core/types"
	"MorphDAG/utils"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"runtime"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// Server defines the MorphDAG server
type Server struct {
	BC              *core.Blockchain
	NodeID          string
	Address         string
	TxPool          *TxPool
	StateDB         *state.StateDB
	State           []byte
	NodeNumber      int
	Concurrency     int
	ProcessingQueue map[string][]*types.Transaction
	CompletedQueue  map[string][]*types.Transaction
	RunSignalNum    int32
	StartOrRun      bool
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

func InitializeServer(nodeID string, nodeNumber int) *Server {
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
		NodeNumber:      nodeNumber,
		Concurrency:     config.InitialConcurrency,
		ProcessingQueue: make(map[string][]*types.Transaction),
		CompletedQueue:  make(map[string][]*types.Transaction),
		RunSignalNum:    0,
		StartOrRun:      false,
	}

	return server
}

func (server *Server) CreateDAG() {
	bc := core.CreateBlockchain(server.NodeID, server.Address, server.Concurrency)
	server.BC = bc
	fmt.Println("MorphDAG initialized!")
}

// Run runs the MorphDAG protocol circularly
func (server *Server) Run(cycles int) {
	defer server.BC.DB.Close()
	fmt.Printf("Server %s starts\n", server.NodeID)

	// wait for 15 seconds for receiving sufficient transactions
	time.Sleep(15 * time.Second)

	for i := 0; i < cycles; i++ {
		var wg sync.WaitGroup
		start := time.Now()
		server.StartOrRun = false
		server.updateConcurrency()
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
//		atomic.AddInt32(&server.StartSignalNum, 1)
//		if atomic.LoadInt32(&server.StartSignalNum) >= config.NodeNumber-1 {
//			server.StartOrRun = true
//			// reset the number counter
//			atomic.StoreInt32(&server.StartSignalNum, 0)
//		}
//	}
//}

// ProcessRunSignal processes the received run signal
func (server *Server) ProcessRunSignal(request []byte) {
	signal := bytesToCommand(request)
	if len(signal) > 0 {
		atomic.AddInt32(&server.RunSignalNum, 1)
		if atomic.LoadInt32(&server.RunSignalNum) >= int32(server.NodeNumber-1) {
			server.StartOrRun = true
			// reset the number counter
			atomic.StoreInt32(&server.RunSignalNum, 0)
		}
	}
}

//// sendStartSignal broadcasts the start signal to the network
//func (server *Server) sendStartSignal() {
//	signal := commandToBytes("start")
//	err := SyncSignal2(signal)
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
	server.TxPool.RetrievePending()
	txs := server.TxPool.Pick(config.BlockSize)
	blk := server.BC.MineBlock(txs, con, server.NodeNumber, server.State)
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

// updateConcurrency updates block concurrency according to the analysis result
func (server *Server) updateConcurrency() {
	scale := server.TxPool.GetScale()
	curCon := core.CalculateConcurrency(scale)
	server.Concurrency = curCon
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
	runtime.GOMAXPROCS(runtime.NumCPU())
	server.CompletedQueue = make(map[string][]*types.Transaction)
	fmt.Printf("load scale is: %d\n", loadScale)
	executor := core.NewExecutor(server.StateDB, config.MaximumProcessors, loadScale)
	stateRoot := executor.Processing(server.ProcessingQueue, config.Frequency)
	for id, txs := range server.ProcessingQueue {
		server.CompletedQueue[id] = txs
	}
	server.setState(stateRoot)
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
