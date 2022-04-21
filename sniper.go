package main

// estrategia de gas

import (
	"context"
	"fmt"
	"log"
	"math/big"

	pancake "sniper/contracts/bsc/pancakeswap"
	"sniper/contracts/tokens"
	"sniper/pkg/config"
	eth "sniper/pkg/eth"
	"sniper/pkg/swap"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

func main() {
	var err error
	ctx := context.Background()

	// configs
	configPath := "./sniper_config.yaml"
	conf, err := config.FromYaml(configPath)
	if err != nil {
		log.Fatalf("Failed to read configuration file %s: %s", configPath, err)
	}

	// Wallet / network setup
	network := &eth.Network{RpcUrl: conf.RpcUrl}
	client, err := network.Connect(ctx)
	if (err != nil) || (!network.IsConnected()) {
		log.Fatalf("Failed to connect to network: %s\n", err)
	} else {
		log.Printf("Connected to network via RPC node at %s", network.RpcUrl)
	}

	wallet, err := eth.NewWallet(conf.Raw.PrivateKey, conf.ChainID)
	if err != nil {
		log.Fatalf("Failed to instantiate Wallet: %s\n", err.Error())
	} else {
		log.Printf("Using wallet of address %s", wallet.Address())
	}

	// contracts setup
	opts := &bind.CallOpts{
		Pending:     true,
		From:        wallet.Address(),
		BlockNumber: nil,
		Context:     ctx,
	}

	factoryContract, err := swap.NewContract(
		common.HexToAddress("0xcA143Ce32Fe78f1f7019d7d551a6402fC5350c73"),
		pancake.PancakeFactoryMetaData,
	)
	if err != nil {
		log.Fatalf("Failed to instantiate PancakeFactory contract: %s", err)
	}
	factoryClient, err := pancake.NewPancakeFactory(factoryContract.Address, client)
	if err != nil {
		log.Fatalf("Failed to instantiate PancakeRouter contract client: %s\n", err)
	}

	routerContract, err := swap.NewContract(
		common.HexToAddress("0x10ED43C718714eb63d5aA57B78B54704E256024E"),
		pancake.PancakeRouterMetaData,
	)
	if err != nil {
		log.Fatalf("Failed to instantiate PancakeRouter contract: %s", err)
	}
	_, err = pancake.NewPancakeRouter(routerContract.Address, client)
	if err != nil {
		log.Fatalf("Failed to instantiate PancakeRouter contract client: %s\n", err)
	}

	inToken, err := tokens.NewErc20Token(conf.InTokenAddr, client)
	if err != nil {
		log.Fatalf("Failed to instatiate InToken contract: %s\n", err)
	}
	inTokenSymbol, err := inToken.Symbol(opts)
	if err != nil {
		log.Fatalf("Failed to get symbol of Token %s: %s", conf.InTokenAddr, err)
	}

	balance, err := inToken.BalanceOf(opts, wallet.Address())
	if err != nil {
		log.Fatalf("Failed to get %s balance of address %s: %s", inTokenSymbol, wallet.Address().Hex(), err)
	} else {
		log.Printf("Current %s balance: %f \n", inTokenSymbol, eth.FromWei(balance, params.Ether))
	}

	// swap setup
	_ = swap.DexSwap{
		FromWallet:   wallet,
		ContractFunc: swap.ExactEthForTokens,
		TokenIn:      conf.InTokenAddr,
		TokenOut:     conf.TargetTokenAddr,
		Amount:       eth.ToWei(big.NewFloat(0.001), params.Ether),
		GasStrategy:  "fast",
		Expiration:   big.NewInt(60 * 60),
	}

	var pairAddr common.Address

	pairAddr, err = factoryClient.GetPair(opts, conf.InTokenAddr, conf.TargetTokenAddr)
	if err != nil {
		log.Fatalf("Failed to get token pair: %s\n", err)
	}

	if pairAddr != *new(common.Address) {
		log.Printf("Liquidity pair for specified tokens already exists: %s\n", pairAddr)
	} else {
		log.Printf("No pair found for specified tokens.\n")
		pairCreated := pairCreatedTrigger(client, ctx, factoryContract, conf.InTokenAddr, conf.TargetTokenAddr)
		select {
		case <-ctx.Done():
			log.Fatalf("Done")
		case addr := <-pairCreated:
			fmt.Printf("PAIR CREATED AT ADDRESS %s!\n", addr.Hex())
			pairAddr = addr
			break
		}
	}

	pairContract, err := swap.NewContract(pairAddr, pancake.PancakePairMetaData)
	if err != nil {
		log.Fatalf("Failed to instantiate Pair contract: %s", err)
	}

	inTokenLiquidity, err := inToken.BalanceOf(opts, pairAddr)
	if err != nil {
		log.Fatalf("Failed to get balance for Pair contract %s", err)
	}

	log.Printf("Liquidity for Pair: %f %s\n", eth.FromWei(inTokenLiquidity, params.Ether), inTokenSymbol)

	var minLiquidity int64 = 1
	if inTokenLiquidity.Cmp(big.NewInt(minLiquidity)) == 1 {
		log.Fatalf("Liquidity at contract is higher than minimun specified %d. We are too late!", minLiquidity)
	} else {
		liquidityAdded := liquidityAddedTrigger(client, ctx, pairContract)
		select {
		case <-ctx.Done():
			log.Fatalf("Done")
		case <-liquidityAdded:
			fmt.Printf("BUY BUY BUY!\n")
		}
	}

}

// send tx
// tx, err := sw.BuildTx(client, ctx, routerClient)
// fmt.Printf("tx: \n%+v\n", tx)
// if err != nil {
// 	log.Fatalf("Failed to build swap transaction: %s\n", err)
// }
// go eth.SendTx(client, ctx, tx)
