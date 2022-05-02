package eth

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"sniper/contracts/tokens"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
)

type Token struct {
	*Contract
	*tokens.Erc20Token
	Symbol string
}

func NewToken(client *ethclient.Client, address common.Address) (*Token, error) {
	var err error

	tokenContract, err := NewContract(address, tokens.Erc20TokenMetaData)
	if err != nil {
		return nil, fmt.Errorf("Failed to instantiate Token contract: %s\n", err)
	}

	tokenClient, err := tokens.NewErc20Token(address, client)
	if err != nil {
		return nil, fmt.Errorf("Failed to instantiate Token client: %s\n", err)
	}

	opts := &bind.CallOpts{
		Pending:     false,
		BlockNumber: nil,
		Context:     context.Background(),
	}
	symbol, err := tokenClient.Symbol(opts)
	if err != nil {
		log.Printf("Failed to get symbol of Token at %s: %s", address, err)
		symbol = "TKN"
	}

	t := &Token{
		tokenContract,
		tokenClient,
		symbol,
	}

	return t, nil
}

func (t *Token) PrintBalanceAt(ctx context.Context, addr common.Address, pending bool) error {
	var err error
	opts := &bind.CallOpts{
		Pending:     pending,
		From:        addr,
		BlockNumber: nil,
		Context:     ctx,
	}

	balance, err := t.BalanceOf(opts, addr)
	if err != nil {
		return fmt.Errorf("Failed to get %s balance of address %s: %s", t.Symbol, addr.Hex(), err)
	}

	log.Printf("Current %s balance: %f \n", t.Symbol, FromWei(balance, params.Ether))
	return nil
}

func FromWei(wei *big.Int, unit float64) *big.Float {
	asFloat := new(big.Float).SetPrec(256).SetMode(big.ToNearestEven)
	weiFloat := new(big.Float).SetPrec(256).SetMode(big.ToNearestEven)

	return asFloat.Quo(weiFloat.SetInt(wei), big.NewFloat(unit))
}

func ToWei(val *big.Float, unit float64) *big.Int {
	valWei := val.Mul(val, big.NewFloat(unit))

	weiTxt := strings.Split(valWei.Text('f', 64), ".")[0]
	wei, ok := new(big.Int).SetString(weiTxt, 10)
	if !ok {
		fmt.Printf("erro na conversao: %v\n", weiTxt)
	}

	val.Int(wei)

	return wei
}
