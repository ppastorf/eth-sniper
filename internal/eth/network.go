package eth

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/ethclient"
)

type Network struct {
	RpcUrl    string
	connected bool
}

func (n *Network) IsConnected() bool {
	return n.connected
}

func (n *Network) Connect(ctx context.Context) (*ethclient.Client, error) {
	client, err := ethclient.Dial(n.RpcUrl)
	if err != nil {
		return nil, fmt.Errorf("Cannot connect to %s: %s\n", n.RpcUrl, err)
	}
	n.connected = true

	return client, err
}
