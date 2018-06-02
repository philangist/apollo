package main

import (
	"fmt"
	"net/http"
)

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

func (b *Batch) Execute () {}

func main (){

}
