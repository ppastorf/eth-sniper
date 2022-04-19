package main

// estrategia de gas

import (
	"context"
	"fmt"
	"log"
	"math/big"

	pancake "sniper/contracts/bsc/pancakeswap"
	"sniper/pkg/config"
	eth "sniper/pkg/eth"
	"sniper/pkg/swap"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
)

func listenForPairCreated(client *ethclient.Client, ctx context.Context, factory *eth.Contract, targetToken common.Address) chan bool {
	trigger := make(chan bool)

	targetTokenHash := targetToken.Hash()
	eventSignature := crypto.Keccak256Hash([]byte("PairCreated(address,address,address,uint256)"))

	querySpec := eth.EventQuerySpec{
		Name:        "PairCreated",
		ContractABI: factory.ABI,
		Query: ethereum.FilterQuery{
			Addresses: []common.Address{factory.Address},
			Topics:    [][]common.Hash{{eventSignature}},
		},
	}

	pairsCreated := eth.ListenForEvents(client, ctx, querySpec)

	go func() {
		select {
		case <-ctx.Done():
			log.Printf("Done")
			return
		case pair := <-pairsCreated:
			tokenA := pair.Topics[1]
			tokenB := pair.Topics[2]

			if (tokenA == targetTokenHash) || (tokenB == targetTokenHash) {
				fmt.Printf("PairCreated: %s\n", pair.ParsedData[0])
				fmt.Printf("TokenA: %s\n", tokenA.Hex())
				fmt.Printf("TokenB: %s\n", tokenA.Hex())
				trigger <- true
				return
			}
		}
	}()

	return trigger
}

func sendTx(client *ethclient.Client, ctx context.Context, tx *types.Transaction) {
	var err error

	err = client.SendTransaction(ctx, tx)
	if err != nil {
		log.Printf("Failed to send transaction: %s\n", err)
	}
	log.Printf("Transaction sent: %s", tx.Hash().Hex())

	receipt, err := bind.WaitMined(ctx, client, tx)
	if err != nil {
		log.Printf("Error waiting transaction to be mined: %s\n", err)
	}
	b, err := receipt.MarshalJSON()
	if err != nil {
		log.Printf("%s", err)
	}
	log.Printf("Transaction mined: %s\n%s\n", tx.Hash().Hex(), string(b))
}

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
	}
	log.Printf("Connected to network via RPC node at %s", network.RpcUrl)

	wallet, err := eth.NewWallet(conf.Raw.PrivateKey, conf.ChainID)
	if err != nil {
		log.Fatalf("Failed to instantiate Wallet: %s\n", err.Error())
	}
	balance, err := wallet.GetEthBalance(client, ctx, params.Ether)
	if err != nil {
		log.Fatalf("Failed to get wallet balance at %s: %s", wallet.Address(), err)
	}
	log.Printf("Using wallet of address %s", wallet.Address())
	log.Printf("Current ETH balance: %f \n", balance)

	// Dex contracts setup
	factoryContract, err := eth.NewContract("0xcA143Ce32Fe78f1f7019d7d551a6402fC5350c73", pancake.PancakeFactoryMetaData)
	if err != nil {
		log.Fatalf("Failed to instantiate PancakeFactory contract: %s", err)
	}
	routerContract, err := eth.NewContract("0xcA143Ce32Fe78f1f7019d7d551a6402fC5350c73", pancake.PancakeRouterMetaData)
	if err != nil {
		log.Fatalf("Failed to instantiate PancakeRouter contract: %s", err)
	}
	routerClient, err := pancake.NewPancakeRouter(routerContract.Address, client)
	if err != nil {
		log.Fatalf("Failed to instantiate PancakeRouter contract client: %s\n", err)
	}

	// swap setup
	sw := swap.DexSwap{
		FromWallet:   wallet,
		ContractFunc: swap.ExactEthForTokens,
		TokenIn:      conf.InToken,
		TokenOut:     conf.TargetToken,
		Amount:       eth.ToWei(big.NewFloat(0.001), params.Ether),
		GasStrategy:  "fast",
		Expiration:   big.NewInt(60 * 60),
	}

	buyTrigger := listenForPairCreated(client, ctx, factoryContract, conf.TargetToken)

	select {
	case <-buyTrigger:
		fmt.Printf("BUY BUY BUY!")
		_, err := sw.BuildTx(client, ctx, routerClient)
		if err != nil {
			log.Fatalf("Failed to build swap transaction: %s\n", err)
		}
		// go sendTx(client, ctx, tx)
	case <-ctx.Done():
		log.Fatalf("Done")
	}

	// sellTrigger := awaitSellTrigger(client, ctx, targetToken)
}
