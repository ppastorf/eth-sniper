package eth

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/assert"
)

var testCases = []struct {
	w string
	e string
}{
	{"500000000000000", "0.0005"},
	{"300000000000000", "0.0003"},
	{"149610870706426", "0.000149610870706426"},
	{"48230227969290", "0.00004823022796929"},
	{"59700000000000000", "0.0597"},
	{"77300000000000000", "0.0773"},
	{"91022452157233200", "0.0910224521572332"},
	{"83765446892566600", "0.0837654468925666"},
	{"515922633653434400", "0.5159226336534344"},
	{"921211090737990000", "0.92121109073799"},
	{"510400000000000000", "0.5104"},
	{"77400000000000000", "0.0774"},
	{"240020593675089000", "0.240020593675089"},
	{"189233317416414700", "0.1892333174164147"},
	{"28826000000000000000", "28.826"},
	{"24651400000000000000", "24.6514"},
	{"91856343949284520000", "91.85634394928452"},
	{"51865807966508150000", "51.86580796650815"},
	{"9390025900000000000000", "9390.0259"},
	{"1630303100000000000000", "1630.3031"},
	{"8228446298667040000000", "8228.44629866704"},
	{"8695571597247163000000", "8695.571597247163"},
	{"279722725800000000000000", "279722.7258"},
	{"327424024600000000000000", "327424.0246"},
	{"706403211048797500000000", "706403.2110487975"},
	{"516741918075423100000000", "516741.9180754231"},
	{"31340130400200000000000000", "31340130.4002"},
	{"161034803847600000000000000", "161034803.8476"},
	{"166588154314913780000000000", "166588154.31491378"},
	{"160640904641444030000000000", "160640904.64144403"},
}

func TestEtherFromWei(t *testing.T) {
	for _, tc := range testCases {
		wei, weiOk := new(big.Int).SetString(tc.w, 10)
		eth, ethOk := new(big.Float).SetString(tc.e)
		if !weiOk || !ethOk {
			t.Fail()
		}
		e := FromWei(wei, params.Ether)
		assert.Equal(t, e.Text('f', 18), eth.Text('f', 18), "fromWei did not convert right")
	}
}

func TestEtherToWei(t *testing.T) {
	for _, tc := range testCases {
		wei, weiOk := new(big.Int).SetString(tc.w, 10)
		eth, ethOk := new(big.Float).SetString(tc.e)
		if !weiOk || !ethOk {
			t.Fail()
		}
		w, err := ToWei(eth, params.Ether)
		if err != nil {
			t.Fail()
		}
		assert.Equal(t, w.String(), wei.String(), "toWei did not convert right")
	}
}

func FuzzEtherFromWei(f *testing.F) {
	for _, tc := range testCases {
		f.Add(tc.w)
	}
	f.Fuzz(func(t *testing.T, val string) {
		wei, ok := new(big.Int).SetString(val, 10)
		if !ok {
			t.Fail()
		}
		eth := FromWei(wei, params.Ether)

		reconverted, err := ToWei(eth, params.Ether)
		if err != nil {
			t.Fail()
		}

		assert.Equal(t, wei.String(), reconverted.String(), "fromWei did not convert right")
	})
}

func FuzzEtherToWei(f *testing.F) {
	for _, tc := range testCases {
		f.Add(tc.e)
	}

	f.Fuzz(func(t *testing.T, val string) {
		eth, ok := new(big.Float).SetString(val)
		if !ok {
			t.Fail()
		}
		wei, err := ToWei(eth, params.Ether)
		if err != nil {
			t.Fail()
		}

		reconverted := FromWei(wei, params.Ether)

		fmt.Printf("\n")
		assert.Equal(t, eth.Text('f', 18), reconverted.Text('f', 18), "toWei did not convert right")
	})
}
