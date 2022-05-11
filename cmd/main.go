package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"

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

	tokensBought := eth.FromWei(new(big.Int), params.Ether)
	buyPrice := new(big.Float)

	gasUsed := new(big.Int).SetUint64(receipt.GasUsed)
	totalFee := new(big.Int).Mul(gasUsed, tx.GasPrice())

	log.Printf(
		`Sniped %f %s at %f %s
			Spent: %f %s
			Fees: %f %s
			Total cost: %f %s`,
		tokensBought, sw.TokenOut.Symbol, buyPrice, "BNB",
		eth.FromWei(tx.Value(), params.Ether), "BNB",
		eth.FromWei(totalFee, params.Ether), "BNB",
		eth.FromWei(tx.Cost(), params.Ether), "BNB",
	)

	return receipt, nil
}

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
	geth := gethclient.New(rpcCon)

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
		log.Fatalf("Failed to setup dex client: %s\n", err)
	}

	pricer, err := swap.NewPriceWatcher(client, dex, ctx, targetToken, inToken, true)
	if err != nil {
		log.Fatalf("Failed to setup target token price watchers: %s\n", err)
	}

	buySwap := &swap.DexSwap{
		FromWallet:  wallet,
		SwapFunc:    swap.ExactEthForTokens,
		TokenOut:    targetToken,
		TokenIn:     inToken,
		AmountIn:    conf.InTokenBuyAmount,
		GasStrategy: "fast",
		Expiration:  big.NewInt(60 * 60),
	}

	<-conf.BuyTrigger.Set(client, geth, dex, targetToken)
	_, err = buyTokens(client, buySwap, dex, pricer)
	if err != nil {
		log.Fatalf("Failed to buy tokens: %s\n", err)
	}

	// sellSwap := &swap.DexSwap{
	// }

	// <-conf.SellTrigger.Set(client, geth, dex, inToken, tokenPrices)
	// _, err = buyTokens(client, sellSwap, dex, targetToken)
	// if err != nil {
	// 	log.Fatalf("Failed to sell tokens: %s\n", err)
	// }
}
