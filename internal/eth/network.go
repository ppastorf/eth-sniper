package eth

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/ethclient"
)

type Network struct {
	ConnectAddress string
	Currency       *Token
	chainID        *big.Int
	connected      bool
}

func (n *Network) GetChainID() *big.Int {
	return n.chainID
}

func (n *Network) IsConnected() bool {
	return n.connected
}

func (n *Network) Connect(ctx context.Context) (*ethclient.Client, error) {
	client, err := ethclient.Dial(n.ConnectAddress)
	if err != nil {
		return nil, fmt.Errorf("Cannot connect to %s: %s\n", n.ConnectAddress, err)
	}

	chainId, err := client.NetworkID(ctx)
	if err != nil {
		return nil, fmt.Errorf("Cannot get chain ID for network at %s: %s", n.ConnectAddress, err)
	}

	n.chainID = chainId
	n.connected = true

	return client, err
}