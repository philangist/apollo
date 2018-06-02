package main

import (
	"fmt"
	"testing"
)

func TestCreateWallet(t *testing.T){
	fmt.Println("Running TestCreateWallet...")

	j := NewJobcoinWallet("Alice")
	expected := Address("Alice")

	if j.Address() != expected {
		t.Errorf("Jobcoin wallet was not created with expected address 'Alice'. Received '%s' instead", j.Address())
	}
}

func TestWalletSend(t *testing.T){
	fmt.Println("Running TestWalletSend...")

	j := NewJobcoinWallet("Alice")
	b := Address("Bob")

	err := j.Send(b, 100)

	if err != nil {
		t.Errorf("Jobcoin wallet did not successfully send amount '100' to recipient 'Bob'. Saw error '%s' instead", err)
	}
}
