package mixer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"
)


func TestBatchGeneratePayouts(t *testing.T) {
	fmt.Println("Running TestBatchGeneratePayouts...")

	amount := Coin(120)
	fee := Coin(20)

	source := NewWallet(Address("Address-1"))
	recipients := []Address{
		Address("Address-1"), Address("Address-2"),
	}

	batch := NewBatch(amount, fee, source, recipients, 1)

	expected := amount - fee
	actual := Coin(0)
	payouts := batch.GeneratePayouts(expected, len(recipients))

	for _, value := range payouts {
		actual += value
	}

	if actual != expected {
		t.Errorf(
			"Expected Batch.GeneratePayouts() to return a list of values that sum up to %d\nReceived %v which sums up to %d instead",
			expected, payouts, actual)
	}
}

func TestNewMixer(t *testing.T) {
	mixer := NewMixer([]*Batch{})

	expected := HourScopedPool().Address
	actual := mixer.Pool().Address
	if actual != expected {
		t.Errorf("Mixer should've returned hour scoped pool address '%v'. Saw %v instead.", expected, actual)
	}
}

func TestMixerRun(t *testing.T) {
	fmt.Println("Running TestMixerRun...")

	future := time.Now().Add(time.Duration(1000) * time.Second)
	amount, _ := CoinFromString("10.00")

	txns := []*Transaction{
		&Transaction{
			future,
			"Alice",
			"Bob",
			amount, // 10.00
		},
	}

	client := &testClient{
		GetResponse:  func(url string) ([]byte, error) { return json.Marshal(txns) },
		PostResponse: func(url string, payload *bytes.Buffer) error { return nil },
	}
	w := &Wallet{client, "Bob"}

	batch := NewBatch(
		120,
		20,
		w,
		[]Address{
			Address("Address-1"), Address("Address-2"),
		},
		1,
	)

	batch.DelayGenerator = func(maxDelay int) int {
		return 0
	}

	poolGenerator := func() *Wallet {
		poolClient := &testClient{
			PostResponse: func(url string, payload *bytes.Buffer) error { return nil },
		}
		return &Wallet{poolClient, "Pool"}
	}

	batches := []*Batch{batch}     //, batch, batch}
	mixer := &Mixer{poolGenerator, batches, &sync.WaitGroup{}}
	mixer.Run()    // use recover/panic behavior here
}
