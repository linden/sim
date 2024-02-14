package sim

import (
	"errors"
	"io"
	"net/rpc"
)

type Client struct {
	*rpc.Client
}

func (c *Client) Ping() error {
	var res string

	err := c.Call("Handler.Ping", NewEmpty(), &res)
	if err != nil {
		return err
	}

	if res != "pong" {
		return errors.New("unexpected reply")
	}

	return nil
}

func (c *Client) Address() (*Address, error) {
	res := &Address{}

	err := c.Call("Handler.Address", NewEmpty(), res)
	return res, err
}

func (c *Client) Send(addr string, amt int64) (*Send, error) {
	res := &Send{}

	err := c.Call("Handler.Send", &SendArgs{
		Address: addr,
		Amount:  amt,
	}, res)
	return res, err
}

func (c *Client) Mine(count uint32) (*Mine, error) {
	res := &Mine{}

	err := c.Call("Handler.Mine", &MineArgs{
		Count: count,
	}, res)
	return res, err
}

func (c *Client) BestBlock() (*BestBlock, error) {
	res := &BestBlock{}

	err := c.Call("Handler.BestBlock", NewEmpty(), res)
	return res, err
}

func (c *Client) Stop() error {
	err := c.Call("Handler.Stop", NewEmpty(), NewEmpty())

	// disregard unexpected EOF, as we expect the unix socket listener to close.
	if errors.Is(err, io.ErrUnexpectedEOF) {
		return nil
	}

	return err
}

func Dial(addr string) (*Client, error) {
	c, err := rpc.Dial("unix", addr)
	if err != nil {
		return nil, err
	}

	return &Client{
		Client: c,
	}, nil
}
