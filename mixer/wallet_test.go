package mixer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"
	"time"
)

const (
	TIMEOUT          = time.Second * 3
	HTTP_OK     = http.StatusOK
	HTTP_UNAVAILABLE = http.StatusServiceUnavailable
)

func mockHandler(status int, entity interface{}) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if status != 0 {
			w.WriteHeader(status)
		}

		if entity != nil {
			serialized, err := json.Marshal(entity)
			if err != nil {
				log.Panic(err)
			}
			w.Write(serialized)
		}
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
				t.Errorf(
					"Request to send amount '%d' to recipient '%s' was unexpectedly successful", c.amount, c.recipient)
			}
		}
	}
}

func mockTransaction() *Transaction {
	expected := &Transaction{
		time.Now(),
		Address("source"),
		Address("recipient"),
		"10",
	}

	return expected
}

func TestApiClientJSONGetRequest(t *testing.T){
	fmt.Println("Running TestApiClientJSONGetRequest...")

	mapping := map[string]string{
		"foo": "bar",
	}
	expected := []byte(`{"foo":"bar"}`)
	handler := mockHandler(HTTP_OK, mapping)
	tServer := httptest.NewServer(http.HandlerFunc(handler))
	defer tServer.Close()

	apiClient := NewApiClient()
	b, err := apiClient.JSONGetRequest(tServer.URL)
	if err != nil {
		t.Errorf("ApiClient.JSONGetRequest returned unexpected error %s", err)
	}
	if !reflect.DeepEqual(b, expected){
		t.Errorf("ApiClient.JSONGetRequest returned value '%q'\nexpected:'%q'\n", b, expected)
	}
}

func TestApiClientJSONGetInvalidRequest(t *testing.T){
	fmt.Println("Running TestApiClientJSONGetInvalidRequest...")

	mapping := map[string]string{
		"foo": "bar",
	}
	expected := []byte("")
	handler := mockHandler(HTTP_UNAVAILABLE, mapping)
	tServer := httptest.NewServer(http.HandlerFunc(handler))
	defer tServer.Close()

	apiClient := NewApiClient()
	b, err := apiClient.JSONGetRequest(tServer.URL)
	if err == nil {
		t.Errorf("ApiClient.JSONGetRequest was unexpectedly successful")
	}
	if fmt.Sprintf("%q", b) != fmt.Sprintf("%q", expected){
		t.Errorf("ApiClient.JSONGetRequest returned value '%q'\nexpected:'%q'\n", b, expected)
	}
}

func TestApiClientJSONPostRequest(t *testing.T){
	fmt.Println("Running TestApiClientJSONPostRequest...")

	handler := mockHandler(HTTP_OK, nil)
	tServer := httptest.NewServer(http.HandlerFunc(handler))
	defer tServer.Close()

	apiClient := NewApiClient()
	payload := bytes.NewBuffer([]byte(`{"foo":"bar"}`))
	err := apiClient.JSONPostRequest(tServer.URL, payload)
	if err != nil {
		t.Errorf("ApiClient.JSONPostRequest returned unexpected error %s", err)
	}
}

func TestApiClientJSONPostInvalidRequest(t *testing.T){
	fmt.Println("Running TestApiClientJSONPostRequest...")

	handler := mockHandler(HTTP_UNAVAILABLE, nil)
	tServer := httptest.NewServer(http.HandlerFunc(handler))
	defer tServer.Close()

	apiClient := NewApiClient()
	payload := bytes.NewBuffer([]byte(`{"foo":"bar"}`))
	err := apiClient.JSONPostRequest(tServer.URL, payload)
	if err == nil {
		t.Errorf("ApiClient.JSONPostRequest was unexpectedly successful")
	}
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
			"Expected Batch.GeneratePayouts() to return a list of values that sum up to %d\nReceived %v which sums up to %d instead",
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
