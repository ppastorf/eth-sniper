package triggers

import (
	"context"
	"log"
	eth "sniper/pkg/eth"
	"sniper/pkg/swap"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
)

type SellTrigger struct {
	Deadline *time.Time
}

func (st *SellTrigger) Set(client *ethclient.Client, geth *gethclient.Client, DEX *swap.Dex, targetToken *eth.Token) <-chan struct{} {
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
				log.Printf("Sell deadline reached")
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
