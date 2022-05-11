package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"
	pancake "sniper/contracts/bsc/pancakeswap"
	"sniper/pkg/config"
	eth "sniper/pkg/eth"
	"sniper/pkg/swap"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
)

func main() {
	var err error
	ctx := context.Background()

	conf, err := config.FromYaml(os.Args[1])
	if err != nil {
		log.Fatalf("Failed to read configuration file: %s", err)
	}

	network := &eth.Network{RpcUrl: conf.RpcUrl}
	client, err := network.Connect(ctx)
	if (err != nil) || (!network.IsConnected()) {
		log.Fatalf("Failed to connect to network: %s\n", err)
	}
	log.Printf("Connected to network via RPC node at %s", network.RpcUrl)

	rpcCon, err := rpc.Dial(conf.RpcUrl)
	if err != nil {
		log.Fatalf("Failed to connect to RPC Node: %s\n", err)
	}
	_ = gethclient.New(rpcCon)

	wallet, err := eth.NewWallet(conf.PrivateKey, conf.ChainID)
	if err != nil {
		log.Fatalf("Failed to instantiate Wallet: %s\n", err.Error())
	}
	log.Printf("Using wallet of address %s", wallet.Address())

	ethBalance, err := wallet.GetEthBalance(client, ctx, params.Ether)
	if err != nil {
		log.Fatalf("Failed to get %s balance: %s", conf.EthSymbol, err)
	}
	log.Printf("Current %s balance: %f \n", conf.EthSymbol, ethBalance)

	inToken, err := eth.NewToken(client, conf.InTokenAddr)
	if err != nil {
		log.Fatalf("Failed to instantiate input Token: %s\n", err)
	}

	targetToken, err := eth.NewToken(client, conf.TargetTokenAddr)
	if err != nil {
		log.Fatalf("Failed to setup target Token: %s\n", err)
	}

	dex, err := swap.SetupDex(client, conf.FactoryAddress, conf.RouterAddress)
	if err != nil {
		log.Fatalf("Failed to setup DEX client: %s\n", err)
	}

	pricer, err := swap.NewPriceWatcher(client, dex, ctx, targetToken, inToken, false)
	if err != nil {
		log.Fatalf("Failed to get token prices: %s\n", err)
	}

	sw := &swap.DexSwap{
		FromWallet:  wallet,
		SwapFunc:    swap.ExactEthForTokens,
		TokenIn:     inToken,
		AmountIn:    conf.InTokenBuyAmount,
		TokenOut:    targetToken,
		GasStrategy: "fast",
		Expiration:  big.NewInt(60 * 60),
	}

	_, err = buyTokens(client, sw, dex, pricer)
	if err != nil {
		log.Fatalf("Failed to buyt tokens: %s\n", err)
	}
}

func buyTokens(client *ethclient.Client, sw *swap.DexSwap, dex *swap.Dex, pricer *swap.PriceWatcher) (*types.Receipt, error) {
	var err error
	ctx := context.Background()

	tx, err := sw.BuildTx(client, ctx, dex.Router)
	if err != nil {
		return nil, fmt.Errorf("Failed to build swap transaction: %s", err)
	}

	err = client.SendTransaction(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("Failed to send transaction: %s", err)
	} else {
		log.Printf("Transaction sent: %s", tx.Hash().Hex())
	}

	receipt, err := bind.WaitMined(ctx, client, tx)
	if err != nil {
		log.Printf("Error waiting for transaction mining: %s\n", err)
	} else {
		log.Printf("Transaction mined: %s\n", tx.Hash().Hex())
	}
	pairABI, err := pancake.PancakePairMetaData.GetAbi()
	if err != nil {
		log.Fatalf("Cannot get pairAbi: %s\n", err)
	}

	// get swapOut
	var weiBought, weiSpent *big.Int
	for _, txLog := range receipt.Logs {
		logData, err := pairABI.Unpack("Swap", txLog.Data)
		if err != nil {
			continue
		}
		weiSpent = logData[1].(*big.Int)
		weiBought = logData[2].(*big.Int)
	}

	tokensBought := eth.FromWei(weiBought, params.Ether)
	tokensSpent := eth.FromWei(weiSpent, params.Ether)
	buyPrice, err := eth.TokenRatio(weiSpent, weiBought)
	if err != nil {
		log.Printf("Failed to get buy price: %s\n", err)
	}

	totalCost := tx.Cost()
	gasUsed := new(big.Int).SetUint64(receipt.GasUsed)
	gasFees := new(big.Int).Mul(gasUsed, tx.GasPrice())
	totalFees := new(big.Int).Sub(totalCost, weiSpent)
	dexFees := new(big.Int).Sub(totalFees, gasFees)

	log.Printf(
		`Bought %.18f %s for %.18f %s
			Buy price: %.18f %s per %s
			Gas fees: %.18f %s
			Dex fees: %.18f %s
			Total fees: %.18f %s
			Total cost: %.18f %s`,
		tokensBought, sw.TokenOut.Symbol, tokensSpent, sw.TokenIn.Symbol,
		buyPrice, sw.TokenIn.Symbol, sw.TokenOut.Symbol,
		eth.FromWei(gasFees, params.Ether), sw.TokenIn.Symbol,
		eth.FromWei(dexFees, params.Ether), sw.TokenIn.Symbol,
		eth.FromWei(totalFees, params.Ether), sw.TokenIn.Symbol,
		eth.FromWei(totalCost, params.Ether), sw.TokenIn.Symbol,
	)

	return receipt, nil
}
