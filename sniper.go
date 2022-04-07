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

	pancake "sniper/contracts/bsc/pancakeswaprouter"
	eth "sniper/internal/eth"
	swap "sniper/internal/swap"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

func main() {
	ctx := context.Background()

	wallet, err := eth.NewWallet("")
	if err != nil {
		log.Fatalf("Failed to create wallet: %s\n", err.Error())
	}

	bsc := &eth.Network{
		ConnectAddress: "https://bsc-dataseed.binance.org",
		Currency:       &eth.Token{Address: common.HexToAddress("0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c"), Repr: "BNB"},
	}

	client, err := bsc.Connect(ctx)
	if err != nil {
		log.Fatalf("Failed to connect to network: %s\n", err)
	}

	balance, err := wallet.GetBalance(client, ctx, params.Ether)
	if err != nil {
		log.Fatalf("Failed to get wallet balance at %s: %s", wallet.Address(), err)
	}
	fmt.Printf("Current balance: %f %s\n", balance, bsc.Currency.Repr)

	pancakeRouter, err := pancake.NewPancakeswaprouter(common.HexToAddress("0x10ED43C718714eb63d5aA57B78B54704E256024E"), client)
	if err != nil {
		log.Fatalf("Failed to instantiate PancakeSwap Router: %s\n", err)
	}

	bnbForCake := &swap.DexSwap{
		Network:      bsc,
		FromWallet:   wallet,
		ContractFunc: swap.ExactEthForTokens,
		TokenIn:      bsc.Currency,
		TokenOut:     &eth.Token{Address: common.HexToAddress("0x0E09FaBB73Bd3Ade0a17ECC321fD13a19e81cE82"), Repr: "CAKE"},
		Amount:       eth.ToWei(big.NewFloat(0.001), params.Ether),
		GasStrategy:  "fast",
		Expiration:   big.NewInt(60 * 60),
	}

	tx, err := bnbForCake.BuildTx(client, ctx, pancakeRouter)
	if err != nil {
		log.Fatalf("Failed to build swap transaction: %s\n", err)
	}

	receipt, err := eth.SendTxAndWait(client, ctx, tx)
	if err != nil {
		log.Fatalf("Failed to send transaction: %s\n", err)
	}

	fmt.Printf("details: %+v\n", receipt)
}
