package triggers

import (
	"context"
	"log"
	"time"

	eth "sniper/pkg/eth"
	"sniper/pkg/swap"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
)

type TxFilter struct {
	From              []common.Address
	To                []common.Address
	Methods           []string
	TargetTokenFields []string
}

type BuyTrigger struct {
	Deadline      *time.Time
	MempoolFilter TxFilter
}

func (bt *BuyTrigger) Set(client *ethclient.Client, geth *gethclient.Client, DEX *swap.Dex, targetToken *eth.Token) <-chan struct{} {
	trigger := make(chan struct{})
	fire := func() { trigger <- struct{}{} }

	deadline, _ := context.WithCancel(context.Background())
	if bt.Deadline != nil {
		deadline, _ = context.WithDeadline(deadline, *bt.Deadline)
		log.Printf("Set deadline to buy at %s", bt.Deadline.String())
	}

	go func() {
		defer close(trigger)

		pendingTxs := ListenForPendingTxs(geth, client, deadline)

		for {
			select {
			case <-deadline.Done():
				log.Printf("Buy deadline reached")
				fire()
				return
			case tx := <-pendingTxs:
				if bt.isTargetTransaction(client, DEX.RouterContract.ABI, targetToken, tx) {
					log.Printf("Found target transaction")
					fire()
					return
				}
			}
		}
	}()

	return trigger
}

func (bt *BuyTrigger) isTargetTransaction(client *ethclient.Client, contractAbi abi.ABI, targetToken *eth.Token, tx *types.Transaction) bool {
	to := tx.To()
	if to == nil {
		return false
	}
	if len(bt.MempoolFilter.To) > 0 {
		if !arrContains(bt.MempoolFilter.To, *to) {
			return false
		}
	}

	from, err := eth.GetTxSender(client, tx)
	if err != nil {
		return false
	}
	if len(bt.MempoolFilter.From) > 0 {
		if !arrContains(bt.MempoolFilter.From, *from) {
			return false
		}
	}

	method, args, err := eth.GetTxCallData(contractAbi, tx)
	if err != nil {
		return false
	}
	if len(bt.MempoolFilter.Methods) > 0 {
		if !arrContains(bt.MempoolFilter.Methods, method.Name) {
			return false
		}
	}

	if len(bt.MempoolFilter.TargetTokenFields) > 0 {
		if !argsContainsValueOnOneOfTheseField(args, targetToken.Address, bt.MempoolFilter.TargetTokenFields) {
			return false
		}
	}

	return true
}

func argsContainsValueOnOneOfTheseField[T comparable](args map[string]interface{}, targetValue T, possibleFields []string) bool {
	for _, field := range possibleFields {
		value, ok := args[field].(T)
		if !ok {
			continue
		}
		if value == targetValue {
			return true
		}
	}
	return false
}

func arrContains[T comparable](arr []T, e T) bool {
	for _, a := range arr {
		if a == e {
			return true
		}
	}
	return false
}

func ListenForPendingTxs(gethClient *gethclient.Client, client *ethclient.Client, ctx context.Context) <-chan *types.Transaction {
	txHashes := make(chan common.Hash)
	txs := make(chan *types.Transaction)
	log.Printf("Listening for pending transactions from node mempool...\n")

	go func() {
		sub, err := gethClient.SubscribePendingTransactions(ctx, txHashes)
		if err != nil {
			log.Fatalf("Failed to subscribe to transactions mempool: %s\n", err)
		}
		defer sub.Unsubscribe()
		defer close(txs)

		for {
			select {
			case <-ctx.Done():
				return
			case err := <-sub.Err():
				log.Fatalf("Received error from mempool subscription: %s\n", err)
			case hash := <-txHashes:
				tx, _, err := client.TransactionByHash(ctx, hash)
				if err == nil {
					txs <- tx
				}
			}
		}
	}()

	return txs
}
