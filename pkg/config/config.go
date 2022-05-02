package config

import (
	"errors"
	"io/ioutil"
	"log"
	"math/big"
	"path/filepath"
	"sniper/pkg/eth"
	"sniper/pkg/triggers"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"gopkg.in/yaml.v2"
)

type ConfigFile struct {
	PrivateKey string `yaml:"privateKey"`
	Network    struct {
		RpcUrl         string `yaml:"rpcUrl"`
		ChainID        int64  `yaml:"chainID"`
		FactoryAddress string `yaml:"factoryAddress"`
		RouterAddress  string `yaml:"routerAddress"`
		CoinSymbol     string `yaml:"coinSymbol"`
	} `yaml:"network"`
	InToken struct {
		Address   string  `yaml:"address"`
		BuyAmount float64 `yaml:"buyAmount"`
	} `yaml:"inputToken"`
	TargetToken struct {
		Address       string  `yaml:"address"`
		StartingPrice float64 `yaml:"startingPrice"`
		MaxBuyPrice   float64 `yaml:"maxBuyPrice"`
	} `yaml:"targetToken"`
	BuyTrigger struct {
		BuyDeadline        string   `yaml:"deadline"`
		LiquidityProviders []string `yaml:"liquidityProviders"`
	} `yaml:"buyTrigger"`
}

type Config struct {
	PrivateKey string

	RpcUrl         string
	ChainID        int64
	FactoryAddress common.Address
	RouterAddress  common.Address
	EthSymbol      string

	InTokenAddr      common.Address
	InTokenBuyAmount *big.Int

	TargetTokenAddr          common.Address
	TargetTokenStartingPrice *big.Float
	TargetTokenMaxBuyPrice   *big.Float

	BuyTrigger triggers.BuyTrigger
}

func parseValues(raw ConfigFile) *Config {
	var err error
	c := new(Config)

	c.PrivateKey = raw.PrivateKey

	c.RpcUrl = raw.Network.RpcUrl
	c.ChainID = raw.Network.ChainID
	c.RouterAddress = common.HexToAddress(raw.Network.RouterAddress)
	c.FactoryAddress = common.HexToAddress(raw.Network.FactoryAddress)
	c.EthSymbol = raw.Network.CoinSymbol

	c.InTokenAddr = common.HexToAddress(raw.InToken.Address)
	c.InTokenBuyAmount = eth.ToWei(big.NewFloat(raw.InToken.BuyAmount), params.Ether)

	c.TargetTokenAddr = common.HexToAddress(raw.TargetToken.Address)
	c.TargetTokenStartingPrice = big.NewFloat(raw.TargetToken.StartingPrice)
	c.TargetTokenMaxBuyPrice = big.NewFloat(raw.TargetToken.MaxBuyPrice)

	c.BuyTrigger.Deadline, err = parseTimeStr("UTC", time.RFC3339, raw.BuyTrigger.BuyDeadline)
	if err != nil {
		log.Printf("Failed to parse buyDeadline, will not buy based on time.")
	}

	var providers []common.Address
	for _, str := range raw.BuyTrigger.LiquidityProviders {
		providers = append(providers, common.HexToAddress(str))
	}

	c.BuyTrigger.MempoolFilter = triggers.TxFilter{
		From:              providers,
		To:                []common.Address{c.RouterAddress},
		Methods:           []string{"addLiquidityETH", "addLiquidity"},
		TargetTokenFields: []string{"token", "tokenA", "tokenB"},
	}

	return c
}

func FromYaml(path string) (c *Config, err error) {
	var raw ConfigFile
	f, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	bytes, err := ioutil.ReadFile(f)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(bytes, &raw)
	if err != nil {
		return nil, err
	}

	return parseValues(raw), nil
}

func parseTimeStr(location string, layout string, str string) (*time.Time, error) {
	loc, err := time.LoadLocation(location)
	if err != nil {
		return nil, errors.New("failed to parse location")
	}
	t, err := time.ParseInLocation(layout, str, loc)
	if err != nil {
		return nil, errors.New("failed to parse time string")
	}
	return &t, nil
}
