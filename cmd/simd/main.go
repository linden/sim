package main

import (
	"flag"
	"log"

	"github.com/linden/sim/internal/types"
	"github.com/linden/sim/pkg/server"
)

var Socket string

func init() {
	log.SetFlags(0)
	log.SetPrefix("simd: ")

	flag.StringVar(&Socket, "socket", types.Socket, "")
	flag.Parse()
}

func main() {
	// create the server.
	s, err := server.New(Socket)
	if err != nil {
		log.Fatal(err)
	}

	// start accepting connections.
	s.Accept()
}
