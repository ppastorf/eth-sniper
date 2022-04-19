package eth

import (
	"fmt"
	"math/big"
	"strings"
)

func FromWei(wei *big.Int, unit float64) *big.Float {
	asFloat := new(big.Float).SetPrec(256).SetMode(big.ToNearestEven)
	weiFloat := new(big.Float).SetPrec(256).SetMode(big.ToNearestEven)

	return asFloat.Quo(weiFloat.SetInt(wei), big.NewFloat(unit))
}

func ToWei(val *big.Float, unit float64) *big.Int {
	valWei := val.Mul(val, big.NewFloat(unit))

	weiTxt := strings.Split(valWei.Text('f', 64), ".")[0]
	wei, ok := new(big.Int).SetString(weiTxt, 10)
	if !ok {
		fmt.Printf("erro na conversao: %v\n", weiTxt)
	}

	val.Int(wei)

	return wei
}
