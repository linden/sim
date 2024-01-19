package main

import (
	"flag"
	"log"

	"github.com/linden/sim"
)

var (
	Socket string

	RPC int
	P2P int
)

func init() {
	log.SetFlags(0)
	log.SetPrefix("simd: ")

	flag.StringVar(&Socket, "socket", sim.DefaultSocket, "")
	flag.IntVar(&RPC, "rpc-port", 0, "")
	flag.IntVar(&P2P, "p2p-port", 0, "")
	flag.Parse()
}

func main() {
	// create the server.
	s, err := sim.NewServer(Socket, RPC, P2P)
	if err != nil {
		log.Fatal(err)
	}

	// start accepting connections.
	s.Accept()
}
