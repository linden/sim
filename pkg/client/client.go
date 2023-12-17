package client

import (
	"errors"
	"net/rpc"

	"github.com/linden/sim/internal/types"
)

type Client struct {
	*rpc.Client
}

func (c *Client) Ping() error {
	var res string

	err := c.Call("Handler.Ping", types.NewEmpty(), &res)
	if err != nil {
		return err
	}

	if res != "pong" {
		return errors.New("unexpected reply")
	}

	return nil
}

func (c *Client) Address() (*types.Address, error) {
	res := &types.Address{}

	err := c.Call("Handler.Address", types.NewEmpty(), res)
	return res, err
}

func (c *Client) Send(addr string, amt int64) (*types.Send, error) {
	res := &types.Send{}

	err := c.Call("Handler.Send", &types.SendArgs{
		Address: addr,
		Amount:  amt,
	}, res)
	return res, err
}

func (c *Client) Mine(count uint32) (*types.Mine, error) {
	res := &types.Mine{}

	err := c.Call("Handler.Mine", &types.MineArgs{
		Count: count,
	}, res)
	return res, err
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
