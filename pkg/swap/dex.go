package swap

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

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
