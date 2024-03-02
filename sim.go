package sim

import (
	"os"
	"strings"
	"time"

	"github.com/btcsuite/btclog"
)

func New(level btclog.Level) (*Client, error) {
	sock, err := os.CreateTemp("", "simd")
	if err != nil {
		return nil, err
	}

	// create a new simd server.
	n, err := NewServer(sock.Name(), 0, 0, level)
	if err != nil {
		return nil, err
	}

	// start accepting connections.
	go n.Accept()

	var c *Client

	// attempt multiple times.
	for {
		// dial simd.
		c, err = Dial(sock.Name())

		// ensure the error isn't that the socket hasn't stared.
		if err != nil && !strings.Contains(err.Error(), "no such file or directory") {
			return nil, err
		}

		// stop when connected.
		if err == nil {
			break
		}

		// wait 1 second between attempts.
		time.Sleep(1 * time.Second)
	}

	return c, nil
}
