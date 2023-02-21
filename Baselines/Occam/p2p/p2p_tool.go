package p2p

import (
	"Occam/config"
	"Occam/utils"
	"bufio"
	"context"
	"fmt"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/protocol"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
	"github.com/multiformats/go-multiaddr"
	"log"
	"os"
	"sync"
)

// maintain a list of all the P2P connection to write data to
var txWriterSet []*bufio.Writer
var blockWriterSet []*bufio.Writer
var signalWriterSet []*bufio.Writer

// maintain a list of bootstrap peers
var defaultBootstrapPeers []multiaddr.Multiaddr

// TxHandler a function to deal with tx received from the P2P connection
type TxHandler func(request []byte)

// BlkHandler a function to deal with block received from the P2P connection
type BlkHandler func(request []byte)

//// StartSignalHandler a function to deal with start signal received from the P2P connection
//type StartSignalHandler func(request []byte)

// RunSignalHandler a function to deal with run signal received from the P2P connection
type RunSignalHandler func(request []byte)

func init() {
	for _, s := range []string{
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt",
		//"/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
	} {
		ma, err := multiaddr.NewMultiaddr(s)
		if err != nil {
			panic(err)
		}
		defaultBootstrapPeers = append(defaultBootstrapPeers, ma)
	}
}

// MakeHost creates a LibP2P host listening on the given port
func MakeHost(listenPort int, pk crypto.PrivKey, addrsPath string) (core.Host, error) {
	sourceMultiAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", listenPort))

	basicHost, err := libp2p.New(
		libp2p.ListenAddrs(sourceMultiAddr),
		libp2p.Identity(pk),
		libp2p.ForceReachabilityPublic(),
		libp2p.NATPortMap(),
		libp2p.EnableRelay(),
	)
	if err != nil {
		return nil, err
	}

	hostAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ipfs/%s", basicHost.ID().String()))
	if err != nil {
		return nil, err
	}

	var fullAddrs []string
	for i, addr := range basicHost.Addrs() {
		fullAddr := addr.Encapsulate(hostAddr).String()
		log.Printf("The %d th fullAddr: %s\n", i, fullAddr)
		fullAddrs = append(fullAddrs, fullAddr)
	}

	if utils.FileExists(addrsPath) {
		log.Printf("File %s already exists!", addrsPath)
	} else {
		f, err := os.Create(addrsPath)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		for _, addr := range fullAddrs {
			_, err = f.WriteString(addr + "\n")
			if err != nil {
				return nil, err
			}
		}
	}

	return basicHost, nil
}

// OpenP2PListen opens a listening for connection
func OpenP2PListen(pid1, pid2, pid3 string, host core.Host, txHandler TxHandler, blkHandler BlkHandler, signalHandler RunSignalHandler) {
	stream1 := func(s core.Stream) {
		log.Println("Receive a connection 1")
		// create a tx stream writer for each connection
		txWriter := bufio.NewWriter(s)
		txWriterSet = append(txWriterSet, txWriter)
		go readDataFromP2P(bufio.NewReader(s), txHandler, blkHandler, signalHandler)
	}

	stream2 := func(s core.Stream) {
		log.Println("Receive a connection 2")
		// create a block stream writer for each connection
		blkWriter := bufio.NewWriter(s)
		blockWriterSet = append(blockWriterSet, blkWriter)
		go readDataFromP2P(bufio.NewReader(s), txHandler, blkHandler, signalHandler)
	}

	stream3 := func(s core.Stream) {
		log.Println("Receive a connection 3")
		// create a signal stream writer for each connection
		signalWriter := bufio.NewWriter(s)
		signalWriterSet = append(signalWriterSet, signalWriter)
		go readDataFromP2P(bufio.NewReader(s), txHandler, blkHandler, signalHandler)
	}

	host.SetStreamHandler(protocol.ID(pid1), stream1)
	host.SetStreamHandler(protocol.ID(pid2), stream2)
	host.SetStreamHandler(protocol.ID(pid3), stream3)
}

// readDataFromP2P reads a transaction or a block from P2P connection and deals with it
func readDataFromP2P(reader *bufio.Reader, txHandler TxHandler, blkHandler BlkHandler, signalHandler RunSignalHandler) {
	for {
		str, err := reader.ReadString('\n')
		if err != nil {
			log.Panic(err)
		}

		if len(str) == 0 {
			return
		}

		bytes := []byte(str)
		command := bytesToCommand(bytes[:config.CommandLength])

		if command == "tx" {
			txHandler(bytes)
		} else if command == "block" {
			blkHandler(bytes)
		} else if command == "run" {
			signalHandler(bytes)
		}
	}
}

