package swap

import (
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

type Contract struct {
	Address common.Address
	ABI     abi.ABI
}

func NewContract(addr common.Address, metadata *bind.MetaData) (*Contract, error) {
	var err error
	ABI, err := abi.JSON(strings.NewReader(metadata.ABI))
	if err != nil {
		return nil, err
	}

	ctt := &Contract{
		Address: addr,
		ABI:     ABI,
	}
	return ctt, nil
}
