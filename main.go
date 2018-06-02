package main

import (
	"fmt"
	"net/http"
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

func main (){}

func init(){
	pool = NewJobcoinWallet("Pool")
}
