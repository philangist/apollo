package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

var pool *JobcoinWallet

type Address string

type JobcoinWallet struct {
	client *http.Client
	Address Address
}

func NewJobcoinWallet(address string) *JobcoinWallet {
	return &JobcoinWallet{
		&http.Client{}, Address(address),
	}
}

func (j *JobcoinWallet) Send(recipient Address, amount int) error {
	if amount <= 0 {
		return fmt.Errorf("amount should be a positive integer value")
	}

	fmt.Printf("Sending amount '%d' to recipient '%s'\n", amount, recipient)
	return nil
}

type Batch struct {
	Amount int
	Fee int
	Sources []Address
	Recipients []Address
	ready chan bool
}

func (b *Batch) Transfer () (err error) {
	pool.Send(pool.Address, b.Fee)
	portion := (b.Amount - b.Fee)/len(b.Recipients)

	for _, recipient := range b.Recipients {
		err = pool.Send(recipient, portion)
		if err != nil {
			return err
		}
	}

	return err
}

func (b *Batch) PollTransactions() {
	poll := make(chan bool, 1)

	// concurrent threads of execution which poll the blockchain for transactions
	// relevant to b.Sources
	for i := 0; i < len(b.Sources); i++ {
		go func(i int){
			poll <- true
		}(i)
	}

	// serial consumer of poll that makes sure every address in b.Sources
	// has had a transaction sent to it
	for j := 0; j < len(b.Sources); j++ {
		<- poll
	}
	b.ready <- true
}

func (b *Batch) Run (wg *sync.WaitGroup){
	go b.PollTransactions()
	for {
		select {
		case <- b.ready:
			b.Transfer()
			wg.Done()
		case <-time.After(5 * time.Second):
			// return errors.New("Timeout hit")
			wg.Done()
		}
	}
}

type Mixer struct {
	Batches []*Batch
	WaitGroup *sync.WaitGroup
}

func (m *Mixer) Run(){
	wg := m.WaitGroup
	for _, b := range m.Batches {
		wg.Add(1)
		go b.Run(wg)
	}
	wg.Wait()
}

func main (){
	batch := &Batch{
		10,
		2,
		[]Address{
			Address("Address-1"), Address("Address-2"), Address("Address-3"),
		},
		[]Address{
			Address("Address-1"), Address("Address-2"), 
		},
		make(chan bool),
	}
	batches := []*Batch{batch, batch, batch}
	mixer := &Mixer{batches, &sync.WaitGroup{}}
	mixer.Run()
}

func init(){
	pool = NewJobcoinWallet("Pool")
}
