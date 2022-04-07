package eth

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// func TxReceiptDetails(receipt *types.Receipt) {
// }

func SendTxAndWait(client *ethclient.Client, ctx context.Context, tx *types.Transaction) (*types.Receipt, error) {
	err := client.SendTransaction(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("Failed to send transaction: %s\n", err)
	}

	fmt.Printf("Transaction sent: %s\n", tx.Hash().Hex())

	receipt, err := bind.WaitMined(ctx, client, tx)
	if err != nil {
		return nil, fmt.Errorf("Error waiting transaction to be mined: %s\n", err)
	}

	return receipt, err
}
