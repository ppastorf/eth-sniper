package eth

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/ethclient"
)

type Node struct {
	Proto    string
	Username string
	Password string
	Address  string
}

func (n *Node) ConnectAddress() string {
	return fmt.Sprintf("%s://%s:%s@%s", n.Proto, n.Username, n.Password, n.Address)
}

type Network struct {
	ConnectAddress string
	EthCurrency    *Token
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
