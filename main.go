package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/philangist/apollo/wallet"
)

type CLI struct {}

func (cli *CLI) Usage() {
	fmt.Println("Usage:")
	fmt.Println("   --amount AMOUNT --destination ADDRESSES - Send AMOUNT of Jobcoins to ADDRESSES that you own")
}

func (cli *CLI) Parse() (int, []wallet.Address) {
	amount := flag.Int("amount", 0, "amount of Jobcoin to tumble")
	destination := flag.String("destination", "", "amount of Jobcoin to tumble")

	flag.Parse()
	if len(*destination) == 0 {
		cli.Usage()
		os.Exit(1)
	}

	var addresses []wallet.Address
	for _, address := range strings.Split(*destination, " ") {
		addresses = append(addresses, wallet.Address(address))
	}
	return *amount, addresses
}

func main() {
	cli := &CLI{}
	amount, recipients := cli.Parse()
	sources := wallet.CreateAddresses(1)

	fmt.Printf("Send %d Jobcoins to tumbler address: %s\n", amount, sources[0])

	fee := 2
	batch := wallet.NewBatch(amount, fee, sources, recipients)

	m := wallet.NewMixer([]*wallet.Batch{batch})
	m.Run()
}
