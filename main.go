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
	fmt.Println("   --amount AMOUNT --destination \"ADDRESS1 ADDRESS2 ...ADDRESSN\" - Send AMOUNT of Jobcoins to ADDRESSES that you own")
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
	source := wallet.CreateAddresses(1)[0]
	fmt.Printf("Send %d Jobcoins to tumbler address: %s\n", amount, source)

	amount = amount * 100
	fee := int(float32(amount) * float32(0.2))

	batch := wallet.NewBatch(amount, fee, source, recipients)
	mixer := wallet.NewMixer([]*wallet.Batch{batch})
	mixer.Run()
}