// ConnectPeer connect to a specific peer
func ConnectPeer(dest string, pid1, pid2, pid3 string, sender bool, h core.Host, txHandler TxHandler, blkHandler BlkHandler, signalHandler RunSignalHandler) error {
	maddr, err := multiaddr.NewMultiaddr(dest)
	if err != nil {
		return err
	}

	info, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return err
	}

	h.Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)

	// start streams with the dest peer
	if sender {
		s1, err2 := h.NewStream(context.Background(), info.ID, protocol.ID(pid1))
		if err2 != nil {
			return err2
		}
		txWriter := bufio.NewWriter(s1)
		txWriterSet = append(txWriterSet, txWriter)
		go readDataFromP2P(bufio.NewReader(s1), txHandler, blkHandler, signalHandler)

		return nil
	}

	s2, err := h.NewStream(context.Background(), info.ID, protocol.ID(pid2))
	if err != nil {
		return err
	}
	blkWriter := bufio.NewWriter(s2)
	blockWriterSet = append(blockWriterSet, blkWriter)
	go readDataFromP2P(bufio.NewReader(s2), txHandler, blkHandler, signalHandler)

	s3, err := h.NewStream(context.Background(), info.ID, protocol.ID(pid3))
	if err != nil {
		return err
	}
	signalWriter := bufio.NewWriter(s3)
	signalWriterSet = append(signalWriterSet, signalWriter)
	go readDataFromP2P(bufio.NewReader(s3), txHandler, blkHandler, signalHandler)

	return nil
}

// PeerDiscover discover peers by using kad-dht and connect peers
func PeerDiscover(h core.Host, announce string, pid string, txHandler TxHandler, blkHandler BlkHandler, signalHandler2 RunSignalHandler) error {
	ctx := context.Background()
	kademliaDHT, err := dht.New(ctx, h)
	if err != nil {
		return err
	}

	// bootstrap the DHT
	if err = kademliaDHT.Bootstrap(ctx); err != nil {
		return err
	}

	// connect to the bootstrap nodes
	var wg sync.WaitGroup
	for _, peerAddr := range defaultBootstrapPeers {
		peerInfo, _ := peer.AddrInfoFromP2pAddr(peerAddr)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err = h.Connect(ctx, *peerInfo); err != nil {
				log.Println(err)
			} else {
				log.Printf("Connection established with bootstrap node: %v \n", *peerInfo)
			}
		}()
	}
	wg.Wait()

	// use a specific rendezvous point to announce our location
	routingDiscovery := drouting.NewRoutingDiscovery(kademliaDHT)
	dutil.Advertise(ctx, routingDiscovery, announce)

	// start a stream with the discovered peer
	peerChan, err := routingDiscovery.FindPeers(ctx, announce)
	if err != nil {
		return err
	}
	for p := range peerChan {
		if p.ID == h.ID() {
			continue
		}
		s, err := h.NewStream(ctx, p.ID, protocol.ID(pid))
		if err != nil {
			log.Println(err)
			continue
		} else {
			//pw := newWriter(s)
			//p2pWriterSet = append(p2pWriterSet, pw)
			go readDataFromP2P(bufio.NewReader(s), txHandler, blkHandler, signalHandler2)
		}
	}

	return nil
}

// SyncTx broadcasts tx to all the connected peers
func SyncTx(tx []byte) error {
	for _, w := range txWriterSet {
		_, err := w.WriteString(fmt.Sprintf("%s\n", string(tx)))
		if err != nil {
			return err
		}
		err = w.Flush()
		if err != nil {
			return err
		}
	}
	return nil
}

// SyncBlock broadcasts block to all the connected peers
func SyncBlock(blk []byte) error {
	for _, w := range blockWriterSet {
		_, err := w.WriteString(fmt.Sprintf("%s\n", string(blk)))
		if err != nil {
			return err
		}
		err = w.Flush()
		if err != nil {
			return err
		}
	}
	return nil
}

// SyncSignal broadcasts signal to all the connected peers
func SyncSignal(signal []byte) error {
	for _, w := range signalWriterSet {
		_, err := w.WriteString(fmt.Sprintf("%s\n", string(signal)))
		if err != nil {
			return err
		}
		err = w.Flush()
		if err != nil {
			return err
		}
	}
	return nil
}
