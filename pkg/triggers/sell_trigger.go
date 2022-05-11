package triggers

import (
	"context"
	"log"
	"math/big"
	"sniper/pkg/swap"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
)

type SellTrigger struct {
	Deadline *time.Time
}

func (st *SellTrigger) Set(client *ethclient.Client, geth *gethclient.Client, DEX *swap.Dex, tokenPrices <-chan *big.Float) <-chan struct{} {
	trigger := make(chan struct{})
	fire := func() { trigger <- struct{}{} }

	deadline, _ := context.WithCancel(context.Background())
	if st.Deadline != nil {
		deadline, _ = context.WithDeadline(deadline, *st.Deadline)
		log.Printf("Set deadline to sell at %s", st.Deadline.String())
	}

	go func() {
		defer close(trigger)

		for {
			select {
			case <-deadline.Done():
				log.Printf("Sell deadline reached\n")
				fire()
				return
			default:
				fire()
				return
			}
		}
	}()

	return trigger
}
