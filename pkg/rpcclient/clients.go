package rpcclient

import (
	"eduseal/pkg/logger"
	"eduseal/pkg/model"
	"errors"
	"fmt"
	"net/rpc"
)

type Client struct {
	log *logger.Log
	rpc map[string]config
}

type config struct {
	addr           string
	v1             string
	statusEndpoint string
}

// New creates a new rpc client for each service in the config file
func New(cfg *model.Cfg, log *logger.Log) (*Client, error) {
	c := &Client{
		log: log,
		rpc: map[string]config{},
	}

	return c, nil
}

// SingleCall calls the rpc server
func (c *Client) SingleCall(server, method string, args, reply any) error {
	c.log.Info("calling rpc server", "server", server, "endpoint", method)

	service, ok := c.rpc[server]
	if !ok {
		return errors.New("service not found")
	}

	rpcClient, err := rpc.DialHTTP("tcp", service.addr)
	if err != nil {
		return err
	}
	defer rpcClient.Close()

	serviceMethod := fmt.Sprintf("%s.%s", service.v1, method)
	err = rpcClient.Call(serviceMethod, args, reply)
	if err != nil {
		return err
	}

	return nil
}
