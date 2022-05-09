package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"path/filepath"
	"sniper/pkg/eth"
	"sniper/pkg/triggers"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"gopkg.in/yaml.v2"
)

type ConfigFile struct {
	PrivateKey string `yaml:"privateKey"`
	Network    struct {
		Rpc struct {
			Url      string `yaml:"url"`
			Login    string `yaml:"login"`
			Password string `yaml:"password"`
		} `yaml:"rpc"`
		RpcLogin       string `yaml:"rpcLogin"`
		RpcPassword    string `yaml:"rpcPassword"`
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
		Deadline           string   `yaml:"deadline"`
		LiquidityProviders []string `yaml:"liquidityProviders"`
	} `yaml:"buyTrigger"`
	SellTrigger struct {
		Deadline string `yaml:"deadline"`
	} `yaml:"sellTrigger"`
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

	BuyTrigger  triggers.BuyTrigger
	SellTrigger triggers.SellTrigger
}

func parseValues(raw ConfigFile) *Config {
	var err error
	c := new(Config)

	c.PrivateKey = raw.PrivateKey

	rpcAuth := ""
	if raw.Network.Rpc.Login != "" {
		rpcAuth = fmt.Sprintf("%s:%s@", raw.Network.Rpc.Login, raw.Network.Rpc.Password)
	}
	rpcUrl := strings.Split(raw.Network.Rpc.Url, "://")
	c.RpcUrl = fmt.Sprintf("%s://%s%s", rpcUrl[0], rpcAuth, rpcUrl[1])

	c.ChainID = raw.Network.ChainID
	c.RouterAddress = common.HexToAddress(raw.Network.RouterAddress)
	c.FactoryAddress = common.HexToAddress(raw.Network.FactoryAddress)
	c.EthSymbol = raw.Network.CoinSymbol

	c.InTokenAddr = common.HexToAddress(raw.InToken.Address)
	c.InTokenBuyAmount, err = eth.ToWei(big.NewFloat(raw.InToken.BuyAmount), params.Ether)
	if err != nil {
		log.Fatalf("Failed to parse buyAmount")
	}

	c.TargetTokenAddr = common.HexToAddress(raw.TargetToken.Address)
	c.TargetTokenStartingPrice = big.NewFloat(raw.TargetToken.StartingPrice)
	c.TargetTokenMaxBuyPrice = big.NewFloat(raw.TargetToken.MaxBuyPrice)

	c.BuyTrigger.Deadline, err = parseTimeStr("UTC", time.RFC3339, raw.BuyTrigger.Deadline)
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

	c.SellTrigger.Deadline, err = parseTimeStr("UTC", time.RFC3339, raw.SellTrigger.Deadline)
	if err != nil {
		log.Printf("Failed to parse sellDeadline, will not sell based on time.")
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
