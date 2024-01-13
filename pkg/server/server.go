package server

import (
	"errors"
	"fmt"
	"net"
	"net/rpc"
	"os"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/integration/rpctest"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"

	"github.com/linden/sim/internal/types"
)

type Handler struct {
	p2p int
	rpc int

	harness *rpctest.Harness
}

func (h *Handler) Ping(args types.Empty, reply *string) error {
	*reply = "pong"

	return nil
}

func (h *Handler) Send(args *types.SendArgs, reply *types.Send) error {
	// decode the the raw address.
	addr, err := btcutil.DecodeAddress(args.Address, types.Chain)
	if err != nil {
		return fmt.Errorf("could not decode address: %v", err)
	}

	// create an output script with our testing address.
	scpt, err := txscript.PayToAddrScript(addr)
	if err != nil {
		return fmt.Errorf("could not create pay to addr script: %v", err)
	}

	// send the output, since we're on simnet funds are created.
	txid, err := h.harness.SendOutputs([]*wire.TxOut{
		// create a new output with the amount.
		wire.NewTxOut(args.Amount, scpt),
	}, 10)
	if err != nil {
		return fmt.Errorf("could not send output to testing address: %v", err)
	}

	*reply = types.Send{
		TXID: txid,
	}

	return nil
}

func (h *Handler) Mine(args *types.MineArgs, reply *types.Mine) error {
	// mine {count} blocks.
	blks, err := h.harness.Client.Generate(args.Count)
	if err != nil {
		return err
	}

	// reply with the block hashes.
	*reply = types.Mine{
		Blocks: blks,
	}

	return nil
}

func (h *Handler) Address(args types.Empty, reply *types.Address) error {
	// reply with the P2P address.
	*reply = types.Address{
		P2P: fmt.Sprintf(":%d", h.p2p),
	}

	return nil
}

type Server struct {
	server   *rpc.Server
	handler  *Handler
	listener net.Listener
}

func (s *Server) Close() error {
	err := s.handler.harness.TearDown()
	if err != nil {
		return err
	}

	return s.listener.Close()
}

func (s *Server) Accept() {
	// start accepting connections.
	s.server.Accept(s.listener)
}

func New(addr string) (*Server, error) {
	// find the next available ports for P2P and RPC.
	p2pp := rpctest.NextAvailablePort()
	rpcp := rpctest.NextAvailablePort()

	// create a new harness.
	h, err := rpctest.New(types.Chain, nil, []string{
		// support neutrino.
		"--txindex",

		// prevent banning during testing.
		"--nobanning",
		"--nostalldetect",

		// listen on all interfaces.
		fmt.Sprintf("--listen=:%d", p2pp),
		fmt.Sprintf("--rpclisten=:%d", rpcp),
	}, "")
	if err != nil {
		return nil, err
	}

	// set up the harness.
	err = h.SetUp(false, 0)
	if err != nil {
		return nil, err
	}

	// ensure the unix socket was removed.
	err = os.Remove(addr)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	// create the rpc server.
	s := rpc.NewServer()

	// create the handler.
	hdlr := &Handler{
		p2p: p2pp,
		rpc: rpcp,

		harness: h,
	}

	// register the handler with the server.
	err = s.Register(hdlr)
	if err != nil {
		return nil, err
	}

	// create the unix socket and listen.
	l, err := net.Listen("unix", addr)
	if err != nil {
		return nil, err
	}

	return &Server{
		server:   s,
		handler:  hdlr,
		listener: l,
	}, nil
}
