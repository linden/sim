package sim

import (
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

const DefaultSocket = "/tmp/simd.sock"

var Chain = &chaincfg.SimNetParams

// RPC needs arguments and a return type. a pointer to an empty struct is recommended when nothing else applies.
// https://groups.google.com/g/golang-nuts/c/rZjXufnbcnA.
type Empty *struct{}

func NewEmpty() Empty {
	return Empty(&struct{}{})
}

type SendArgs struct {
	Address string
	Amount  int64
}

type Send struct {
	TXID *chainhash.Hash
}

type MineArgs struct {
	Count uint32
}

type Mine struct {
	Blocks []*chainhash.Hash
}

type Address struct {
	P2P string
}
