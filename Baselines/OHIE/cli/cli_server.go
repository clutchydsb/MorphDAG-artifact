package cli

import (
	"fmt"
	"github.com/urfave/cli/v2"
	"log"
	"os"
)

type CMDServer struct {
	P2PPort       int
	RPCPort       int
	Cycles        int
	NodeFile      string
	Pid1          string
	Pid2          string
	Pid3          string
	FullAddrsPath string
	PKFile        string
	TxSender      bool
}

func (cmd *CMDServer) Run(nodeID string) {
	app := &cli.App{
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:        "p2pport",
				Aliases:     []string{"pp"},
				Usage:       "P2P Port to listen to",
				Value:       7000,
				Destination: &cmd.P2PPort,
			},
			&cli.IntFlag{
				Name:        "rpcport",
				Aliases:     []string{"rp"},
				Usage:       "RPC port to listen to",
				Value:       5000,
				Destination: &cmd.RPCPort,
			},
			&cli.IntFlag{
				Name:        "cycles",
				Aliases:     []string{"c"},
				Usage:       "Number of cycles to run",
				Value:       30,
				Destination: &cmd.Cycles,
			},
			&cli.StringFlag{
				Name:        "nodefile",
				Aliases:     []string{"nf"},
				Usage:       "Path of a file storing the destination node addresses for connecting",
				Value:       fmt.Sprintf("../nodefile/nodeaddrs_%s.txt", nodeID),
				Destination: &cmd.NodeFile,
			},
			&cli.StringFlag{
				Name:        "pid1",
				Aliases:     []string{"p1"},
				Value:       "Occam-Tx",
				Usage:       "pid to identify a network protocol",
				Destination: &cmd.Pid1,
			},
			&cli.StringFlag{
				Name:        "pid2",
				Aliases:     []string{"p2"},
				Value:       "Occam-Block",
				Usage:       "pid to identify a network protocol",
				Destination: &cmd.Pid2,
			},
			&cli.StringFlag{
				Name:        "pid3",
				Aliases:     []string{"p3"},
				Value:       "Occam-Signal",
				Usage:       "pid to identify a network protocol",
				Destination: &cmd.Pid3,
			},
			&cli.StringFlag{
				Name:        "fulladdrs",
				Aliases:     []string{"f"},
				Value:       fmt.Sprintf("../nodefile/fulladdrs_%s.txt", nodeID),
				Usage:       "Path of a file storing the full node addresses for listening",
				Destination: &cmd.FullAddrsPath,
			},
			&cli.StringFlag{
				Name:        "pkfile",
				Aliases:     []string{"pk"},
				Value:       fmt.Sprintf("../nodefile/Occam_%s.pk", nodeID),
				Usage:       "Path of a file storing the private key",
				Destination: &cmd.PKFile,
			},
			&cli.BoolFlag{
				Name:        "sender",
				Aliases:     []string{"sd"},
				Value:       false,
				Usage:       "Identifier to identify the transaction sender",
				Destination: &cmd.TxSender,
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Panic(err)
	}
}
