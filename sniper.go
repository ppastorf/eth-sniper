package main

// estrategia de gas

import (
	"context"
	"log"
	"math/big"

	pancake "sniper/contracts/bsc/pancakeswap"
	"sniper/pkg/config"
	eth "sniper/pkg/eth"
	"sniper/pkg/swap"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

func main() {
	var err error
	ctx := context.Background()

	// configs
	configPath := "./sniper_config.yaml"
	conf, err := config.FromYaml(configPath)
	if err != nil {
		log.Fatalf("Failed to read configuration file %s: %s", configPath, err)
	}

	// node setup
	network := &eth.Network{RpcUrl: conf.RpcUrl}
	client, err := network.Connect(ctx)
	if (err != nil) || (!network.IsConnected()) {
		log.Fatalf("Failed to connect to network: %s\n", err)
	}
	log.Printf("Connected to network via RPC node at %s", network.RpcUrl)

	// wallet setup
	wallet, err := eth.NewWallet(conf.Raw.PrivateKey, conf.ChainID)
	if err != nil {
		log.Fatalf("Failed to instantiate Wallet: %s\n", err.Error())
	}
	log.Printf("Using wallet of address %s", wallet.Address())

	ethBalance, err := wallet.GetEthBalance(client, ctx, params.Ether)
	if err != nil {
		log.Fatalf("Failed to get ETH balance: %s", err)
	}
	log.Printf("Current ETH balance: %f \n", ethBalance)

	// need to make contract calls
	opts := &bind.CallOpts{
		Pending:     false,
		From:        wallet.Address(),
		BlockNumber: nil,
		Context:     ctx,
	}

	// input token
	inToken, err := eth.NewToken(client, opts, conf.InTokenAddr)
	if err != nil {
		log.Fatalf("Failed to instantiate input Token: %s\n", err)
	}

	// target token
	targetToken, err := eth.NewToken(client, opts, conf.TargetTokenAddr)
	if err != nil {
		log.Fatalf("Failed to instantiate input Token: %s\n", err)
	}

	// DEX contracts
	DEX, err := swap.SetupDex(client, conf.FactoryAddress, conf.RouterAddress)
	if err != nil {
		log.Fatalf("Failed to setup DEX client: %s\n", err)
	}

	// bot logic

	// get pair
	ctx, cancel := context.WithCancel(context.Background())

	pairAddr, _ := DEX.FactoryClient.GetPair(opts, conf.InTokenAddr, conf.TargetTokenAddr)
	if pairAddr != *new(common.Address) {
		log.Printf("Liquidity pair for specified tokens exists: %s\n", pairAddr)
	} else {
		log.Printf("No pair found for specified tokens\n")
		pairsCreated := pairCreatedTrigger(client, ctx, DEX.FactoryContract, conf.InTokenAddr, conf.TargetTokenAddr)
		for addr := range pairsCreated {
			log.Printf("PAIR CREATED AT ADDRESS %s!\n", addr.Hex())
			pairAddr = addr
			cancel()
		}
	}

	pairContract, err := eth.NewContract(pairAddr, pancake.PancakePairMetaData)
	if err != nil {
		log.Fatalf("Failed to instantiate Pair contract: %s", err)
	}

	// get liquidity
	inTokenLiquidity, err := inToken.BalanceOf(opts, pairAddr)
	if err != nil {
		log.Fatalf("Failed to get balance for Pair contract %s", err)
	}
	targetTokenLiquidity, err := targetToken.BalanceOf(opts, pairAddr)
	if err != nil {
		log.Fatalf("Failed to get balance for Pair contract %s", err)
	}
	log.Printf("%s liquidity: %f\n", inToken.Symbol, eth.FromWei(inTokenLiquidity, params.Ether))
	log.Printf("%s liquidity: %f\n", targetToken.Symbol, eth.FromWei(targetTokenLiquidity, params.Ether))

	// if inTokenLiquidity.Cmp(conf.MinLiquidity) != -1 {
	// 	log.Fatalf("Liquidity at contract is already equal or higher than minimium %f", eth.FromWei(conf.MinLiquidity, params.Ether))
	// }

	ctx, cancel = context.WithCancel(context.Background())
	liquidityAdded := liquidityAddedTrigger(client, ctx, pairContract)
	for _ = range liquidityAdded {
		// totalLiq := new(big.Int).Add(inTokenLiquidity, amount)
		// if totalLiq.Cmp(conf.MinLiquidity) == -1 {
		// 	continue
		// }

		sw := swap.DexSwap{
			FromWallet:  wallet,
			SwapFunc:    swap.ExactEthForTokens,
			TokenIn:     conf.InTokenAddr,
			TokenOut:    conf.TargetTokenAddr,
			Amount:      conf.BuyAmount,
			GasStrategy: "fast",
			Expiration:  big.NewInt(60 * 60),
		}

		tx, err := sw.BuildTx(client, ctx, DEX.RouterClient)
		if err != nil {
			log.Fatalf("Failed to build swap transaction: %s\n", err)
		}

		err = client.SendTransaction(ctx, tx)
		if err != nil {
			log.Printf("Failed to send transaction: %s\n", err)
		}
		log.Printf("Transaction sent: %s", tx.Hash().Hex())

		receipt, err := bind.WaitMined(ctx, client, tx)
		if err != nil {
			log.Printf("Error waiting for transaction mining: %s\n", err)
		}
		log.Printf("Transaction mined: %s\n", tx.Hash().Hex())

		// todo
		tokensBought := eth.FromWei(new(big.Int), params.Ether)
		buyPrice := new(big.Float)

		gasUsed := new(big.Int).SetUint64(receipt.GasUsed)
		totalFee := new(big.Int).Mul(gasUsed, tx.GasPrice())

		log.Printf(
			`Sniped %f %s at %f %s
				Spent: %f %s
				Fees: %f %s
				Total cost: %f %s`,
			tokensBought, targetToken.Symbol, buyPrice, inToken.Symbol,
			eth.FromWei(tx.Value(), params.Ether), inToken.Symbol,
			eth.FromWei(totalFee, params.Ether), inToken.Symbol,
			eth.FromWei(tx.Cost(), params.Ether), inToken.Symbol,
		)
		cancel()
	}
}
