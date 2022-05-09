package swap

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"

	pancake "sniper/contracts/bsc/pancakeswap"
	eth "sniper/pkg/eth"
)

type DexRouter interface {
	SwapExactETHForTokens(opts *bind.TransactOpts, amountOutMin *big.Int, path []common.Address, to common.Address, deadline *big.Int) (*types.Transaction, error)
	SwapExactTokensForETH(opts *bind.TransactOpts, amountIn *big.Int, amountOutMin *big.Int, path []common.Address, to common.Address, deadline *big.Int) (*types.Transaction, error)
	SwapExactETHForTokensSupportingFeeOnTransferTokens(opts *bind.TransactOpts, amountOutMin *big.Int, path []common.Address, to common.Address, deadline *big.Int) (*types.Transaction, error)
	SwapExactTokensForETHSupportingFeeOnTransferTokens(opts *bind.TransactOpts, amountIn *big.Int, amountOutMin *big.Int, path []common.Address, to common.Address, deadline *big.Int) (*types.Transaction, error)
}

type DexFactory interface {
	GetPair(opts *bind.CallOpts, arg0 common.Address, arg1 common.Address) (common.Address, error)
}

type DexPair interface {
	GetReserves(opts *bind.CallOpts) (struct {
		Reserve0           *big.Int
		Reserve1           *big.Int
		BlockTimestampLast uint32
	}, error)
	Token0(opts *bind.CallOpts) (common.Address, error)
	Token1(opts *bind.CallOpts) (common.Address, error)
}

type Dex struct {
	Factory         DexFactory
	FactoryContract *eth.Contract
	Router          DexRouter
	RouterContract  *eth.Contract
}

func SetupDex(client bind.ContractBackend, factoryAddress, routerAddress common.Address) (*Dex, error) {
	factoryContract, err := eth.NewContract(factoryAddress, pancake.PancakeFactoryMetaData)
	if err != nil {
		return nil, fmt.Errorf("Failed to instantiate PancakeFactory contract: %s", err)
	}
	Factory, err := pancake.NewPancakeFactory(factoryAddress, client)
	if err != nil {
		return nil, fmt.Errorf("Failed to instantiate PancakeRouter contract client: %s\n", err)
	}

	routerContract, err := eth.NewContract(routerAddress, pancake.PancakeRouterMetaData)
	if err != nil {
		return nil, fmt.Errorf("Failed to instantiate PancakeRouter contract: %s", err)
	}
	Router, err := pancake.NewPancakeRouter(routerAddress, client)
	if err != nil {
		return nil, fmt.Errorf("Failed to instantiate PancakeRouter contract client: %s\n", err)
	}

	d := &Dex{
		Factory:         Factory,
		FactoryContract: factoryContract,
		Router:          Router,
		RouterContract:  routerContract,
	}
	return d, nil
}

func (dex *Dex) SubscribeTokenPrice(client *ethclient.Client, ctx context.Context, tokenB *eth.Token, tokenA *eth.Token) (<-chan *big.Float, error) {
	prices := make(chan *big.Float)
	var err error
	opts := &bind.CallOpts{
		Pending:     false,
		BlockNumber: nil,
		Context:     ctx,
	}

	pairAddr, err := dex.Factory.GetPair(opts, tokenA.Address, tokenB.Address)
	if err != nil {
		return nil, err
	}

	pair, err := pancake.NewPancakePair(pairAddr, client)
	if err != nil {
		return nil, err
	}

	sameOrder, err := isAtSameOrderAsPair(ctx, pair, tokenA, tokenB)
	if err != nil {
		return nil, fmt.Errorf("Cannot determine token order of pair: %s", err)
	}

	go func() {
		opts := &bind.CallOpts{
			Pending:     true,
			BlockNumber: nil,
			Context:     ctx,
		}
		defer close(prices)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				reserves, err := pair.GetReserves(opts)
				if err != nil {
					log.Printf("Error calling pancakePair.GetReserves: %s\n", err)
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
				prices <- price
			}
		}
	}()
	return prices, nil
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
