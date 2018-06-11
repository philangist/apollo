package mixer

import (
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"
)

type DelayGenerator func(int) int

func RandomDelay(maxDelay int) int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(maxDelay)
}

type Batch struct {
	Amount         Coin
	Fee            Coin
	Source         *Wallet
	Recipients     []Address
	StartTime      time.Time
	PollInterval   time.Duration
	Timeout        time.Duration
	DelayGenerator DelayGenerator
}

func NewBatch(amount, fee Coin, source *Wallet, recipients []Address, timeout int) *Batch {
	return &Batch{
		amount,
		fee,
		source,
		recipients,
		time.Now(),
		time.Duration(1) * time.Second,
		time.Duration(timeout) * time.Second,
		RandomDelay,
	}
}

func (b *Batch) GeneratePayouts(amount Coin, totalRecipients int) []Coin {
	rand.Seed(time.Now().UnixNano())
	payouts := []Coin{}

	for i := 0; i < totalRecipients; i++ {
		if (i + 1) == totalRecipients {
			payouts = append(payouts, amount)
		} else {
			// successively take a random integer payout between (1, n/2 + 1) from amount
			// and update amount with the new value
			upperBound := int(amount / 2)
			if upperBound == 0 {
				//if upperBound == 0 that imples amount was 1, so we can just add that
				// payout value and early exit. Note that this implies that
				// not every recipient account necessarily receives a payout
				payouts = append(payouts, amount)
				break
			}
			payout := Coin(rand.Intn(upperBound) + 1)
			payouts = append(payouts, payout)
			amount -= payout
		}
	}

	return payouts
}

func (b *Batch) Tumble(pool *Wallet) (err error) {
	amount := b.Amount - b.Fee //keep b.Fee amount in the pool
	totalRecipients := len(b.Recipients)

	payouts := b.GeneratePayouts(amount, totalRecipients)

	for i, payout := range payouts {
		delay := time.Duration(b.DelayGenerator(10))
		time.Sleep(delay * time.Second)

		err = pool.SendTransaction(b.Recipients[i], payout)
		if err != nil {
			return err
		}
	}

	return err
}

func (b *Batch) PollTransactions(pool *Wallet) {
	fmt.Printf("b.StartTime: %s\nPolling address: %s\n", b.StartTime, b.Source.Address)

	sum := Coin(0)
	cutoff := b.StartTime            // look for new transactions after cutoff
	timeout := cutoff.Add(b.Timeout) // exit if no new transactions are seen by timeout

	for {
		if timeout.Before(time.Now()) {
			return
		}

		txns, err := b.Source.GetTransactions(cutoff)
		if err != nil {
			log.Panic(err)
		}

		for _, txn := range txns {
			b.Source.SendTransaction(pool.Address, txn.Amount)
			sum += txn.Amount
		}

		if sum >= b.Amount {
			b.Tumble(pool)
			break
		}

		cutoff = time.Now()
		time.Sleep(b.PollInterval)
	}
}

type PoolStrategy func() *Wallet

// generate a new Pool address every hour
func HourlyPool() *Wallet {
	now := time.Now()
	address := fmt.Sprintf(
		"Pool-%v-%v-%v-%v",
		now.Year(), now.Month(), now.Hour(), now.Day(),
	)

	fmt.Println("Address is ", address)
	return NewWallet(Address(address))
}

type Mixer struct {
	Pool          PoolStrategy
	Batches       []*Batch
	WaitGroup     *sync.WaitGroup
}

func NewMixer(batches []*Batch) *Mixer {
	return &Mixer{
		HourlyPool,
		batches,
		&sync.WaitGroup{},
	}
}

func (m *Mixer) Run() {
	wg := m.WaitGroup
	pool := m.Pool()

	for _, b := range m.Batches {
		wg.Add(1)
		go func(b *Batch) {
			b.PollTransactions(pool)
			wg.Done()
		}(b)
	}
	wg.Wait()
}
