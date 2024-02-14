package sim

import (
	"errors"
	"fmt"
	"math"
	"net"
	"net/rpc"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/integration/rpctest"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btclog"
)

func init() {
	rpctest.ListenAddressGenerator = func() (string, string) {
		return fmt.Sprintf(rpctest.ListenerFormat, newPort()), fmt.Sprintf(rpctest.ListenerFormat, newPort())
	}
}

type Handler struct {
	p2p int
	rpc int

	stopChan chan bool

	harness *rpctest.Harness
}

func (h *Handler) Ping(args Empty, reply *string) error {
	*reply = "pong"

	return nil
}

func (h *Handler) Send(args *SendArgs, reply *Send) error {
	// decode the the raw address.
	addr, err := btcutil.DecodeAddress(args.Address, Chain)
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

	*reply = Send{
		TXID: txid,
	}

	return nil
}

func (h *Handler) Mine(args *MineArgs, reply *Mine) error {
	// mine {count} blocks.
	blks, err := h.harness.Client.Generate(args.Count)
	if err != nil {
		return err
	}

	// reply with the block hashes.
	*reply = Mine{
		Blocks: blks,
	}

	return nil
}

func (h *Handler) Address(args Empty, reply *Address) error {
	// reply with the P2P address.
	*reply = Address{
		P2P: fmt.Sprintf(":%d", h.p2p),
	}

	return nil
}

func (h *Handler) BestBlock(args Empty, reply *BestBlock) error {
	// query the best block.
	hsh, ht, err := h.harness.Client.GetBestBlock()
	if err != nil {
		return err
	}

	// reply with the best block.
	*reply = BestBlock{
		Height: ht,
		Hash:   hsh,
	}

	return nil
}

func (h *Handler) Stop(args Empty, reply Empty) error {
	// teardown the btcd node.
	err := h.harness.TearDown()
	if err != nil {
		return err
	}

	// signal to stop listening.
	h.stopChan <- true

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

// accept connections with a graceful shutdown.
func (s *Server) Accept() error {
	connChan := make(chan net.Conn)
	errChan := make(chan error)

	// accept connections.
	for {
		// wait for connections on another routine, as it's blocking.
		go func() {
			conn, err := s.listener.Accept()
			if err != nil {
				errChan <- err
				return
			}

			connChan <- conn
		}()

		select {
		// stoo running.
		case <-s.handler.stopChan:
			return s.listener.Close()

		// handle a connection.
		case conn := <-connChan:
			go s.server.ServeConn(conn)

		// handle a failed connection.
		case err := <-errChan:
			return err
		}
	}

	return nil

}

func watch(path string) {
	// path to the log file.
	p := filepath.Join(path, "simnet", "btcd.log")

	var lst int64

	const tm = 50 * time.Millisecond

	for {
		// open the log file.
		f, err := os.Open(p)
		if err != nil {
			// ignore error if the file hasn't been created yet.
			if os.IsNotExist(err) {
				time.Sleep(tm)
				continue
			}

			panic(err)
		}

		// query the file info.
		inf, err := f.Stat()
		if err != nil {
			panic(err)
		}

		sz := inf.Size()

		// check if the file size has changed.
		if sz != lst && sz != 0 {
			// create a slice with the difference in size.
			b := make([]byte, sz-lst)

			// read from the last point.
			_, err = f.ReadAt(b, lst)
			if err != nil {
				panic(err)
			}

			fmt.Printf("%s", b)

			lst = sz
		}

		time.Sleep(tm)
	}
}

var used = map[int]struct{}{}

func newPort() int {
	// iterate through all possible ports, starting from the highest.
	for i := math.MaxUint16; i > 0; i-- {
		// skip if already being used by us.
		if _, ok := used[i]; ok {
			continue
		}

		// connect to the port on any interface.
		conn, err := net.Dial("tcp", fmt.Sprintf(":%d", i))
		if err != nil {
			// ensure the syscall error is "connection refused".
			if !errors.Is(err, syscall.ECONNREFUSED) {
				continue
			}

			// set the port as used.
			used[i] = struct{}{}

			return i
		}

		// close the connection.
		conn.Close()
	}

	panic("port exhaustion")
}

func NewServer(addr string, rpcp, p2pp int, level btclog.Level) (*Server, error) {
	// find the next available ports for P2P and RPC if not defined.
	if rpcp == 0 {
		rpcp = newPort()
	}

	if p2pp == 0 {
		p2pp = newPort()
	}

	args := []string{
		// support neutrino.
		"--txindex",

		// prevent banning during testing.
		"--nobanning",
		"--nostalldetect",

		// listen on all interfaces.
		fmt.Sprintf("--listen=:%d", p2pp),
		fmt.Sprintf("--rpclisten=:%d", rpcp),
	}

	// create a temporary directory for storing logs.
	lgs, err := os.MkdirTemp("", "*-simd")
	if err != nil {
		return nil, err
	}

	if level != btclog.LevelOff {
		var l string

		// `btclog.(Level).String()` uses a shorten version of the name, which does not work as an argument.
		// https://github.com/btcsuite/btclog/blob/84c8d2346e9f/log.go#L98
		// so instead we convert it manually here.
		switch level {
		case btclog.LevelTrace:
			l = "trace"

		case btclog.LevelDebug:
			l = "debug"

		case btclog.LevelInfo:
			l = "info"

		case btclog.LevelWarn:
			l = "warn"

		case btclog.LevelError:
			l = "error"

		case btclog.LevelCritical:
			l = "critical"
		}

		args = append(args,
			"--logdir="+lgs,
			"--debuglevel="+l,
		)

		go watch(lgs)
	} else {
		// btcd doesn't offer an option to completely disable logs, so instead we store them in temporary directory and ignore them.
		args = append(args, "--logdir="+lgs)
	}

	// create a new harness.
	h, err := rpctest.New(Chain, nil, args, "")
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

		stopChan: make(chan bool, 1),

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
