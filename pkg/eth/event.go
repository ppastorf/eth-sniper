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
	Data []interface{}
}

func ListenForEvents(client *ethclient.Client, ctx context.Context, spec EventQuerySpec) <-chan Event {
	received := make(chan Event)
	log.Printf("Listening for %s events from %s...", spec.Name, spec.Query.Addresses)

	go func() {
		logs := make(chan types.Log)
		sub, err := client.SubscribeFilterLogs(ctx, spec.Query, logs)
		if err != nil {
			log.Fatal(err)
		}
		defer sub.Unsubscribe()
		defer close(received)

		for {
			select {
			case <-ctx.Done():
				return
			case err := <-sub.Err():
				log.Printf("Received error from event logs: %s", err)
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
	}()

	return received
}
