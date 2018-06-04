package wallet

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestCreateWallet(t *testing.T) {
	fmt.Println("Running TestCreateWallet...")

	j := NewWallet("Alice")
	expected := Address("Alice")

	if j.Address != expected {
		t.Errorf("Wallet was not created with expected address 'Alice'. Received '%s' instead", j.Address)
	}
}

func TestWalletSendTransaction(t *testing.T) {
	fmt.Println("Running TestWalletSendTransaction...")

	j := NewWallet("Alice")
	b := Address("Bob")

	cases := []struct {
		recipient Address
		amount    int
		valid     bool
	}{
		{b, 100, true},
		{b, 0, false},
		{b, -100, false},
	}

	for _, c := range cases {
		err := j.SendTransaction(c.recipient, c.amount)
		if c.valid {
			if err != nil {
				t.Errorf("Did not successfully send amount '%d' to recipient '%s'. Saw error '%s' instead", c.amount, c.recipient, err)
			}
		} else {
			if err == nil {
				t.Errorf("Request to send amount '%d' to recipient '%s' was unexpectedly successful", c.amount, c.recipient)
			}
		}
	}
}

func TestBatchTumble(t *testing.T) {
	fmt.Println("Running TestBatchTumble...")

	b := &Batch{
		120,
		20,
		Address("Address-1"),
		[]Address{
			Address("Address-1"), Address("Address-2"), Address("Address-3"), Address("Address-4"), Address("Address-5"),
		},
		time.Now(),
	}
	err := b.Tumble()
	if err != nil {
		t.Errorf("Expected Batch.Execute() to run successfully, received error '%s'", err)
	}
}

func TestMixerRun(t *testing.T) {
	fmt.Println("Running TestMixerRun...")

	batch := &Batch{
		10,
		2,
		Address("Alice"),
		[]Address{
			Address("Address-1"), Address("Address-2"),
		},
		time.Now(),
	}
	batches := []*Batch{batch} //, batch, batch}
	mixer := &Mixer{batches, &sync.WaitGroup{}}
	mixer.Run()
}
