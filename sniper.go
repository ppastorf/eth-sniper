package main

// - API de configuração
// - estrategia de gas
// - agendamento de transação
// - ouvir eventos de pair created da blockchain para começar a comprar
// - transacoes em paralelo
//   - dependente do valor max por transacao e do valor de entrada
//   - criação automatica de carteiras
// - otmimização de gas

import (
	"context"
	"fmt"
	"log"
	"strings"

	pancake "sniper/contracts/bsc/pancakeswap"
	eth "sniper/internal/eth"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
)

type Contract struct {
	Address common.Address
	ABI     abi.ABI
}

func listenForPairCreated(client *ethclient.Client, ctx context.Context, factory Contract, targetToken common.Address) {
	eventSignature := crypto.Keccak256Hash([]byte("PairCreated(address,address,address,uint256)"))
	querySpec := eth.EventQuerySpec{
		Name:        "PairCreated",
		ContractABI: factory.ABI,
		Query: ethereum.FilterQuery{
			Addresses: []common.Address{factory.Address},
			Topics:    [][]common.Hash{{eventSignature}},
		},
	}

	pairsCreated := make(chan eth.Event)
	go eth.ListenForEvents(client, ctx, querySpec, pairsCreated)

	for pair := range pairsCreated {
		tokenA := pair.Topics[1].Hex()
		tokenB := pair.Topics[2].Hex()

		fmt.Printf("PairCreated: %s\n", pair.ParsedData[0])
		fmt.Printf("TokenA: %s\n", common.HexToAddress(tokenA))
		fmt.Printf("TokenB: %s\n", common.HexToAddress(tokenB))

		if (common.HexToAddress(tokenA) == targetToken) ||
			(common.HexToAddress(tokenB) == targetToken) {
			// buy trigger
			fmt.Printf("Its the target pair!")
			return
		}
	}
}

func main() {
	ctx := context.Background()

	wallet, err := eth.NewWallet("af44b8442594fd667f19779f79fc92e5f99aa8f150a3c0e171a942e33d9b8c08")
	if err != nil {
		log.Fatalf("Failed to create wallet: %s\n", err.Error())
	}

	chainstackNode := &eth.Node{
		Proto:    "wss",
		Username: "hopeful-ride",
		Password: "cinch-taco-mute-yummy-cussed-suds",
		Address:  "ws-nd-734-719-750.p2pify.com/ee56826bc7c085ae6b90d2114e6c1e28",
	}

	bsc := &eth.Network{
		ConnectAddress: chainstackNode.ConnectAddress(),
		EthCurrency: &eth.Token{
			Name:    "BNB",
			Address: common.HexToAddress("0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c"),
		},
	}

	client, err := bsc.Connect(ctx)
	if err != nil {
		log.Fatalf("Failed to connect to network: %s\n", err)
	}

	balance, err := wallet.GetEthBalance(client, ctx, params.Ether)
	if err != nil {
		log.Fatalf("Failed to get wallet balance at %s: %s", wallet.Address(), err)
	}
	fmt.Printf("Current balance: %f %s\n", balance, bsc.EthCurrency.Name)

	targetToken := common.HexToAddress("0x0E09FaBB73Bd3Ade0a17ECC321fD13a19e81cE82")

	factoryABI, err := abi.JSON(strings.NewReader(string(pancake.PancakeFactoryMetaData.ABI)))
	if err != nil {
		log.Fatal(err)
	}

	pancakeFactory := Contract{
		Address: common.HexToAddress("0xcA143Ce32Fe78f1f7019d7d551a6402fC5350c73"),
		ABI:     factoryABI,
	}

	listenForPairCreated(client, ctx, pancakeFactory, targetToken)

	// routerAddress := common.HexToAddress("0x10ED43C718714eb63d5aA57B78B54704E256024E")
	// pancakeRouter, err := pancake.NewPancakeRouter(routerAddress, client)
	// if err != nil {
	// 	log.Fatalf("Failed to instantiate PancakeSwap Router: %s\n", err)
	// }

	// bnbForCake := &swap.DexSwap{
	// 	Network:      bsc,
	// 	FromWallet:   wallet,
	// 	ContractFunc: swap.ExactEthForTokens,
	// 	TokenIn:      bsc.Currency,
	// 	TokenOut:     &eth.Token{Address: targetToken, Name: "CAKE"},
	// 	Amount:       eth.ToWei(big.NewFloat(0.001), params.Ether),
	// 	GasStrategy:  "fast",
	// 	Expiration:   big.NewInt(60 * 60),
	// }

	// tx, err := bnbForCake.BuildTx(client, ctx, pancakeRouter)
	// if err != nil {
	// 	log.Fatalf("Failed to build swap transaction: %s\n", err)
	// }

	// receipt, err := eth.SendTxAndWait(client, ctx, tx)
	// if err != nil {
	// 	log.Fatalf("Failed to send transaction: %s\n", err)
	// }

	// fmt.Printf("details: %+v\n", receipt)
}
