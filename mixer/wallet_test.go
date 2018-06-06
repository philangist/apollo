package mixer

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

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
				t.Errorf(
					"Request to send amount '%d' to recipient '%s' was unexpectedly successful", c.amount, c.recipient)
			}
		}
	}
}

func TestApiClientJSONGetRequest(t *testing.T){
	fmt.Println("Running TestApiClientJSONGetRequest...")

	apiClient := NewApiClient()
	fmt.Println(apiClient)

	apiClient.JSONGetRequest("")
	// test get with valid and invalid url
}

func TestApiClientJSONPostRequest(t *testing.T){
	fmt.Println("Running TestApiClientJSONPostRequest...")

	apiClient := NewApiClient()
	fmt.Println(apiClient)
	// test post with valid and invalid url
}

func TestBatchGeneratePayouts(t *testing.T) {
	fmt.Println("Running TestBatchGeneratePayouts...")

	amount := 120
	fee := 20

	source := Address("Address-1")
	recipients := []Address{
			Address("Address-1"), Address("Address-2"),
		}

	b := NewBatch(amount, fee, source, recipients)

	expected := amount - fee
	actual := 0
	payouts := b.GeneratePayouts(expected, len(recipients))

	for _, value := range payouts {
		actual += value
	}

	if actual != expected {
		t.Errorf(
			"Expected Batch.GeneratePayouts() to return a list of values that sum up to %d\nReceived %v which sums up to %d instead'",
			expected, payouts, actual)
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
