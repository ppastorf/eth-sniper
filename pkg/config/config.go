package config

import (
	"io/ioutil"
	"math/big"
	"path/filepath"
	"sniper/pkg/eth"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Raw struct {
		RpcUrl          string  `yaml:"rpcUrl"`
		ChainID         int64   `yaml:"chainID"`
		PrivateKey      string  `yaml:"privateKey"`
		TargetTokenAddr string  `yaml:"targetToken"`
		InTokenAddr     string  `yaml:"inToken"`
		FactoryAddress  string  `yaml:"factoryAddress"`
		RouterAddress   string  `yaml:"routerAddress"`
		BuyAmount       float64 `yaml:"buyAmount"`
		MinLiquidity    float64 `yaml:"minLiquidity"`
	} `yaml:"config"`
	RpcUrl          string         `yaml:"-"`
	ChainID         int64          `yaml:"-"`
	PrivateKey      common.Address `yaml:"-"`
	TargetTokenAddr common.Address `yaml:"-"`
	InTokenAddr     common.Address `yaml:"-"`
	FactoryAddress  common.Address `yaml:"-"`
	RouterAddress   common.Address `yaml:"-"`
	BuyAmount       *big.Int       `yaml:"-"`
	MinLiquidity    *big.Int       `yaml:"-"`
}

func (c *Config) ParseValues() *Config {
	c.RpcUrl = c.Raw.RpcUrl
	c.ChainID = c.Raw.ChainID
	c.PrivateKey = common.HexToAddress(c.Raw.PrivateKey)
	c.TargetTokenAddr = common.HexToAddress(c.Raw.TargetTokenAddr)
	c.InTokenAddr = common.HexToAddress(c.Raw.InTokenAddr)
	c.FactoryAddress = common.HexToAddress(c.Raw.FactoryAddress)
	c.RouterAddress = common.HexToAddress(c.Raw.RouterAddress)
	c.BuyAmount = eth.ToWei(big.NewFloat(c.Raw.BuyAmount), params.Ether)
	c.MinLiquidity = eth.ToWei(big.NewFloat(c.Raw.MinLiquidity), params.Ether)
	return c
}

func FromYaml(path string) (*Config, error) {
	var c Config
	var err error
	f, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	bytes, err := ioutil.ReadFile(f)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(bytes, &c)
	if err != nil {
		return nil, err
	}
	return c.ParseValues(), nil
}
