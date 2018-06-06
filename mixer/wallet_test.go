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
	j := &Wallet{client, "Alice"}
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
	j := &Wallet{client, "Bob"}

	returnedTxns, err := j.GetTransactions(now)
	if err != nil {
		t.Errorf("Did not successfully fetch transactions. Saw error '%s' instead", err)
	}

	expected := txns[1]
	actual := returnedTxns[0]

	// have to do a manual deep-comparison because of the transaction.Amount values
	// this to me screens code smell and reimplies a refactoring is needed somewhere
	// probably a Coin type that can abstract away all this complexity
	expectedAmount, _ := j.JobcoinToInt(expected.Amount)
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
	mixer := NewMixer(batches)
	mixer.Run()
}
