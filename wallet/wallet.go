package wallet

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
	"time"
)

var (
	FETCH_TXNS_URL = "http://jobcoin.gemini.com/victory/api/transactions"
	SEND_TXN_URL = "http://jobcoin.gemini.com/victory/send"
)

type Address string

func (a Address) HashString(input string) string {
	return ""
}

func CreateAddresses(total int) (addresses []Address) {
	// hash this shieeeeet
	rand.Seed(time.Now().UnixNano())
	nonce := rand.Intn(4294967296)
	prefix := fmt.Sprintf("%d-%d", time.Now().Unix(), nonce)

	for i:=0; i < total; i++ {
		addresses = append(
			addresses,
			Address(fmt.Sprintf("%s-%d", prefix, i)),
		)
	}

	return addresses
}

// func JobcoinToInt
// func IntToJobcoin

type ApiClient struct {
	*http.Client
}

func NewApiClient() *ApiClient {
	return &ApiClient{&http.Client{}}
}


func (a *ApiClient) JSONGetRequest(url string) ([]byte, error) {
	var byteStream []byte

	request, err := http.NewRequest("GET", url, nil)
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

func (a *ApiClient) JSONPostRequest(url string, payload *bytes.Buffer) (error) {
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
	client  *ApiClient
	Address Address
}

func NewWallet(address string) *Wallet {
	return &Wallet{
		NewApiClient(), Address(address),
	}
}

func (w *Wallet) convertAmount(amount int) string {
	fmt.Printf("ConvertAmount(%d)\n", amount)
	cents := fmt.Sprintf("%v", amount)
	size := len(cents)
	if size <= 2 {
		return fmt.Sprintf("%v", float32(amount)/float32(100))
	}
	return fmt.Sprintf("%v.%v", cents[:size-2], cents[size-2:])
}

func (w *Wallet) SendTransaction(recipient Address, amount int) error {
	if amount <= 0 {
		return fmt.Errorf("amount should be a positive integer value")
	}

	convertedAmount := w.convertAmount(amount)
	fmt.Printf("Sending amount '%s' to recipient '%s'\n", convertedAmount, recipient)

	tx := Transaction{time.Now(), w.Address, recipient, convertedAmount}
	serializedTx, err := json.Marshal(tx)
	if err != nil {
		log.Panic(err)
	}

	txBuffer := bytes.NewBuffer(serializedTx)
	err = w.client.JSONPostRequest(SEND_TXN_URL, txBuffer)
	if err != nil {
		log.Panic(err)
	}

	return nil
}

func (w *Wallet) GetTransactions(cutoff time.Time) ([]*Transaction, error) {
	var allTxs []*Transaction
	var filteredTxs []*Transaction

	b, err := w.client.JSONGetRequest(FETCH_TXNS_URL)
	if err != nil {
		return allTxs, err
	}

	json.Unmarshal(b, &allTxs)

	for _, tx := range allTxs {
		if ((tx.Recipient == w.Address) && tx.Timestamp.After(cutoff)){
			fmt.Printf("New tx seen: %v\n", tx)
			amount, _ := strconv.ParseInt(tx.Amount, 10, 32)
			tx.Amount = fmt.Sprintf("%d", amount)
			filteredTxs = append(filteredTxs, tx)
		}
	}

	return filteredTxs, nil
}
