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
	HTTP_OK          = http.StatusOK
	HTTP_UNAVAILABLE = http.StatusServiceUnavailable
)

func TestCreateAddresses(t *testing.T){
	addresses := CreateAddresses(5)
	if len(addresses) != 5 {
		t.Errorf("mixer.CreateAddresses(5) did not create 5 addresses, receieved %d back instead", len(addresses))
	}
}

type testClient struct {
	GetResponse  func(url string) ([]byte, error)
	PostResponse func(url string, payload *bytes.Buffer) error
}

func (t *testClient) JSONGetRequest(url string) ([]byte, error) {
	return t.GetResponse(url)
}

func (t *testClient) JSONPostRequest(url string, payload *bytes.Buffer) error {
	return t.PostResponse(url, payload)
}

func TestWalletSendTransaction(t *testing.T) {
	fmt.Println("Running TestWalletSendTransaction...")

	client := &testClient{
		PostResponse: func(url string, payload *bytes.Buffer) error { return nil },
	}
	w := &Wallet{client, "Alice"}
	b := Address("Bob")

	cases := []struct {
		recipient Address
		amount    Coin
		valid     bool
	}{
		{b, Coin(100), true},
		{b, Coin(0), false},
		{b, Coin(-100), false},
	}

	for _, c := range cases {
		err := w.SendTransaction(c.recipient, c.amount)
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

func TestWalletGetTransactions(t *testing.T) {
	fmt.Println("Running TestWalletGetTransactions...")

	now := time.Now()
	past := now.Add(time.Duration(-1000) * time.Second)
	future := now.Add(time.Duration(1000) * time.Second)

	amount, _ := CoinFromString("10.00")
	fmt.Println("amount is ", amount)

	txns := []*Transaction{
		&Transaction{
			past,
			"Alice",
			"Bob",
			amount,
		},
		&Transaction{
			future,
			"Alice",
			"Bob",
			amount,
		},
	}

	client := &testClient{
		GetResponse: func(url string) ([]byte, error) {
			return json.Marshal(txns)
		},
	}

	w := &Wallet{client, "Bob"}
	returnedTxns, err := w.GetTransactions(now)
	if err != nil {
		t.Errorf("Did not successfully fetch transactions. Saw error '%s' instead", err)
	}

	expected := txns[1]
	actual := returnedTxns[0]

	expectedAmount := expected.Amount
	actualAmount := actual.Amount

	fmt.Println("expectedAmount, actualAmount ", expectedAmount, actualAmount)

	if !((expectedAmount == actualAmount) &&
		(expected.Timestamp.Equal(actual.Timestamp)) &&
		(expected.Source == actual.Source) &&
		(expected.Recipient == actual.Recipient)) {
		t.Errorf("Returned transaction %v did not match expected transaction %v", actual, expected)
	}
}

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

func TestApiClientJSONGetRequest(t *testing.T) {
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
	if !reflect.DeepEqual(b, expected) {
		t.Errorf("ApiClient.JSONGetRequest returned value '%q'\nexpected:'%q'\n", b, expected)
	}
}

func TestApiClientJSONGetInvalidRequest(t *testing.T) {
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
	if fmt.Sprintf("%q", b) != fmt.Sprintf("%q", expected) {
		t.Errorf("ApiClient.JSONGetRequest returned value '%q'\nexpected:'%q'\n", b, expected)
	}
}

func TestApiClientJSONGetInvalidURL(t *testing.T) {
	fmt.Println("Running TestApiClientJSONGetInvalidRequest...")

	handler := mockHandler(0, nil)
	tServer := httptest.NewServer(http.HandlerFunc(handler))
	defer tServer.Close()

	apiClient := NewApiClient()
	_, err := apiClient.JSONGetRequest("INVALID-URL")
	if err == nil {
		t.Errorf("ApiClient.JSONGetRequest was unexpectedly successful with an invalid url")
	}
}

func TestApiClientJSONPostRequest(t *testing.T) {
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

func TestApiClientJSONPostInvalidRequest(t *testing.T) {
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

func TestApiClientJSONPostInvalidURL(t *testing.T) {
	fmt.Println("Running TestApiClientJSONPostRequest...")

	handler := mockHandler(0, nil)
	tServer := httptest.NewServer(http.HandlerFunc(handler))
	defer tServer.Close()

	apiClient := NewApiClient()
	payload := bytes.NewBuffer([]byte(`{"foo":"bar"}`))
	err := apiClient.JSONPostRequest("INVALID-URL", payload)
	if err == nil {
		t.Errorf("ApiClient.JSONPostRequest was unexpectedly successful")
	}
}

func TestCoinFromString(t *testing.T) {
	cases := []struct {
		input  string
		output Coin
	}{
		{"0", Coin(0)},
		{"0.01", Coin(1)},
		{"0.10", Coin(10)},
		{"0.1", Coin(10)},
		{"1.00", Coin(100)},
		{"1.0", Coin(100)},
		{"10.00", Coin(1000)},
		{"10.0", Coin(1000)},
		{"10", Coin(1000)},
		{"12345", Coin(1234500)},
		{"99.98", Coin(9998)},
	}
	for _, c := range cases {
		actual, err := CoinFromString(c.input)
		if err != nil {
			t.Errorf("CoinFromString(%s) returned error '%s'\n", c.input, err)
		}

		if actual != c.output {
			t.Errorf("CoinFromString(%v) did not return expected value %v, received %v instead",
				c.input, c.output, actual)
		}
	}
}

func TestCoinToString(t *testing.T) {
	cases := []struct {
		input  Coin
		output string
	}{
		{Coin(0), "0.00"},
		{Coin(1), "0.01"},
		{Coin(10), "0.10"},
		{Coin(100), "1.00"},
		{Coin(1000), "10.00"},
		{Coin(9998), "99.98"},
		{Coin(1234500), "12345.00"},
	}
	for _, c := range cases {
		actual := c.input.ToString()
		if actual != c.output {
			t.Errorf("%v.ToString() did not return expected value %v, received %v instead",
				c.input, c.output, actual)
		}
	}
}

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
