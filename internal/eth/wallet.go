package eth

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Wallet struct {
	publicKey  *ecdsa.PublicKey
	privateKey *ecdsa.PrivateKey
}

func (w *Wallet) Address() common.Address {
	return crypto.PubkeyToAddress(*w.publicKey)
}

func NewWallet(privKey string) (*Wallet, error) {
	priv, err := crypto.HexToECDSA(privKey)
	if err != nil {
		return nil, err
	}

	pub, ok := priv.Public().(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("Failed to cast public key to ECDSA")
	}

	w := &Wallet{
		publicKey:  pub,
		privateKey: priv,
	}

	return w, nil
}

func (w *Wallet) GetEthBalance(client *ethclient.Client, ctx context.Context, unit float64) (*big.Float, error) {
	balanceWei, err := client.BalanceAt(ctx, w.Address(), nil)
	if err != nil {
		return nil, err
	}

	return FromWei(balanceWei, unit), nil
}

func (w *Wallet) SignTransaction(tx *types.Transaction, network *Network) (*types.Transaction, error) {
	return types.SignTx(tx, types.NewEIP155Signer(network.GetChainID()), w.privateKey)
}

func (w *Wallet) GetSignerOpts(network *Network) (*bind.TransactOpts, error) {
	return bind.NewKeyedTransactorWithChainID(w.privateKey, network.GetChainID())
}
