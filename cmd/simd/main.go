package main

import (
	"flag"
	"log"

	"github.com/btcsuite/btclog"

	"github.com/linden/sim"
)

var (
	Socket string

	RPC int
	P2P int

	Level string
)

func init() {
	log.SetFlags(0)
	log.SetPrefix("simd: ")

	flag.StringVar(&Socket, "socket", sim.DefaultSocket, "")
	flag.StringVar(&Level, "debuglevel", "off", "")
	flag.IntVar(&RPC, "rpc-port", 0, "")
	flag.IntVar(&P2P, "p2p-port", 0, "")
	flag.Parse()
}

func main() {
	l, ok := btclog.LevelFromString(Level)
	if !ok {
		// fallback to off.
		l = btclog.LevelOff
	}

	// create the server.
	s, err := sim.NewServer(Socket, RPC, P2P, l)
	if err != nil {
		log.Fatal(err)
	}

	// start accepting connections.
	s.Accept()
}
