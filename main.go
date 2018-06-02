package main

import (
	"fmt"
	"net/http"
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

func (b *Batch) PollTransactions() chan bool {
	ready := make(chan bool, 1)

	go func(){
		time.Sleep(1 * time.Second)
		fmt.Printf("PollTransactions() finished for batch %v\n", b)
		ready <- true
	}()
	fmt.Println("PollTransactions() early exit")
	return ready
}

type Mixer struct {
	Batches []*Batch
}

func (p *Mixer) Run() (err error){
	for _, b := range p.Batches {
		// go func(b *Batch){
		ready := b.PollTransactions()
		fmt.Printf("Blocked on batch %v's ready channel\n", b)
		<- ready
		fmt.Printf("Unblocked on batch %v's ready channel\n", b)
		b.Execute()
		// }(b)
	}
	return err
}

func main (){
	batch := &Batch{
		10,
		2,
		[]Address{
			Address("Address-1"),
		},
		[]Address{
			Address("Address-1"), Address("Address-2"), 
		},
	}
	batches := []*Batch{batch}
	mixer := &Mixer{batches}
	mixer.Run()
}

func init(){
	pool = NewJobcoinWallet("Pool")
}
