package swap

import (
	"context"
	"fmt"
	"math/big"
	"time"

	eth "sniper/pkg/eth"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type DexSwap struct {
	SwapFunc    swapFuncWrapper
	FromWallet  *eth.Wallet
	TokenIn     common.Address
	TokenOut    common.Address
	Amount      *big.Int
	Expiration  *big.Int
	GasStrategy string
}

func (s *DexSwap) GetTxDeadlineFromNow() *big.Int {
	now := big.NewInt(time.Now().Unix())
	return new(big.Int).Add(now, s.Expiration)
}

func (s *DexSwap) BuildTxOpts(client *ethclient.Client, ctx context.Context) (*bind.TransactOpts, error) {
	nonce, err := client.PendingNonceAt(ctx, s.FromWallet.Address())
	if err != nil {
		return nil, err
	}

	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, err
	}

	opts, err := s.FromWallet.GetSignerOpts()
	if err != nil {
		return nil, err
	}

	opts.Nonce = new(big.Int).SetUint64(nonce)
	opts.Value = s.Amount
	opts.GasPrice = gasPrice
	opts.GasLimit = 0
	opts.Context = ctx
	opts.NoSend = true

	return opts, err
}

func (s *DexSwap) BuildTx(client *ethclient.Client, ctx context.Context, router DexRouter) (*types.Transaction, error) {
	opts, err := s.BuildTxOpts(client, ctx)
	if err != nil {
		return nil, fmt.Errorf("Failed to build swap transaction options: %s\n", err)
	}

	tx, err := s.SwapFunc(router, s, opts)
	if err != nil {
		return nil, fmt.Errorf("Failed to build contract method call: %s\n", err)
	}

	return tx, nil
}

// Supported swap methods
type swapFuncWrapper func(router DexRouter, swap *DexSwap, opts *bind.TransactOpts) (*types.Transaction, error)

func ExactEthForTokens(router DexRouter, swap *DexSwap, opts *bind.TransactOpts) (*types.Transaction, error) {
	return router.SwapExactETHForTokensSupportingFeeOnTransferTokens(
		opts,
		big.NewInt(0), // amountOutMin
		[]common.Address{swap.TokenIn, swap.TokenOut},
		swap.FromWallet.Address(),
		swap.GetTxDeadlineFromNow(),
	)
}

func ExactTokensForEth(router DexRouter, swap *DexSwap, opts *bind.TransactOpts) (*types.Transaction, error) {
	return router.SwapExactTokensForETHSupportingFeeOnTransferTokens(
		opts,
		swap.Amount,   // amountIn
		big.NewInt(0), // amountOutMin
		[]common.Address{swap.TokenIn, swap.TokenOut},
		swap.FromWallet.Address(),
		swap.GetTxDeadlineFromNow(),
	)
}
