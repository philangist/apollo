package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
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
	client  *http.Client
	Address Address
}

func NewWallet(address string) *Wallet {
	return &Wallet{
		&http.Client{}, Address(address),
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

	err = w.JSONPostRequest(SEND_TXN_URL, txBuffer)
	if err != nil {
		log.Panic(err)
	}
	
	fmt.Printf("Sending amount '%d' to recipient '%s'\n", amount, recipient)
	return nil
}

func (w *Wallet) GetTransactions() ([]*Transaction, error) {
	var allTxs []*Transaction
	var filteredTxs []*Transaction

	b, err := w.JSONGetRequest(FETCH_TXNS_URL)
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

func (w *Wallet) JSONPostRequest(url string, payload *bytes.Buffer) (error) {
	request, err := http.NewRequest(
		"POST",
		url,
		payload,
	)

	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := w.client.Do(request)
	fmt.Printf("response was %v\n", response)
	fmt.Printf("error was %s\n", err)
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

func (w *Wallet) JSONGetRequest(url string) ([]byte, error) {
	var byteStream []byte

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return byteStream, err
	}

	response, err := w.client.Do(request)
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

func (b *Batch) Transfer() (err error) {
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
			w := &Wallet{&http.Client{}, source}
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
		w := Wallet{&http.Client{}, message.address}
		w.SendTransaction(pool.Address, message.amount)
	}
	b.ready <- true
}

func (b *Batch) Run(wg *sync.WaitGroup) {
	go b.PollTransactions()
	for {
		select {
		case <-b.ready:
			b.Transfer()
			wg.Done()
		case <-time.After(15 * time.Second):
			// return errors.New("Timeout hit")
			wg.Done()
		}
	}
}

type Mixer struct {
	Batches   []*Batch
	WaitGroup *sync.WaitGroup
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
	w := NewWallet("to")
	w.SendTransaction(Address("Alice"), 5)
}

func init() {
	pool = NewWallet("Pool")
}
