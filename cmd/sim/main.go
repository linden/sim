package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/linden/sim"
)

const help = `sim - commands:

help - display this message.
address - display the server's P2P address.
mine <count> - mine blocks.
send <address> <amount> - send sats to an address.
bestblock - query the highest block.`

var Socket string

func init() {
	log.SetFlags(0)
	log.SetPrefix("simd: ")

	flag.StringVar(&Socket, "socket", sim.DefaultSocket, "")
	flag.Parse()
}

func main() {
	// connect to the daemon.
	c, err := sim.Dial(Socket)
	if err != nil {
		log.Fatal(err)
	}

	defer c.Close()

	// ensure ping/pong works.
	err = c.Ping()
	if err != nil {
		log.Fatal(err)
	}

	switch flag.Arg(0) {
	case "address":
		// query the P2P address.
		addr, err := c.Address()
		if err != nil {
			log.Fatal(err)
		}

		fmt.Fprintf(os.Stdout, "P2P address: %s\n", addr.P2P)

	case "send":
		addr := flag.Arg(1)
		if addr == "" {
			log.Fatal("address argument is required")
		}

		// convert the amount argument to an integer.
		amt, err := strconv.Atoi(flag.Arg(2))
		if err != nil {
			log.Fatalf("count argument is invalid: %v", err)
		}

		// mine some blocks than return the count.
		res, err := c.Send(addr, int64(amt))
		if err != nil {
			log.Fatal(err)
		}

		fmt.Fprintf(os.Stdout, "TXID: %+v\n", res.TXID)

	case "mine":
		// convert the count argument to an integer.
		cnt, err := strconv.Atoi(flag.Arg(1))
		if err != nil {
			log.Fatalf("count argument is invalid: %v", err)
		}

		// mine some blocks than return the count.
		blks, err := c.Mine(uint32(cnt))
		if err != nil {
			log.Fatal(err)
		}

		fmt.Fprintf(os.Stdout, "blocks: %+v\n", blks.Blocks)

	case "bestblock":
		// query the best block.
		bst, err := c.BestBlock()
		if err != nil {
			log.Fatal(err)
		}

		fmt.Fprintf(os.Stdout, "height: %d, hash: %x\n", bst.Height, bst.Hash)

	case "stop":
		err := c.Stop()
		if err != nil {
			log.Fatal(err)
		}

	case "help":
		fmt.Fprintf(os.Stdout, "%s\n", help)

	default:
		log.Fatalf("expected command: %s\n", help)
	}
}
