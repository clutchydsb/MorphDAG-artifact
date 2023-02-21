package p2p

import (
	"Occam/core/types"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type RPCServer struct {
	RPC *Server
}

type SendBatchTxsCmd struct {
	Txs      []*types.Transaction
	Interval float64
}

type SendBatchTxsReply struct {
	Msg string
}

type GetBalanceCmd struct {
	Address string
}

type GetBalanceReply struct {
	Msg string
}

type ObserveCmd struct{}

type ObserveReply struct {
	Completed map[string][]*types.Transaction
}

func StartRPCServer(server *Server) *RPCServer {
	return &RPCServer{server}
}

//func (server *RPCServer) SendSignal(cmd SendSignalCmd, reply *SendSignalReply) error {
//	signal := commandToBytes("signal")
//	err := SyncData(signal)
//	if err != nil {
//		return err
//	}
//
//	server.RPC.Start = true
//	reply.Msg = "Signal sent success!"
//	return nil
//}

func (server *RPCServer) SendBatchTxs(cmd SendBatchTxsCmd, reply *SendBatchTxsReply) error {
	txs := cmd.Txs
	for _, t := range txs {
		// serialize tx data and broadcast to the network
		txData := tx{server.RPC.NodeID, *t}
		payload, _ := json.Marshal(txData)
		request := append(commandToBytes("tx"), payload...)
		err := SyncTx(request)
		if err != nil {
			return err
		}
		for k := 0; k < int(cmd.Interval); k++ {
			time.Sleep(time.Millisecond)
		}
	}

	reply.Msg = "Batch of TXs sent success!"
	return nil
}

func (server *RPCServer) GetBalance(cmd GetBalanceCmd, reply *GetBalanceReply) error {
	stateDB := server.RPC.StateDB
	addr := []byte(cmd.Address)
	bal := stateDB.GetBalance(addr)
	if bal == 0 {
		err := errors.New("cannot find the target account")
		return err
	}
	reply.Msg = fmt.Sprintf("Balance of '%s': %x\n", cmd.Address, bal)
	return nil
}

func (server *RPCServer) ObserveLatency(cmd ObserveCmd, reply *ObserveReply) error {
	reply.Completed = server.RPC.CompletedQueue
	if len(reply.Completed) == 0 {
		err := errors.New("empty queue")
		return err
	}
	return nil
}
