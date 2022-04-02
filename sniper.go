package main

import (
    "fmt"
    "log"
	"strings"
	"context"
    "math/big"
    "crypto/ecdsa"
	// "github.com/ethereum/go-ethereum"
    "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
    "github.com/ethereum/go-ethereum/crypto"
    "github.com/ethereum/go-ethereum/ethclient"
)

type Token struct {
	Contract common.Address
	Repr string 
} 

type Wallet struct {
	publicKey *ecdsa.PublicKey
	privateKey *ecdsa.PrivateKey
	tokensHeld []*Token
}

func (w *Wallet) Address() common.Address {
	return crypto.PubkeyToAddress(*w.publicKey)
}

func NewWallet(privKey string) (*Wallet, error) {
	priv, err := crypto.HexToECDSA(privKey)
	if err != nil {
		return nil, err
	}
	
	pub, ok := priv.Public().(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("Failed to cast public key to ECDSA")
	}

	w := &Wallet{
		publicKey: pub,
		privateKey: priv,
	}

	return w, nil
}

func (w* Wallet) GetBalance(client *ethclient.Client, unit float64) (*big.Float, error) {
	balanceWei, err := client.BalanceAt(context.Background(), w.Address(), nil)
	if err != nil {
		return nil, err
	}
	
	return fromWei(balanceWei, unit), nil
}

func (w *Wallet) SignTransaction(tx *types.Transaction, network *Network) (*types.Transaction, error) {	
	return types.SignTx(tx, types.NewEIP155Signer(network.ChainId()), w.privateKey)
} 

func fromWei(wei *big.Int, unit float64) *big.Float {
	asFloat := new(big.Float).SetPrec(256).SetMode(big.ToNearestEven)
	weiFloat := new(big.Float).SetPrec(256).SetMode(big.ToNearestEven)

	return asFloat.Quo(weiFloat.SetInt(wei), big.NewFloat(unit))
}

func toWei(val *big.Float, unit float64) *big.Int {
	valWei := val.Mul(val, big.NewFloat(unit))
	
	weiTxt := strings.Split(valWei.Text('f', 64), ".")[0]
	wei, ok := new(big.Int).SetString(weiTxt, 10)
	if !ok {
		fmt.Printf("erro na conversao: %v\n", weiTxt)
	}

	val.Int(wei)

	return wei
}

type DexRouter struct {
	ContractAddress common.Address
	Abi string
}

type Network struct {
	ConnectAddress string
	BaseCurrency *Token
	chainId *big.Int
}

func (n *Network) SetChainId(client *ethclient.Client) (chainId *big.Int, err error) {
	chainId, err = client.NetworkID(context.Background())
	n.chainId = chainId
	return
}

func (n *Network) ChainId() *big.Int {
	return n.chainId
}

type SwapTransaction struct {
	Client      *ethclient.Client
	Router 		*DexRouter
	FromAddress common.Address
	TokenIn 	common.Address
	TokenOut 	common.Address
	AmountIn 	*big.Int
	GasStrategy string
}

func (swap *SwapTransaction) Build() (*types.Transaction, error) {
	ctx := context.Background()
	nonce, err := swap.Client.PendingNonceAt(ctx, swap.FromAddress)
	if err != nil {
		return nil, err
	}

	gasLimit := uint64(21000)

	gasPrice, err := swap.Client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, err
	}

	var data []byte

	tx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		GasPrice: gasPrice,
		Gas:      gasLimit,
		To:       &swap.Router.ContractAddress,
		Value:    swap.AmountIn,
		Data:     data,
	})

	return tx, nil
}

func main() {
	wallet, err := NewWallet()
	if err != nil {
		log.Fatalf("Failed to create wallet: %s\n", err.Error())
	}

	bsc := &Network{
		ConnectAddress: "https://bsc-dataseed.binance.org",
		BaseCurrency: &Token{common.HexToAddress("0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c"), "BNB"},
	}
	
    client, err := ethclient.Dial(bsc.ConnectAddress)
    if err != nil {
		log.Fatalf("Cannot connect to network at %s: %s", bsc.ConnectAddress, err)
    }

	balance, err := wallet.GetBalance(client, params.Ether)
	if err != nil {
        log.Fatalf("Failed to get wallet balance at %s: %s", wallet.Address(), err)
	}
	fmt.Printf("Current balance: %f %s\n", balance, bsc.BaseCurrency.Repr)

	pancakeSwapRouter := &DexRouter{
		ContractAddress: common.HexToAddress("0x10ED43C718714eb63d5aA57B78B54704E256024E"),
		// Abi: open("pancakeABI", "r").read().replace("\n", ""),
	}
	
	swap := &SwapTransaction{
		Client:		 client,
		Router: 	 pancakeSwapRouter,
		FromAddress: wallet.Address(),
		TokenIn:   	 common.HexToAddress("0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c"), // WBNB
		TokenOut:  	 common.HexToAddress("0x0E09FaBB73Bd3Ade0a17ECC321fD13a19e81cE82"), // CAKE
		AmountIn:  	 toWei(big.NewFloat(0.002), params.Ether),
		GasStrategy: "safe",
	}

	tx, err := swap.Build()
    if err != nil {
		log.Fatalf("Failed to build new transaction: %s", err)
    }

	fmt.Printf("%+v\n", tx)

	signedTx, err := wallet.SignTransaction(tx, bsc)
    if err != nil {
		log.Fatalf("Failed to sign transaction: %s", err)
    }

	fmt.Printf("%+v\n", signedTx)

	// ######
	// built_tx = tx.build()

	// signed_tx = wallet.sign_transaction(built_tx)

	// sent_tx = web3.eth.send_raw_transaction(signed_tx.rawTransaction)
	// print("sent transaction", web3.toHex(sent_tx))
		
	// txn_receipt = None
	// while txn_receipt is None and (time.time() < tx.expiration_time):
	// 	try:
	// 	txn_receipt = web3.eth.getTransactionReceipt(sent_tx)
	// 	if txn_receipt is not None: 
	// 		print(txn_receipt)
	// 		break
	// 	time.sleep(10)
	// 	except Exception as err:
	// 	print(err)

}