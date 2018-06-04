package wallet

import (
	"fmt"
	"strconv"
	"sync"
	"time"
)

var (
	pool     *Wallet
)

func init() {
	pool = NewWallet("Pool")
}

type Batch struct {
	Amount     int
	Fee        int
	Source     Address
	Recipients []Address
	ready      chan bool
	StartTime  time.Time
}

func NewBatch(amount, fee int, source Address, recipients []Address) *Batch {
	return &Batch{
		amount,
		fee,
		source,
		recipients,
		make(chan bool),
		time.Now(),
	}
}

func (b *Batch) Tumble() (err error) {
	portion := (b.Amount - b.Fee) / len(b.Recipients)

	for _, recipient := range b.Recipients {
		err = pool.SendTransaction(recipient, portion)
		if err != nil {
			return err
		}
	}

	return err
}

func (b *Batch) PollTransactions() {
	source := b.Source
	fmt.Printf("b.StartTime: %s\nPolling address: %s\n", b.StartTime, source)
	w := &Wallet{NewApiClient(), source}
	seen := false
	timeout := time.Now().Add(15 * time.Second)

	for {
		if (timeout.After(time.Now())) {
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
