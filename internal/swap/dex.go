package swap

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type contractFuncWrapper func(router DexRouter, swap *DexSwap, opts *bind.TransactOpts) (*types.Transaction, error)

type DexRouter interface {
	SwapExactETHForTokens(*bind.TransactOpts, *big.Int, []common.Address, common.Address, *big.Int) (*types.Transaction, error)
}

func ExactEthForTokens(router DexRouter, swap *DexSwap, opts *bind.TransactOpts) (*types.Transaction, error) {
	return router.SwapExactETHForTokens(
		opts,
		big.NewInt(0),
		[]common.Address{swap.TokenIn, swap.TokenOut},
		swap.FromWallet.Address(),
		swap.GetTxDeadlineFromNow(),
	)
}
