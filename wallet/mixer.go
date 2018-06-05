package wallet

import (
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"time"
)

var pool = NewWallet("Pool")

// deal in cents
type Batch struct {
	Amount     int
	Fee        int
	Source     Address
	Recipients []Address
	StartTime  time.Time
}

// add timeout
func NewBatch(amount, fee int, source Address, recipients []Address) *Batch {
	return &Batch{
		amount,
		fee,
		source,
		recipients,
		time.Now(),
	}
}

func (b *Batch) GeneratePayouts(amount, totalRecipients int) []int {
	rand.Seed(time.Now().UnixNano())
	payouts := []int{}

	for index := 0; index < totalRecipients; index++ {
		if (index + 1) == totalRecipients {
			payouts = append(payouts, amount)
		} else {
			// comment explaining this tomfoolery
			upperBound := amount / 2
			if  upperBound == 0 {
				payouts = append(payouts, amount)
				break
			}
			portion := rand.Intn(upperBound) + 1
			payouts = append(payouts, portion)
			amount -= portion
		}
	}

	return payouts
}

func (b *Batch) Tumble() (err error) {
	amount := b.Amount - b.Fee //pay b.Fee amount to the pool
	totalRecipients := len(b.Recipients)

	payouts := b.GeneratePayouts(amount, totalRecipients)

	rand.Seed(time.Now().UnixNano())
	for index, payout := range payouts {
		sleep := time.Duration(rand.Intn(10)) * time.Second
		time.Sleep(sleep)

		err = pool.SendTransaction(b.Recipients[index], payout)
		if err != nil {
			return err
		}
	}

	return err
}

func (b *Batch) PollTransactions() {
	fmt.Printf("b.StartTime: %s\nPolling address: %s\n", b.StartTime, b.Source)

	w := &Wallet{NewApiClient(), b.Source}
	sum := 0
	seen := false
	timeout := time.Now().Add(time.Duration(30) * time.Second)

	for {
		if (timeout.Before(time.Now())) {
			return
		}

		txns, _ := w.GetTransactions(b.StartTime)
		for _, txn := range txns {
			fmt.Printf("new txn.Timestamp: %s\n", txn.Timestamp)
			amount, _ := strconv.ParseInt(txn.Amount, 10, 32)
			w.SendTransaction(pool.Address, int(amount) * 100)
			sum += int(amount) * 100
			fmt.Printf("amount is %d\n", int(amount) * 100)
			seen = true
		}
		// if sum < b.Amount {
		if seen == true {
			fmt.Printf("sum is %d\n", sum)
			b.Tumble()
			return
		}else {
			time.Sleep(5 * time.Second)
			continue
		}
	}
}

func (b *Batch) Run(wg *sync.WaitGroup) {
	b.PollTransactions()
	wg.Done()
}

type Mixer struct {
	Batches   []*Batch
	WaitGroup *sync.WaitGroup
}

func NewMixer(batches []*Batch) *Mixer {
	return &Mixer{
		batches,
		&sync.WaitGroup{},
	}
}

func (m *Mixer) Run() {
	wg := m.WaitGroup
	for _, b := range m.Batches {
		wg.Add(1)
		go b.Run(wg)
	}
	wg.Wait()
}
