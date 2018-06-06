package mixer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

const (
	TIMEOUT          = time.Second * 3
	HTTP_OK          = http.StatusOK
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

type testClient struct {
	GetResponse  func(url string) ([]byte, error)
	PostResponse func(url string, payload *bytes.Buffer) error
}

func (t *testClient) JSONGetRequest(url string) ([]byte, error) {
	return t.GetResponse(url) // []byte(""), nil
}

func (t *testClient) JSONPostRequest(url string, payload *bytes.Buffer) error {
	return t.PostResponse(url, payload) // nil
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
		amount    int
		valid     bool
	}{
		{b, 100, true},
		{b, 0, false},
		{b, -100, false},
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

	txns := []*Transaction{
		&Transaction{
			past,
			"Alice",
			"Bob",
			"10.00",
		},
		&Transaction{
			future,
			"Alice",
			"Bob",
			"10.00",
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

	// have to do a manual deep-comparison because of the transaction.Amount values
	// this to me screens code smell and reimplies a refactoring is needed somewhere
	// probably a Coin type that can abstract away all this complexity
	expectedAmount, _ := JobcoinToInt(expected.Amount)
	actualAmount := actual.Amount
	if !((fmt.Sprintf("%v", expectedAmount) == actualAmount) &&
		(expected.Timestamp.Equal(actual.Timestamp)) &&
		(expected.Source == actual.Source) &&
		(expected.Recipient == actual.Recipient)) {
		t.Errorf("Returned transactions %q did not match expected transactions %q", returnedTxns, txns)
	}

	fmt.Println(returnedTxns)
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

func TestIntToJobcoin(t *testing.T){
	cases := []struct{
		input int
		output string
	}{
		{0, "0"},
		{1, "0.01"},
		{10, "0.1"},
		{100, "1.00"},
		{1000, "10.00"},
	}
	for _, c := range cases {
		actual := IntToJobcoin(c.input)
		if actual != c.output {
			t.Errorf("IntToJobcoin(%v) did not return expected value %v, received %v instead",
				c.input, c.output, actual)
		}
	}
}

func TestJobcoinToInt(t *testing.T){
	cases := []struct{
		input string
		output int
	}{
		{"0", 0},
		{"0.01", 1},
		{"0.10", 10},
		{"0.1", 10},
		{"1.00", 100},
		// {"1.0", 100},
		{"10.00", 1000},
		// {"10.0", 1000},
	}
	for _, c := range cases {
		actual, _ := JobcoinToInt(c.input)
		if actual != c.output {
			t.Errorf("JobcoinToInt(%v) did not return expected value %v, received %v instead",
				c.input, c.output, actual)
		}
	}
}


func TestBatchGeneratePayouts(t *testing.T) {
	fmt.Println("Running TestBatchGeneratePayouts...")

	amount := 120
	fee := 20

	source := NewWallet(Address("Address-1"))
	recipients := []Address{
		Address("Address-1"), Address("Address-2"),
	}

	batch := NewBatch(amount, fee, source, recipients)

	expected := amount - fee
	actual := 0
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

func NoDelay(maxDelay int) int {
	return 0
}

func TestBatchTumble(t *testing.T) {
	fmt.Println("Running TestBatchTumble...")

	batch := NewBatch(
		120,
		20,
		NewWallet(Address("Address-1")),
		[]Address{
			Address("Address-1"), Address("Address-2"), Address("Address-3"), Address("Address-4"), Address("Address-5"),
		},
	)
	batch.DelayGenerator = NoDelay
	err := batch.Tumble()
	if err != nil {
		t.Errorf("Expected Batch.Execute() to run successfully, received error '%s'", err)
	}
}

func TestBatchPollTransactions(t *testing.T) {
	fmt.Println("Running TestBatchPollTransactions...")


	future := time.Now().Add(time.Duration(1000) * time.Second)
	txns := []*Transaction{
		&Transaction{
			future,
			"Alice",
			"Bob",
			"10.00",
		},
	}

	client := &testClient{
		GetResponse: func(url string) ([]byte, error) {return json.Marshal(txns)},
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
	)
	batch.DelayGenerator = NoDelay

	batch.PollTransactions() // use recover/panic behavior here
}

func TestMixerRun(t *testing.T) {
	fmt.Println("Running TestMixerRun...")

	batch := NewBatch(
		10,
		2,
		NewWallet(Address("Alice")),
		[]Address{
			Address("Address-1"), Address("Address-2"),
		},
	)
	batch.DelayGenerator = NoDelay

	batches := []*Batch{batch} //, batch, batch}
	mixer := NewMixer(batches)
	mixer.Run()
}
