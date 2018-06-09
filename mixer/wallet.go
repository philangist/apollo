package mixer

import (
	"bytes"
	// "crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var (
	FETCH_TXNS_URL = "http://jobcoin.gemini.com/victory/api/transactions"
	SEND_TXN_URL   = "http://jobcoin.gemini.com/victory/send"
)

type Address string

func CreateAddresses(total int) (addresses []Address) {
	rand.Seed(time.Now().UnixNano())
	prefix := fmt.Sprintf("%d-%d", time.Now().Unix(), rand.Intn(4294967296))

	for i := 0; i < total; i++ {
		addresses = append(
			addresses,
			Address(
				HashString(fmt.Sprintf("%s-%d", prefix, i)),
			),
		)
	}

	return addresses
}

type Coin int64 // deals with jobcoin values in cents

func CoinFromString(amount string) (Coin, error) {
	index := strings.Index(amount, ".")

	if index >= 0 {
		padding := (len(amount) - (index + 1))
		if padding < 2 {
			for i := 0; i < padding; i++ {
				amount = fmt.Sprintf("%s0", amount)
			}
		}
		amount = strings.Replace(amount, ".", "", 1)
	} else {
		amount = fmt.Sprintf("%s00", amount)
	}

	val, err := strconv.ParseInt(amount, 10, 64)
	if err != nil {
		return 0, err
	}

	return Coin(val), nil
}

func CoinFromInt(amount int) Coin {
	return Coin(amount * 100)
}

func (c Coin) ToString() string {
	cents := fmt.Sprintf("%v", c)
	size := len(cents)
	if size <= 2 {
		return fmt.Sprintf("%v", float64(c.ToInt64())/float64(100))
	}
	return fmt.Sprintf("%v.%v", cents[:size-2], cents[size-2:])
}

func (c Coin) ToInt64() int64 {
	return int64(c)
}

type Client interface {
	JSONGetRequest(url string) ([]byte, error)
	JSONPostRequest(url string, payload *bytes.Buffer) error
}

type ApiClient struct {
	*http.Client
}

func NewApiClient() *ApiClient {
	return &ApiClient{&http.Client{}}
}

func (a *ApiClient) JSONGetRequest(url string) ([]byte, error) {
	var byteStream []byte

	request, err := http.NewRequest(
		"GET",
		url,
		nil,
	)
	if err != nil {
		return byteStream, err
	}

	response, err := a.Do(request)
	if err != nil {
		return byteStream, err
	}

	if response.StatusCode != http.StatusOK {
		return byteStream, fmt.Errorf(
			"Url: '%s' returned unexpected status code %d", url, response.StatusCode)
	}
	reader := response.Body
	defer reader.Close()

	return ioutil.ReadAll(reader)
}

func (a *ApiClient) JSONPostRequest(url string, payload *bytes.Buffer) error {
	request, err := http.NewRequest(
		"POST",
		url,
		payload,
	)
	if err != nil {
		return err
	}

	request.Header.Set("Content-Type", "application/json")
	response, err := a.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf(
			"POST json request to url '%s' returned unexpected status code %d", url, response.StatusCode)
	}

	return nil
}

type Transaction struct {
	Timestamp time.Time `json:"timestamp"`
	Source    Address   `json:"fromAddress"`
	Recipient Address   `json:"toAddress"`
	Amount    string    `json:"amount"`
}

type Wallet struct {
	client  Client
	Address Address
}

func NewWallet(address Address) *Wallet {
	return &Wallet{
		NewApiClient(), address,
	}
}

func IntToJobcoin(amount int) string {
	cents := fmt.Sprintf("%v", amount)
	size := len(cents)
	if size <= 2 {
		return fmt.Sprintf("%v", float32(amount)/float32(100))
	}
	return fmt.Sprintf("%v.%v", cents[:size-2], cents[size-2:])
}

func JobcoinToInt(jobcoin string) (int, error) {
	fmt.Println(jobcoin)
	cents := strings.Replace(jobcoin, ".", "", 1)
	value, err := strconv.ParseInt(cents, 10, 32)
	if err != nil {
		return int(value), err
	}
	// make sure values like '0.1' are returned as 10, not 1
	if (len(cents) == 2) && (string(cents[0]) == "0") {
		return int(value) * 10, err
	}

	return int(value), err
}

func (w *Wallet) SendTransaction(recipient Address, amount Coin) error {
	if amount <= 0 {
		return fmt.Errorf("amount should be a positive integer value")
	}

	fmt.Printf("Sending amount '%s' to recipient '%s'\n", amount.ToString(), recipient)
	if 1 == 1 {
		return nil
	}

	txn := Transaction{time.Now(), w.Address, recipient, amount.ToString()}
	serializedTxn, err := json.Marshal(txn)
	if err != nil {
		log.Panic(err)
	}

	txnBuffer := bytes.NewBuffer(serializedTxn)
	err = w.client.JSONPostRequest(SEND_TXN_URL, txnBuffer)
	if err != nil {
		log.Panic(err)
	}

	return nil
}

func (w *Wallet) GetTransactions(cutoff time.Time) ([]*Transaction, error) {
	var allTxns []*Transaction
	var filteredTxns []*Transaction

	b, err := w.client.JSONGetRequest(FETCH_TXNS_URL)
	if err != nil {
		return allTxns, err
	}

	err = json.Unmarshal(b, &allTxns)
	if err != nil {
		return allTxns, err
	}

	for _, txn := range allTxns {
		if (txn.Recipient == w.Address) && txn.Timestamp.After(cutoff) {
			fmt.Printf("New txn seen: %v\n", txn)
			amount, err := CoinFromString(txn.Amount)
			if err != nil {
				return filteredTxns, err
			}

			txn.Amount = fmt.Sprintf("%d", amount)
			filteredTxns = append(filteredTxns, txn)
		}
	}

	return filteredTxns, nil
}
