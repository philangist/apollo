package main

import (
	"fmt"

	"github.com/philangist/apollo/wallet"
)

func main() {
	// w := NewWallet("to")
	// w.SendTransaction(Address("Alice"), 5)
	// fmt.Printf("%s\n", now)

	amount := 10
	fee := 2
	// sources := wallet.CreateAddresses(1)
	sources := []wallet.Address{
		wallet.Address("1528021144-134020434-0")}
	fmt.Printf("sources: %v", sources)
	recipients := []wallet.Address{
		wallet.Address("Alice")}
	batch := wallet.NewBatch(amount, fee, sources, recipients)
	m := wallet.NewMixer(
		[]*wallet.Batch{batch})
	m.Run()
}
