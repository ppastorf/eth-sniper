package swap

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/big"

	pancake "sniper/contracts/bsc/pancakeswap"
	eth "sniper/pkg/eth"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

type PriceWatcher struct {
	tokenA       *eth.Token
	tokenB       *eth.Token
	currentPrice *big.Float
}

func NewPriceWatcher(client *ethclient.Client, dex *Dex, ctx context.Context, tokenA, tokenB *eth.Token, pending bool) (*PriceWatcher, error) {
	var err error
	p := &PriceWatcher{
		tokenA: tokenA,
		tokenB: tokenB,
	}
	err = p.subscribe(client, dex, ctx, tokenA, tokenB, pending)
	if err != nil {
		return nil, fmt.Errorf("Cannot subscribe to token prices: %s\n", err)
	}

	return p, nil
}

func (p *PriceWatcher) CurrentPrice() *big.Float {
	return p.currentPrice
}

func (p *PriceWatcher) Tokens() []common.Address {
	return []common.Address{p.tokenA.Address, p.tokenB.Address}
}

func (p *PriceWatcher) subscribe(client *ethclient.Client, dex *Dex, ctx context.Context, tokenA *eth.Token, tokenB *eth.Token, pending bool) error {
	var err error
	opts := &bind.CallOpts{
		Pending:     false,
		BlockNumber: nil,
		Context:     ctx,
	}

	pairAddr, err := dex.Factory.GetPair(opts, tokenA.Address, tokenB.Address)
	if err != nil {
		return err
	}

	pair, err := pancake.NewPancakePair(pairAddr, client)
	if err != nil {
		return err
	}

	sameOrder, err := isAtSameOrderAsPair(ctx, pair, tokenA, tokenB)
	if err != nil {
		return fmt.Errorf("cannot determine token order of pair: %s", err)
	}

	go func() {
		opts := &bind.CallOpts{
			Pending:     pending,
			BlockNumber: nil,
			Context:     ctx,
		}
		for {
			select {
			case <-ctx.Done():
				return
			default:
				reserves, err := pair.GetReserves(opts)
				if err != nil {
					log.Printf("Error calling pair.GetReserves: %s\n", err)
					return
				}

				var price *big.Float
				if sameOrder {
					price, err = eth.TokenRatio(reserves.Reserve0, reserves.Reserve1)
				} else {
					price, err = eth.TokenRatio(reserves.Reserve1, reserves.Reserve0)
				}
				if err != nil {
					log.Printf("Failed to determine token price: %s\n", err)
					continue
				}
				p.currentPrice = price
			}
		}
	}()
	return nil
}

func isAtSameOrderAsPair(ctx context.Context, pair DexPair, tokenA, tokenB *eth.Token) (is bool, err error) {
	opts := &bind.CallOpts{
		Pending:     false,
		BlockNumber: nil,
		Context:     ctx,
	}
	token0Addr, err := pair.Token0(opts)
	if err != nil {
		return false, err
	}
	token1Addr, err := pair.Token1(opts)
	if err != nil {
		return false, err
	}

	if tokenA.Address == token0Addr && tokenB.Address == token1Addr {
		return true, nil
	}
	if tokenA.Address == token1Addr && tokenB.Address == token0Addr {
		return false, nil
	}
	return false, errors.New("Tokens passed are not the same as of pair")
}
