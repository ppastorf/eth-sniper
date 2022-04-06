package main

// - agendamento de transação
// - criação automatica de carteiras
// - transacoes em paralelo
// - dependente do valor max por transacao
// - dependente do investimento
// - otmimização de gas

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"math/big"
	pancake "sniper/contracts/bsc/pancakeswaprouter"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
)

type Token struct {
	Contract common.Address
	Repr     string
}

type Wallet struct {
	publicKey  *ecdsa.PublicKey
	privateKey *ecdsa.PrivateKey
	tokensHeld []*Token
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

func (w *Wallet) GetBalance(client *ethclient.Client, ctx context.Context, unit float64) (*big.Float, error) {
	balanceWei, err := client.BalanceAt(ctx, w.Address(), nil)
	if err != nil {
		return nil, err
	}

	return fromWei(balanceWei, unit), nil
}

func (w *Wallet) SignTransaction(tx *types.Transaction, network *Network) (*types.Transaction, error) {
	return types.SignTx(tx, types.NewEIP155Signer(network.ChainID), w.privateKey)
}

func fromWei(wei *big.Int, unit float64) *big.Float {
	asFloat := new(big.Float).SetPrec(256).SetMode(big.ToNearestEven)
	weiFloat := new(big.Float).SetPrec(256).SetMode(big.ToNearestEven)

	return asFloat.Quo(weiFloat.SetInt(wei), big.NewFloat(unit))
}

func toWei(val *big.Float, unit float64) *big.Int {
	valWei := val.Mul(val, big.NewFloat(unit))

	weiTxt := strings.Split(valWei.Text('f', 64), ".")[0]
	wei, ok := new(big.Int).SetString(weiTxt, 10)
	if !ok {
		fmt.Printf("erro na conversao: %v\n", weiTxt)
	}

	val.Int(wei)

	return wei
}

type DexRouter struct {
	Address common.Address
	Abi     string
}

type Network struct {
	ConnectAddress string
	BaseCurrency   *Token
	ChainID        *big.Int
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

	n.ChainID = chainId

	return client, err
}

type SwapTransaction struct {
	Network     *Network
	Client      *ethclient.Client
	Router      *pancake.Pancakeswaprouter
	FromWallet  *Wallet
	TokenIn     common.Address
	TokenOut    common.Address
	Amount      *big.Int
	Expiration  *big.Int
	GasStrategy string
}

func (swap *SwapTransaction) BuildAndSign(ctx context.Context) (*types.Transaction, error) {
	nonce, err := swap.Client.PendingNonceAt(ctx, swap.FromWallet.Address())
	if err != nil {
		return nil, err
	}

	gasPrice, err := swap.Client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, err
	}

	opts, err := bind.NewKeyedTransactorWithChainID(swap.FromWallet.privateKey, swap.Network.ChainID)
	if err != nil {
		return nil, err

	}
	opts.Nonce = new(big.Int).SetUint64(nonce)
	opts.Value = swap.Amount
	opts.GasPrice = gasPrice
	opts.GasLimit = 0
	opts.Context = ctx
	opts.NoSend = true

	now := big.NewInt(time.Now().Unix())
	exp := new(big.Int).Add(now, swap.Expiration)

	tx, err := swap.Router.SwapExactETHForTokens(
		opts,
		big.NewInt(0),
		[]common.Address{swap.TokenIn, swap.TokenOut},
		swap.FromWallet.Address(),
		exp,
	)

	return tx, err
}

func SendTx(client *ethclient.Client, ctx context.Context, tx *types.Transaction) (*types.Receipt, error) {
	err := client.SendTransaction(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("Failed to send transaction: %s\n", err)
	}

	fmt.Printf("Transaction sent: %s\n", tx.Hash().Hex())

	receipt, err := bind.WaitMined(ctx, client, tx)
	if err != nil {
		return nil, fmt.Errorf("Error waiting transaction to be mined: %s\n", err)
	}

	return receipt, err
}

// func TxReceiptDetails(receipt *types.Receipt) {
// }

func main() {
	ctx := context.Background()

	wallet, err := NewWallet("")
	if err != nil {
		log.Fatalf("Failed to create wallet: %s\n", err.Error())
	}

	bsc := &Network{
		ConnectAddress: "https://bsc-dataseed.binance.org",
		BaseCurrency:   &Token{common.HexToAddress("0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c"), "BNB"},
	}

	client, err := bsc.Connect(ctx)
	if err != nil {
		log.Fatalf("Failed to connect to network: %s\n", err)
	}

	balance, err := wallet.GetBalance(client, ctx, params.Ether)
	if err != nil {
		log.Fatalf("Failed to get wallet balance at %s: %s", wallet.Address(), err)
	}
	fmt.Printf("Current balance: %f %s\n", balance, bsc.BaseCurrency.Repr)

	pancakeRouter, err := pancake.NewPancakeswaprouter(common.HexToAddress("0x10ED43C718714eb63d5aA57B78B54704E256024E"), client)
	if err != nil {
		log.Fatalf("Failed to instantiate PancakeSwap Router: %s\n", err)
	}

	swap := &SwapTransaction{
		Client:      client,
		Network:     bsc,
		Router:      pancakeRouter,
		FromWallet:  wallet,
		TokenIn:     common.HexToAddress("0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c"), // WBNB
		TokenOut:    common.HexToAddress("0x0E09FaBB73Bd3Ade0a17ECC321fD13a19e81cE82"), // CAKE
		Amount:      toWei(big.NewFloat(0.001), params.Ether),
		GasStrategy: "fast",
		Expiration:  big.NewInt(60 * 60),
	}

	tx, err := swap.BuildAndSign(ctx)
	if err != nil {
		log.Fatalf("Failed to sign swap transaction: %s\n", err)
	}

	fmt.Printf("tx: %+v\n", tx)

	_, err = SendTx(client, ctx, tx)
	if err != nil {
		log.Fatalf("Failed to send transaction: %s\n", err)
	}
}
