package main

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
	"sync"
	"time"
)

var (
	FETCH_TXNS_URL = "http://jobcoin.gemini.com/victory/api/transactions"
	SEND_TXN_URL = "http://jobcoin.gemini.com/victory/send"
	pool     *Wallet
)

type Address string

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

type Batch struct {
	Amount     int
	Fee        int
	Sources    []Address
	Recipients []Address
	ready      chan bool
	StartTime  time.Time
}

func (b *Batch) Tumble() (err error) {
	pool.SendTransaction(pool.Address, b.Fee)
	portion := (b.Amount - b.Fee) / len(b.Recipients)

	for _, recipient := range b.Recipients {
		err = pool.SendTransaction(recipient, portion)
		if err != nil {
			return err
		}
	}

	return err
}

func (b *Batch) PollTransactions() {
	type pollMessage struct {
		address Address
		amount  int
	}

	poll := make(chan pollMessage)

	pollAddress := func(source Address) {
		fmt.Printf("b.StartTime is %s\n", b.StartTime)

		for {
			sum := 0
			w := &Wallet{NewApiClient(), source}
			txns, _ := w.GetTransactions()

			for _, txn := range txns {
				if txn.Timestamp.Before(b.StartTime){
					continue
				}
				fmt.Printf("latest txn.Timestamp is %s\n", txn.Timestamp)
				amount, _ := strconv.ParseInt(txn.Amount, 10, 32)
				sum += int(amount)
			}

			fmt.Printf("Sum is %d\n", sum)

			if sum > 0 {
				poll <- pollMessage{source, sum}
				break
			}
			time.Sleep(5 * time.Second)
		}
	}

	// concurrent threads of execution which poll the blockchain for transactions
	// relevant to b.Sources
	for _, source := range b.Sources {
		go pollAddress(source)
	}

	// serial consumer of poll that makes sure that:
	// 1. every address in b.Sources has had a transaction sent to it
	// 2. the Jobcoins for each address are then forwarded to the central pool
	for j := 0; j < len(b.Sources); j++ {
		message := <-poll
		w := Wallet{NewApiClient(), message.address}
		w.SendTransaction(pool.Address, message.amount)
	}
	b.ready <- true
}

func (b *Batch) Run(wg *sync.WaitGroup) {
	go b.PollTransactions()
	for {
		select {
		case <-b.ready:
			b.Tumble()
			wg.Done()
		case <-time.After(15 * time.Second):
			wg.Done()
		}
	}
}

type Mixer struct {
	Batches   []*Batch
	WaitGroup *sync.WaitGroup
}

func (m *Mixer) CreateAddresses(total int) (addresses []Address) {
	// hash this shieeeeet
	nonce := rand.Intn(4294967296)
	prefix := fmt.Sprintf("Address: %d-%d", time.Now().Unix(), nonce)

	for i:=0; i < total; i++ {
		addresses = append(
			addresses,
			Address(fmt.Sprintf("%s-%d", prefix, i)),
		)
	}

	return addresses
}

func (m *Mixer) Run() {
	wg := m.WaitGroup
	for _, b := range m.Batches {
		wg.Add(1)
		go b.Run(wg)
	}
	wg.Wait()
}

func main() {
	// w := NewWallet("to")
	// w.SendTransaction(Address("Alice"), 5)
	// fmt.Printf("%s\n", now)
	m := &Mixer{}
	fmt.Printf("%q", m.CreateAddresses(5))
}

func init() {
	pool = NewWallet("Pool")
}
