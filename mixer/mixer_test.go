package mixer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
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
	timeout := 1

	batch := NewBatch(amount, fee, source, recipients, timeout)
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
	fmt.Println("Running TestNewMixer...")

	mixer := NewMixer([]*Batch{})
	expected := HourlyPool().Address
	actual := mixer.Pool().Address

	if actual != expected {
		t.Errorf("Mixer should've returned hour scoped pool address '%v'. Saw %v instead.", expected, actual)
	}
}

func TestMixerRun(t *testing.T) {
	fmt.Println("Running TestMixerRun...")
	rand.Seed(time.Now().UnixNano())

	// build response when polling for txns on blockchain
	future := time.Now().Add(time.Duration(1000) * time.Second)
	amount := Coin(120)
	txns := []*Transaction{
		&Transaction{
			future,
			"Alice",
			"Bob",
			amount,
		},
	}

	bobGetCalls := 0
	bobPostCalls := 0
	client := &testClient{
		GetResponse: func(url string) ([]byte, error) {
			bobGetCalls += 1
			return json.Marshal(txns)
		},
		PostResponse: func(url string, payload *bytes.Buffer) error {
			bobPostCalls += 1
			return nil
		},
	}
	w := &Wallet{client, "Bob"}
	recipients := NewAddresses(rand.Intn(10) + 1)

	batch := NewBatch(120, 20, w, recipients, 1)
	batch.DelayGenerator = func(maxDelay int) int {
		return 0
	}
	batches := []*Batch{batch}

	poolPostCalls := 0
	poolGenerator := func() *Wallet {
		poolClient := &testClient{
			PostResponse: func(url string, payload *bytes.Buffer) error {
				poolPostCalls += 1
				return nil
			},
		}
		return &Wallet{poolClient, "Pool"}
	}
	mixer := &Mixer{poolGenerator, batches, &sync.WaitGroup{}}

	mixer.Run() // use recover/panic behavior here

	if (bobGetCalls != bobPostCalls) && (bobGetCalls != 1) {
		t.Errorf(
			"Expected bobGetCalls and bobPostCalls to have a value of 1. Saw values '%d' and '%d' respectively instead.",
			bobGetCalls, bobPostCalls,
		)
	}

	if poolPostCalls != len(recipients) {
		t.Errorf("Expected poolPostCalls to have a value of %d for recipients: %v. Saw '%d' instead.", poolPostCalls, recipients, len(recipients))
	}
}
