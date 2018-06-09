package mixer

import (
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"
)

var pool = NewWallet("Pool")

type DelayGenerator func(int) int

func RandomDelay(maxDelay int) int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(maxDelay)
}

// deal in cents
type Batch struct {
	Amount         Coin
	Fee            Coin
	Source         *Wallet
	Recipients     []Address
	StartTime      time.Time // probably rename StartTime
	PollInterval   time.Duration
	DelayGenerator DelayGenerator // rename this also
}

// add timeout
func NewBatch(amount, fee Coin, source *Wallet, recipients []Address) *Batch {

	return &Batch{
		amount,
		fee,
		source,
		recipients,
		time.Now(),
		time.Duration(1),
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

func (b *Batch) Tumble() (err error) {
	amount := b.Amount - b.Fee //pay b.Fee amount to the pool
	totalRecipients := len(b.Recipients)

	payouts := b.GeneratePayouts(amount, totalRecipients)

	for i, payout := range payouts {
		fmt.Printf("recipient: %v, payouts: %d\n", b.Recipients[i], payout)
	}

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

func (b *Batch) PollTransactions() {
	fmt.Printf("b.StartTime: %s\nPolling address: %s\n", b.StartTime, b.Source)
	fmt.Printf("b.Amount is %v\n", b.Amount)

	sum := Coin(0)
	cutoff := b.StartTime                                 // look for new transactions after cutoff
	timeout := cutoff.Add(time.Duration(1) * time.Second) // exit if no new transactions are seen by timeout

	for {
		if timeout.Before(time.Now()) {
			return
		}

		txns, err := b.Source.GetTransactions(cutoff)
		if err != nil {
			log.Panic(err)
		}

		for _, txn := range txns {
			fmt.Printf("new txn.Timestamp: %s\n", txn.Timestamp)
			amount, _ := CoinFromString(txn.Amount) // validity of amount is guaranteed since b.Source.GetTransactions() did not panic
			b.Source.SendTransaction(pool.Address, amount)
			sum += amount
			fmt.Printf("amount is %d\n", amount)
		}
		if sum >= b.Amount {
			fmt.Printf("sum is %d\n", sum)
			b.Tumble()
			break
		} else {
			cutoff = time.Now()
			time.Sleep(b.PollInterval * time.Second)
			continue
		}
	}
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
		go func(b *Batch) {
			b.PollTransactions()
			wg.Done()
		}(b)
	}
	wg.Wait()
}
