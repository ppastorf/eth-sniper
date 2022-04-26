package main

import (
	"context"
	"log"
	"math/big"
	eth "sniper/pkg/eth"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
)

func pairCreatedTrigger(client *ethclient.Client, ctx context.Context, factory *eth.Contract, inToken common.Address, targetToken common.Address) chan common.Address {
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
		for {
			select {
			case <-ctx.Done():
				return
			case pair := <-pairsCreated:
				tokenA := pair.Topics[1]
				tokenB := pair.Topics[2]

				if (tokenA == inTokenHash && tokenB == targetTokenHash) ||
					(tokenB == inTokenHash && tokenA == targetTokenHash) {

					address := pair.Data[0].(common.Address)
					if address == *new(common.Address) {
						break
					}

					log.Printf("PairCreated: %s\n", address)
					log.Printf("\tTokenA: %s\n", tokenA.Hex())
					log.Printf("\tTokenB: %s\n", tokenB.Hex())
					trigger <- address
				}
			}
		}
	}()

	return trigger
}

func liquidityAddedTrigger(client *ethclient.Client, ctx context.Context, pair *eth.Contract) chan *big.Int {
	trigger := make(chan *big.Int)

	eventSignature := crypto.Keccak256Hash([]byte("Mint(address,uint256,uint256)"))
	querySpec := eth.EventQuerySpec{
		Name:        "Mint",
		ContractABI: pair.ABI,
		Query: ethereum.FilterQuery{
			Addresses: []common.Address{pair.Address},
			Topics:    [][]common.Hash{{eventSignature}},
		},
	}

	tokensMinted := eth.ListenForEvents(client, ctx, querySpec)

	go func() {
		defer close(trigger)
		for {
			select {
			case <-ctx.Done():
				return
			case minted := <-tokensMinted:
				senderAddr := minted.Topics[1].Hex()
				amountA := minted.Data[0].(*big.Int)
				amountB := minted.Data[1].(*big.Int)

				log.Printf("Liquidity added by %s\n", senderAddr)
				log.Printf("\tAmountA: %f\n", eth.FromWei(amountA, params.Ether))
				log.Printf("\tAmountB: %f\n", eth.FromWei(amountB, params.Ether))
				trigger <- amountB // need to check which token is which and send liquidity amount for input token here
			}
		}
	}()

	return trigger
}

func sellTrigger(client *ethclient.Client, ctx context.Context, pair *eth.Contract) chan bool {
	trigger := make(chan bool)
	price := make(chan *big.Int)

	// TODO

	go func() {
		defer close(trigger)
		for {
			select {
			case <-ctx.Done():
				return
			case <-price:
				trigger <- true
			}
		}
	}()
	return trigger
}
