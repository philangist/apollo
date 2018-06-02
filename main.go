package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"sync"
	"time"
)

var (
	TXNS_URL = "http://jobcoin.gemini.com/victory/api/transactions"
	pool *Wallet
)

type Address string

type Transaction struct {
	Timestamp time.Time `json:"timestamp"`
	Source    Address `json:"fromAddress"`
	Recipient Address `json:"toAddress"`
	Amount    string  `json:"amount"`
}

type Wallet struct {
	client *http.Client
	Address Address
}

func NewWallet(address string) *Wallet {
	return &Wallet{
		&http.Client{}, Address(address),
	}
}

func (w *Wallet) Send(recipient Address, amount int) error {
	if amount <= 0 {
		return fmt.Errorf("amount should be a positive integer value")
	}

	fmt.Printf("Sending amount '%d' to recipient '%s'\n", amount, recipient)
	return nil
}

func (w *Wallet) GetTransactions() ([]*Transaction, error) {
	var allTxs []*Transaction
	var filteredTxs []*Transaction

	b, err := w.JSONRequest(TXNS_URL)
	if err != nil {
		return allTxs, err
	}	

	json.Unmarshal(b, &allTxs)
	for _, tx := range allTxs {
		if tx.Recipient == w.Address{
			filteredTxs = append(filteredTxs, tx)	
		}
	}

	return filteredTxs, nil
}

func (w *Wallet) JSONRequest(url string) ([]byte, error) {
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
	Amount int
	Fee int
	Sources []Address
	Recipients []Address
	ready chan bool
	StartTime time.Time
}

func (b *Batch) Transfer () (err error) {
	pool.Send(pool.Address, b.Fee)
	portion := (b.Amount - b.Fee)/len(b.Recipients)

	for _, recipient := range b.Recipients {
		err = pool.Send(recipient, portion)
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

	// concurrent threads of execution which poll the blockchain for transactions
	// relevant to b.Sources
	for _, source := range b.Sources {
		go func(source Address){
			sum := 0
			w := &Wallet{&http.Client{}, source}
			txns, _ := w.GetTransactions()

			for _, txn := range txns {
				amount, _ := strconv.ParseInt(txn.Amount, 10, 32)
				sum += int(amount)
			}

			poll <- pollMessage{source, sum}
		}(source)
	}

	// serial consumer of poll that makes sure that:
	// 1. every address in b.Sources has had a transaction sent to it
	// 2. the Jobcoins for each address are then forwarded to the central pool
	for j := 0; j < len(b.Sources); j++ {
		message := <- poll
		w := Wallet{&http.Client{}, message.address}
		w.Send(pool.Address, message.amount)
	}
	b.ready <- true
}

func (b *Batch) Run (wg *sync.WaitGroup){
	go b.PollTransactions()
	for {
		select {
		case <- b.ready:
			b.Transfer()
			wg.Done()
		case <-time.After(5 * time.Second):
			// return errors.New("Timeout hit")
			wg.Done()
		}
	}
}

type Mixer struct {
	Batches []*Batch
	WaitGroup *sync.WaitGroup
}

func (m *Mixer) Run(){
	wg := m.WaitGroup
	for _, b := range m.Batches {
		wg.Add(1)
		go b.Run(wg)
	}
	wg.Wait()
}

func main(){
	w := NewWallet("Alice")
	w.GetTransactions()
}

func init(){
	pool = NewWallet("Pool")
}
