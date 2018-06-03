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
	"time"
)

var (
	FETCH_TXNS_URL = "http://jobcoin.gemini.com/victory/api/transactions"
	SEND_TXN_URL = "http://jobcoin.gemini.com/victory/send"
)

type Address string

func CreateAddresses(total int) (addresses []Address) {
	// hash this shieeeeet
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

func (w *Wallet) SendTransaction(recipient Address, amount int) error {
	if amount <= 0 {
		return fmt.Errorf("amount should be a positive integer value")
	}

	tx := Transaction{time.Now(), w.Address, recipient, fmt.Sprintf("%d", amount)}
	serializedTx, err := json.Marshal(tx)
	if err != nil {
		log.Panic(err)
	}

	txBuffer := bytes.NewBuffer(serializedTx)

	err = w.client.JSONPostRequest(SEND_TXN_URL, txBuffer)
	if err != nil {
		log.Panic(err)
	}
	
	fmt.Printf("Sending amount '%d' to recipient '%s'\n", amount, recipient)
	return nil
}

func (w *Wallet) GetTransactions() ([]*Transaction, error) {
	var allTxs []*Transaction
	var filteredTxs []*Transaction

	b, err := w.client.JSONGetRequest(FETCH_TXNS_URL)
	if err != nil {
		return allTxs, err
	}

	json.Unmarshal(b, &allTxs)


	for _, tx := range allTxs {
		if tx.Recipient == w.Address {
			filteredTxs = append(filteredTxs, tx)
		}
	}

	return filteredTxs, nil
}

type ApiClient struct {
	*http.Client
}

func NewApiClient() *ApiClient {
	return &ApiClient{&http.Client{}}
}

func (j *ApiClient) JSONPostRequest(url string, payload *bytes.Buffer) (error) {
	request, err := http.NewRequest(
		"POST",
		url,
		payload,
	)
	if err != nil {
		return err
	}

	request.Header.Set("Content-Type", "application/json")
	response, err := j.Do(request)
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

func (j *ApiClient) JSONGetRequest(url string) ([]byte, error) {
	var byteStream []byte

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return byteStream, err
	}

	response, err := j.Do(request)
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
