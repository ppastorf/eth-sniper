package eth

import (
	"context"
	"fmt"
	"log"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

func SendTx(client *ethclient.Client, ctx context.Context, tx *types.Transaction) {
	var err error

	err = client.SendTransaction(ctx, tx)
	if err != nil {
		log.Printf("Failed to send transaction: %s\n", err)
	}
	log.Printf("Transaction sent: %s", tx.Hash().Hex())

	receipt, err := bind.WaitMined(ctx, client, tx)
	if err != nil {
		log.Printf("Error waiting transaction mining: %s\n", err)
	}
	b, err := receipt.MarshalJSON()
	if err != nil {
		log.Printf("Cannot decode transaction receipt: %s", err)
	}
	log.Printf("Transaction mined: %s\n%s\n", tx.Hash().Hex(), string(b))
}

func GetTxSender(client *ethclient.Client, tx *types.Transaction) (*common.Address, error) {
	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		return nil, fmt.Errorf("Failed to get network Chain ID: %s", err)
	}

	msg, err := tx.AsMessage(types.NewEIP155Signer(chainID), nil)
	if err != nil {
		return nil, fmt.Errorf("Failed to get transaction as message: %s", err)
	}
	sender := msg.From()

	return &sender, nil
}

func GetTxCallData(contractABI abi.ABI, tx *types.Transaction) (method *abi.Method, args map[string]interface{}, err error) {
	callData := tx.Data()
	args = make(map[string]interface{}, 0)

	method, err = contractABI.MethodById(callData[:4])
	if err != nil {
		return nil, nil, fmt.Errorf("cannot decode tx method: %s", err)
	}

	err = method.Inputs.UnpackIntoMap(args, callData[4:])
	if err != nil {
		return method, nil, fmt.Errorf("cannot decode arguments data: %s\n", err)
	}

	return method, args, nil
}
