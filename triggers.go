package main

import (
	"context"
	"fmt"
	"log"
	eth "sniper/pkg/eth"
	"sniper/pkg/swap"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

func pairCreatedTrigger(client *ethclient.Client, ctx context.Context, factory *swap.Contract, inToken common.Address, targetToken common.Address) chan common.Address {
	trigger := make(chan common.Address)

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
	inTokenHash := inToken.Hash()
	targetTokenHash := targetToken.Hash()

	go func() {
		defer close(trigger)

		select {
		case <-ctx.Done():
			log.Printf("Done")
			return
		case pair := <-pairsCreated:
			tokenA := pair.Topics[1]
			tokenB := pair.Topics[2]

			if (tokenA == inTokenHash && tokenB == targetTokenHash) ||
				(tokenB == inTokenHash && tokenA == targetTokenHash) {

				address := common.HexToAddress(pair.Data[0].(string))
				fmt.Printf("PairCreated: %s\n", address)
				fmt.Printf("\tTokenA: %s\n", tokenA.Hex())
				fmt.Printf("\tTokenB: %s\n", tokenA.Hex())
				trigger <- address
				return
			}
		}
	}()

	return trigger
}

func liquidityAddedTrigger(client *ethclient.Client, ctx context.Context, pair *swap.Contract) chan common.Address {
	trigger := make(chan common.Address)

	eventSignature := crypto.Keccak256Hash([]byte("Mint(address,uint,uint)"))
	querySpec := eth.EventQuerySpec{
		Name:        "Mint",
		ContractABI: pair.ABI,
		Query: ethereum.FilterQuery{
			Addresses: []common.Address{pair.Address},
			Topics:    [][]common.Hash{{eventSignature}},
		},
	}

	lpTokensMinted := eth.ListenForEvents(client, ctx, querySpec)

	go func() {
		defer close(trigger)
		select {
		case <-ctx.Done():
			log.Printf("Done")
			return
		case minted := <-lpTokensMinted:
			senderAddr := minted.Topics[1].Hex()
			amountA := minted.Data[0].(uint)
			amountB := minted.Data[1].(uint)

			fmt.Printf("Liquidity added by %s\n", senderAddr)
			fmt.Printf("\tAmountA: %d\n", amountA)
			fmt.Printf("\tAmountB: %d\n", amountB)
			trigger <- common.HexToAddress(senderAddr)
			return
		}
	}()
	return trigger
}
