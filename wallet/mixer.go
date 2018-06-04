package wallet

import (
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"time"
)

var pool = NewWallet("Pool")

type Batch struct {
	Amount     int
	Fee        int
	Source     Address
	Recipients []Address
	StartTime  time.Time
}

func NewBatch(amount, fee int, source Address, recipients []Address) *Batch {
	return &Batch{
		amount,
		fee,
		source,
		recipients,
		time.Now(),
	}
}

func (b *Batch) Tumble() (err error) {
	portion := (b.Amount - b.Fee) / len(b.Recipients)

	for _, recipient := range b.Recipients {
		time.Sleep(time.Duration(rand.Intn(120)) * time.Second)
		err = pool.SendTransaction(recipient, portion)
		if err != nil {
			return err
		}
	}

	return err
}

func (b *Batch) PollTransactions() {
	fmt.Printf("b.StartTime: %s\nPolling address: %s\n", b.StartTime, b.Source)

	w := &Wallet{NewApiClient(), b.Source}
	seen := false
	timeout := time.Now().Add(time.Duration(5) * time.Second)

	for {
		if (timeout.Before(time.Now())) {
			return
		}

		txns, _ := w.GetTransactions(b.StartTime)
		for _, txn := range txns {
			fmt.Printf("new txn.Timestamp: %s\n", txn.Timestamp)
			amount, _ := strconv.ParseInt(txn.Amount, 10, 32)
			w.SendTransaction(pool.Address, int(amount))
			seen = true
		}
		if seen == false {
			time.Sleep(5 * time.Second)
			continue
		}
		b.Tumble()
		return
	}
}

func (b *Batch) Run(wg *sync.WaitGroup) {
	b.PollTransactions()
	wg.Done()
}

type Mixer struct {
	Batches   []*Batch
	WaitGroup *sync.WaitGroup
}

func NewMixer(batches []*Batch) *Mixer {
	return &Mixer{
		batches,
		&sync.WaitGroup{},
	}
}

func (m *Mixer) Run() {
	wg := m.WaitGroup
	for _, b := range m.Batches {
		wg.Add(1)
		go b.Run(wg)
	}
	wg.Wait()
}
