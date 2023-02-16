package main

import (
	"MorphDAG/config"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
)

func main() {
	var local bool
	flag.BoolVar(&local, "le", true, "specify the node operating environment. defaults to true.")
	flag.Parse()

	if local {
		CreateLocalNodeAddrs()
	} else {
		CreateRemoteNodeAddrs()
	}
}

// CreateLocalNodeAddrs creates node connection file for each node (in a local environment)
func CreateLocalNodeAddrs() {
	for i := 0; i < config.NodeNumber; i++ {
		// each node runs one MorphDAG instance
		fileName := fmt.Sprintf("../nodefile/nodeaddrs_%s.txt", strconv.Itoa(i))
		file, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_TRUNC|os.O_APPEND, 0666)
		if err != nil {
			fmt.Printf("error: %v\n", err)
		}

		for j := i + 1; j < 100; j++ {
			readName := fmt.Sprintf("../nodefile/config/fulladdrs_%s.txt", strconv.Itoa(j))
			bs, err := ioutil.ReadFile(readName)
			if err != nil {
				log.Panic("Read error: ", err)
			}
			strs := strings.Split(string(bs), "\n")
			ipaddr := strs[1] + "\n"

			_, err = file.WriteString(ipaddr)
			if err != nil {
				log.Printf("error: %v\n", err)
			}
		}
	}

	// assume node #10000 is the tx sender
	fileName := "../nodefile/nodeaddrs_10000.txt"
	file, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_TRUNC|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}

	for i := 0; i < config.NodeNumber; i++ {
		readName := fmt.Sprintf("../nodefile/config/fulladdrs_%s.txt", strconv.Itoa(i))
		bs, err := ioutil.ReadFile(readName)
		if err != nil {
			log.Panic("Read error: ", err)
		}
		strs := strings.Split(string(bs), "\n")
		ipaddr := strs[1] + "\n"

		_, err = file.WriteString(ipaddr)
		if err != nil {
			log.Printf("error: %v\n", err)
		}
	}
}

// CreateRemoteNodeAddrs creates node connection file for each node (in a distributed environment)
func CreateRemoteNodeAddrs() {
	var count int
	for i := 0; i < config.NodeNumber; i++ {
		// each node runs two MorphDAG instances
		if i%2 == 0 {
			count++
		}
		fileName := fmt.Sprintf("../nodefile/ecs/node%s/nodeaddrs_%s.txt", strconv.Itoa(count-1), strconv.Itoa(i))
		file, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_TRUNC|os.O_APPEND, 0666)
		if err != nil {
			fmt.Printf("error: %v\n", err)
		}

		for j := i + 1; j < 100; j++ {
			readName := fmt.Sprintf("../nodefile/config/fulladdrs_%s.txt", strconv.Itoa(j))
			bs, err := ioutil.ReadFile(readName)
			if err != nil {
				log.Panic("Read error: ", err)
			}
			strs := strings.Split(string(bs), "\n")
			ipaddr := strs[0] + "\n"

			//node := math.Floor(float64(j) / 2)
			//var ip string
			//switch node {
			//case 0:
			//	ip = "172.16.168.55"
			//case 1:
			//	ip = "172.16.168.54"
			//case 2:
			//	ip = "172.16.168.56"
			//case 3:
			//	ip = "172.16.168.58"
			//case 4:
			//	ip = "172.16.168.59"
			//case 5:
			//	ip = "172.16.168.60"
			//case 6:
			//	ip = "172.16.168.65"
			//case 7:
			//	ip = "172.16.168.66"
			//case 8:
			//	ip = "172.16.168.63"
			//case 9:
			//	ip = "172.16.168.64"
			//case 10:
			//	ip = "172.16.168.67"
			//case 11:
			//	ip = "172.16.168.68"
			//case 12:
			//	ip = "172.16.168.70"
			//case 13:
			//	ip = "172.16.168.75"
			//case 14:
			//	ip = "172.16.168.76"
			//case 15:
			//	ip = "172.16.168.73"
			//case 16:
			//	ip = "172.16.168.69"
			//case 17:
			//	ip = "172.16.168.72"
			//case 18:
			//	ip = "172.16.168.71"
			//case 19:
			//	ip = "172.16.168.74"
			//case 20:
			//	ip = "172.16.168.79"
			//case 21:
			//	ip = "172.16.168.80"
			//case 22:
			//	ip = "172.16.168.81"
			//case 23:
			//	ip = "172.16.168.86"
			//case 24:
			//	ip = "172.16.168.78"
			//case 25:
			//	ip = "172.16.168.85"
			//case 26:
			//	ip = "172.16.168.84"
			//case 27:
			//	ip = "172.16.168.77"
			//case 28:
			//	ip = "172.16.168.83"
			//case 29:
			//	ip = "172.16.168.82"
			//case 30:
			//	ip = "172.16.168.89"
			//case 31:
			//	ip = "172.16.168.103"
			//case 32:
			//	ip = "172.16.168.97"
			//case 33:
			//	ip = "172.16.168.91"
			//case 34:
			//	ip = "172.16.168.105"
			//case 35:
			//	ip = "172.16.168.99"
			//case 36:
			//	ip = "172.16.168.95"
			//case 37:
			//	ip = "172.16.168.102"
			//case 38:
			//	ip = "172.16.168.100"
			//case 39:
			//	ip = "172.16.168.94"
			//case 40:
			//	ip = "172.16.168.92"
			//case 41:
			//	ip = "172.16.168.90"
			//case 42:
			//	ip = "172.16.168.96"
			//case 43:
			//	ip = "172.16.168.101"
			//case 44:
			//	ip = "172.16.168.104"
			//case 45:
			//	ip = "172.16.168.106"
			//case 46:
			//	ip = "172.16.168.88"
			//case 47:
			//	ip = "172.16.168.98"
			//case 48:
			//	ip = "172.16.168.87"
			//case 49:
			//	ip = "172.16.168.93"
			//}
			//lines := strings.Split(ipaddr, "/")
			//final := fmt.Sprintf("/ip4/%s/tcp/%s/p2p/%s\n", ip, lines[4], lines[6])

			_, err = file.WriteString(ipaddr)
			if err != nil {
				log.Printf("error: %v\n", err)
			}
		}
	}

	// assume node #10000 is the tx sender (run in the instance #0 by default)
	fileName := "../nodefile/ecs/node0/nodeaddrs_10000.txt"
	file, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_TRUNC|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}

	for i := 0; i < config.NodeNumber; i++ {
		readName := fmt.Sprintf("../nodefile/config/fulladdrs_%s.txt", strconv.Itoa(i))
		bs, err := ioutil.ReadFile(readName)
		if err != nil {
			log.Panic("Read error: ", err)
		}
		strs := strings.Split(string(bs), "\n")
		ipaddr := strs[0] + "\n"

		_, err = file.WriteString(ipaddr)
		if err != nil {
			log.Printf("error: %v\n", err)
		}
	}
}
