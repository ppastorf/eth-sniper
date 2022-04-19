package config

import (
	"io/ioutil"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Raw struct {
		RpcUrl      string `yaml:"rpcUrl"`
		ChainID     int    `yaml:"chainID"`
		PrivateKey  string `yaml:"privateKey"`
		TargetToken string `yaml:"targetToken"`
		InToken     string `yaml:"inToken"`
	} `yaml:"config"`
	RpcUrl      string         `yaml:"-"`
	ChainID     int64          `yaml:"-"`
	PrivateKey  common.Address `yaml:"-"`
	TargetToken common.Address `yaml:"-"`
	InToken     common.Address `yaml:"-"`
}

func (c *Config) ParseValues() *Config {
	c.RpcUrl = c.Raw.RpcUrl
	c.ChainID = int64(c.Raw.ChainID)
	c.PrivateKey = common.HexToAddress(c.Raw.PrivateKey)
	c.TargetToken = common.HexToAddress(c.Raw.TargetToken)
	c.InToken = common.HexToAddress(c.Raw.InToken)
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
