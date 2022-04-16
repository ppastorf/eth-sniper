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
	"math/big"

	pancake "sniper/contracts/bsc/pancakeswap"
	eth "sniper/internal/eth"
	"sniper/internal/swap"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
)

func listenForPairCreated(client *ethclient.Client, ctx context.Context, factory *eth.Contract, targetToken common.Address) {
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

	pairsCreated := make(chan eth.Event)
	go eth.ListenForEvents(client, ctx, querySpec, pairsCreated)

	for pair := range pairsCreated {
		tokenA := pair.Topics[1]
		tokenB := pair.Topics[2]

		if (tokenA == targetTokenHash) || (tokenB == targetTokenHash) {
			// buy trigger
			fmt.Printf("PairCreated: %s\n", pair.ParsedData[0])
			fmt.Printf("TokenA: %s\n", tokenA.Hex())
			fmt.Printf("TokenB: %s\n", tokenA.Hex())
			return
		}
	}
}

func sendTxAsync(client *ethclient.Client, ctx context.Context, tx *types.Transaction, sent chan<- common.Hash, mined chan<- *types.Receipt) {
	var err error

	err = client.SendTransaction(ctx, tx)
	if err != nil {
		log.Printf("Failed to send transaction: %s\n", err)

	}
	sent <- tx.Hash()

	receipt, err := bind.WaitMined(ctx, client, tx)
	if err != nil {
		log.Printf("Error waiting transaction to be mined: %s\n", err)
	}
	mined <- receipt
}

func logTransactions(sent <-chan common.Hash, mined <-chan *types.Receipt) {
	go func() {
		for tx := range sent {
			log.Printf("Transaction sent: %s", tx.Hex())
		}
		log.Printf("Done sending transactions\n")
	}()
	go func() {
		for receipt := range mined {
			b, err := receipt.MarshalJSON()
			if err != nil {
				log.Printf("%s", err)
			}
			log.Printf("Transaction mined !!: %s", string(b))
		}
		log.Printf("No more transactions to be mined\n")
	}()
}

func buyWhenTriggered(client *ethclient.Client, ctx context.Context, router swap.DexRouter, sw swap.DexSwap, sent chan common.Hash, mined chan *types.Receipt) {
	tx, err := sw.BuildTx(client, ctx, router)
	if err != nil {
		log.Fatalf("Failed to build swap transaction: %s\n", err)
	}
	go sendTxAsync(client, ctx, tx, sent, mined)
}

func main() {
	var err error
	ctx := context.Background()

	// configs
	rpcUrl := "wss://hopeful-ride:cinch-taco-mute-yummy-cussed-suds@ws-nd-734-719-750.p2pify.com/ee56826bc7c085ae6b90d2114e6c1e28"
	chainID := int64(56)

	walletPriv := "af44b8442594fd667f19779f79fc92e5f99aa8f150a3c0e171a942e33d9b8c08"
	inToken := common.HexToAddress("0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c")
	targetToken := common.HexToAddress("0x0E09FaBB73Bd3Ade0a17ECC321fD13a19e81cE82")

	// Wallet / network setup
	wallet, err := eth.NewWallet(walletPriv, chainID)
	if err != nil {
		log.Fatalf("Failed to instantiate Wallet: %s\n", err.Error())
	}

	network := &eth.Network{RpcUrl: rpcUrl}
	client, err := network.Connect(ctx)
	if err != nil {
		log.Fatalf("Failed to connect to network: %s\n", err)
	}

	balance, err := wallet.GetEthBalance(client, ctx, params.Ether)
	if err != nil {
		log.Fatalf("Failed to get wallet balance at %s: %s", wallet.Address(), err)
	}
	fmt.Printf("Current ETH balance: %f \n", balance)

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
	bnbForCake := swap.DexSwap{
		FromWallet:   wallet,
		ContractFunc: swap.ExactEthForTokens,
		TokenIn:      inToken,
		TokenOut:     targetToken,
		Amount:       eth.ToWei(big.NewFloat(0.001), params.Ether),
		GasStrategy:  "fast",
		Expiration:   big.NewInt(60 * 60),
	}

	maxParallelTx := 100
	txsSent := make(chan common.Hash, maxParallelTx)
	txsMined := make(chan *types.Receipt, maxParallelTx)

	go logTransactions(txsSent, txsMined)
	go listenForPairCreated(client, ctx, factoryContract, targetToken)
	buyWhenTriggered(client, ctx, routerClient, bnbForCake, txsSent, txsMined)

	// go listenForSellTrigger()
	// sellWhenTriggered(client, ctx, routerClient, bnbForCake, txsSent, txsMined)
}
