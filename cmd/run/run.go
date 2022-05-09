package main

import (
	"context"
	"log"
	"os"
	"sniper/pkg/config"
	eth "sniper/pkg/eth"
	"sniper/pkg/swap"

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

	DEX, err := swap.SetupDex(client, conf.FactoryAddress, conf.RouterAddress)
	if err != nil {
		log.Fatalf("Failed to setup DEX client: %s\n", err)
	}

	tokenPrice, err := DEX.SubscribeTokenPrice(client, ctx, targetToken, inToken)
	if err != nil {
		log.Fatalf("Failed to get token prices: %s\n", err)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case price := <-tokenPrice:
			log.Printf("Current %s/%s price: %f\n", targetToken.Symbol, inToken.Symbol, price)
		}
	}
}
