package main

import (
	// "errors"
	"fmt"
	"net/http"
	"sync"
	"time"
)

var pool *JobcoinWallet

type Address string

type Wallet interface {
	Address() Address
	Send(recipient *Address, amount int) error
}

type JobcoinWallet struct {
	client *http.Client
	address Address
}

func NewJobcoinWallet(address string) *JobcoinWallet {
	return &JobcoinWallet{
		client: &http.Client{},
		address: Address(address),
	}
}

func (j *JobcoinWallet) Send(recipient Address, amount int) error {
	if amount <= 0 {
		return fmt.Errorf("amount should be a positive integer value")
	}

	fmt.Printf("Sending amount '%d' to recipient '%s'\n", amount, recipient)
	return nil
}

func (j *JobcoinWallet) Address() Address {
	return j.address
}

type Batch struct {
	Amount int
	Fee int
	Sources []Address
	Recipients []Address
	ready chan bool
}

func (b *Batch) Execute () (err error) {
	pool.Send(pool.Address(), b.Fee)
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
	for i := 0; i < len(b.Sources); i++ {
		go func(i int){
			time.Sleep(1 * time.Second)
			fmt.Printf("fetched source i=%d within polling goroutine\n", i)
			poll <- true
		}(i)
	}

	for j := 0; j < len(b.Sources); j++ {
		fmt.Printf("blocked on source i=%d within iterator goroutine\n", j)
		<- poll
	}
	b.ready <- true
	fmt.Println("b.ready <- true")
}

type Mixer struct {
	Batches []*Batch
}

func (p *Mixer) Run(wg *sync.WaitGroup) (err error){
	for _, b := range p.Batches {
		wg.Add(1)
		go func(b *Batch){
			fmt.Println("Running b.PollTransactions()")
			go b.PollTransactions()
			for {
				select {
				case <- b.ready:
					fmt.Println("read <- b.ready")
					b.Execute()
					wg.Done()
				case <-time.After(5 * time.Second):
					fmt.Println("Timeout hit")
					// return errors.New("Timeout hit")
					wg.Done()
				}
			}
		}(b)
	}
	return err
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
	mixer := &Mixer{batches}
	wg := &sync.WaitGroup{}
	mixer.Run(wg)
	wg.Wait()
}

func init(){
	pool = NewJobcoinWallet("Pool")
}
