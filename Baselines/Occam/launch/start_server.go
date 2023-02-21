package main

import (
	"Occam/cli"
	"Occam/p2p"
	"Occam/utils"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"strconv"
	"time"
)

var nodeID string
var cmd cli.CMDServer
var dagServer *p2p.Server

func init() {
	nodeID = os.Getenv("NODE_ID")
	if nodeID == "" {
		fmt.Printf("NODE_ID env. var is not set!")
		os.Exit(1)
	}
	cmd.Run(nodeID)
}

// start the rpc server
func init() {
	dagServer = p2p.InitializeServer(nodeID)
	if !cmd.TxSender {
		// initialize DAG blockchain
		dagServer.CreateDAG()
	}

	// start the rpc server
	rpcServer := p2p.StartRPCServer(dagServer)
	err := rpc.Register(rpcServer)
	if err != nil {
		log.Fatal("Wrong format of service!", err)
	}

	rpc.HandleHTTP()

	listener, err2 := net.Listen("tcp", "localhost:"+strconv.Itoa(cmd.RPCPort))
	if err2 != nil {
		log.Fatal("Listen error: ", err2)
	}

	log.Printf("RPC server listening on port %d", cmd.RPCPort)
	go http.Serve(listener, nil)
}

// start the p2p connection
func main() {
	sk, err := utils.GetPrivateKey(cmd.PKFile)
	if err != nil {
		log.Fatal("Opening file error: ", err)
	}

	host, err := p2p.MakeHost(cmd.P2PPort, sk, cmd.FullAddrsPath)
	if err != nil {
		log.Fatal("Fail to build P2P host: ", err)
	}

	txHandler := dagServer.ProcessTx
	blkHandler := dagServer.ProcessBlock
	//signalHandler1 := dagServer.ProcessStartSignal
	signalHandler := dagServer.ProcessRunSignal

	if !cmd.TxSender {
		p2p.OpenP2PListen(cmd.Pid1, cmd.Pid2, cmd.Pid3, host, txHandler, blkHandler, signalHandler)
		log.Printf("Open %d port for p2p listening", cmd.P2PPort)
	}

	time.Sleep(15 * time.Second)
	//announce := "meet me here"
	//err = p2p.PeerDiscover(host, announce, cmd.Pid, txHandler, blkHandler, signalHandler)
	//if err != nil {
	//	log.Fatal("Fail to connect: ", err)
	//}
	ipaddrs, err := utils.ReadStrings(cmd.NodeFile)
	if err != nil {
		log.Fatal("Fail to read ip addresses: ", err)
	}

	for _, ip := range ipaddrs {
		// log.Printf("The next address to connect is: %s", ip)
		err = p2p.ConnectPeer(ip, cmd.Pid1, cmd.Pid2, cmd.Pid3, cmd.TxSender, host, txHandler, blkHandler, signalHandler)
		if err != nil {
			log.Fatal("Fail to connect: ", err)
		}
		//log.Printf("Successfully connect to the node: %s", ip)
	}
	fmt.Printf("Successfully connect to the connected peers, %s th node\n", nodeID)

	time.Sleep(10 * time.Second)

	if !cmd.TxSender {
		dagServer.Run(cmd.Cycles)
	} else {
		select {}
	}
}
