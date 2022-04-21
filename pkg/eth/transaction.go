package eth

import (
	"context"
	"log"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

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
