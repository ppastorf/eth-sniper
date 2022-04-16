package eth

import (
	"context"
	"log"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type EventQuerySpec struct {
	Name        string
	ContractABI abi.ABI
	Query       ethereum.FilterQuery
}

type Event struct {
	types.Log
	ParsedData []interface{}
}

func ListenForEvents(client *ethclient.Client, ctx context.Context, spec EventQuerySpec, received chan<- Event) {
	logs := make(chan types.Log)

	sub, err := client.SubscribeFilterLogs(ctx, spec.Query, logs)
	if err != nil {
		log.Fatal(err)
	}
	defer sub.Unsubscribe()
	defer close(received)

	for {
		select {
		case err := <-sub.Err():
			log.Printf("Received error from subscription: %s", err)
			return
		case eventLog := <-logs:
			parsedData, err := spec.ContractABI.Unpack(spec.Name, eventLog.Data)
			if err != nil {
				log.Fatal(err)
			}
			received <- Event{
				eventLog,
				parsedData,
			}
		}
	}
}
