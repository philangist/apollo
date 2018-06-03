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
	Sources    []Address
	Recipients []Address
	ready      chan bool
	StartTime  time.Time
}

func NewBatch(amount, fee int, sources, recipients []Address) *Batch {
	return &Batch{
		amount,
		fee,
		sources,
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
	type pollMessage struct {
		address Address
		amount  int
	}

	poll := make(chan pollMessage)

	pollAddress := func(source Address) {
		fmt.Printf("b.StartTime: %s\nPolling address: %s\n", b.StartTime, source)

		for {
			sum := 0
			w := &Wallet{NewApiClient(), source}
			txns, _ := w.GetTransactions()

			for _, txn := range txns {
				if txn.Timestamp.Before(b.StartTime){
					continue
				}
				fmt.Printf("latest txn.Timestamp is %s\n", txn.Timestamp)
				amount, _ := strconv.ParseInt(txn.Amount, 10, 32)
				sum += int(amount)
			}

			fmt.Printf("Sum is %d\n", sum)

			if sum > 0 {
				poll <- pollMessage{source, sum}
				break
			}
			time.Sleep(5 * time.Second)
		}
	}

	// concurrent threads of execution which poll the blockchain for transactions
	// relevant to b.Sources
	for _, source := range b.Sources {
		go pollAddress(source)
	}

	// serial consumer of poll that makes sure that:
	// 1. every address in b.Sources has had a transaction sent to it
	// 2. the Jobcoins for each address are then forwarded to the central pool
	for j := 0; j < len(b.Sources); j++ {
		message := <-poll
		w := Wallet{NewApiClient(), message.address}
		w.SendTransaction(pool.Address, message.amount)
	}
	b.ready <- true
}

func (b *Batch) Run(wg *sync.WaitGroup) {
	go b.PollTransactions()
	for {
		select {
		case <-b.ready:
			b.Tumble()
			wg.Done()
		}
	}
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
