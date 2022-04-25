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
}

type DexFactory interface {
	GetPair(opts *bind.CallOpts, arg0 common.Address, arg1 common.Address) (common.Address, error)
}

type Dex struct {
	FactoryClient   DexFactory
	FactoryContract *eth.Contract
	RouterClient    DexRouter
	RouterContract  *eth.Contract
}

func SetupDex(client bind.ContractBackend, factoryAddress, routerAddress common.Address) (*Dex, error) {
	factoryContract, err := eth.NewContract(factoryAddress, pancake.PancakeFactoryMetaData)
	if err != nil {
		return nil, fmt.Errorf("Failed to instantiate PancakeFactory contract: %s", err)
	}
	factoryClient, err := pancake.NewPancakeFactory(factoryAddress, client)
	if err != nil {
		return nil, fmt.Errorf("Failed to instantiate PancakeRouter contract client: %s\n", err)
	}

	routerContract, err := eth.NewContract(routerAddress, pancake.PancakeRouterMetaData)
	if err != nil {
		return nil, fmt.Errorf("Failed to instantiate PancakeRouter contract: %s", err)
	}
	routerClient, err := pancake.NewPancakeRouter(routerAddress, client)
	if err != nil {
		return nil, fmt.Errorf("Failed to instantiate PancakeRouter contract client: %s\n", err)
	}

	d := &Dex{
		FactoryClient:   factoryClient,
		FactoryContract: factoryContract,
		RouterClient:    routerClient,
		RouterContract:  routerContract,
	}
	return d, nil
}

// Supported swap methods

type swapFuncWrapper func(router DexRouter, swap *DexSwap, opts *bind.TransactOpts) (*types.Transaction, error)

func ExactEthForTokens(router DexRouter, swap *DexSwap, opts *bind.TransactOpts) (*types.Transaction, error) {
	return router.SwapExactETHForTokens(
		opts,
		big.NewInt(0), // amountOutMin
		[]common.Address{swap.TokenIn, swap.TokenOut},
		swap.FromWallet.Address(),
		swap.GetTxDeadlineFromNow(),
	)
}

func ExactTokensForEth(router DexRouter, swap *DexSwap, opts *bind.TransactOpts) (*types.Transaction, error) {
	return router.SwapExactTokensForETH(
		opts,
		swap.Amount,   // amountIn
		big.NewInt(0), // amountOutMin
		[]common.Address{swap.TokenIn, swap.TokenOut},
		swap.FromWallet.Address(),
		swap.GetTxDeadlineFromNow(),
	)
}
