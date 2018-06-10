package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/philangist/apollo/mixer"
)

type CLI struct{}

func (cli *CLI) Usage() {
	fmt.Println("Usage:")
	fmt.Println("   --amount AMOUNT --destination \"ADDRESS1 ADDRESS2 ...ADDRESSN\" - Send AMOUNT of Jobcoins to ADDRESSES that you own")
}

func (cli *CLI) Parse() (string, []mixer.Address) {
	amount := flag.String("amount", "", "amount of Jobcoin to tumble")
	destination := flag.String("destination", "", "amount of Jobcoin to tumble")

	flag.Parse()
	if len(*destination) == 0 {
		cli.Usage()
		os.Exit(1)
	}

	var addresses []mixer.Address
	for _, address := range strings.Split(*destination, " ") {
		addresses = append(addresses, mixer.Address(address))
	}
	return *amount, addresses
}

func main() {
	cli := &CLI{}
	amount, recipients := cli.Parse()
	source := mixer.NewWallet(mixer.CreateAddresses(1)[0])
	fmt.Printf("Send %v Jobcoins to tumbler address: %s\n", amount, source.Address)

	parsedAmount, err := mixer.CoinFromString(amount)
	if err != nil {
		log.Panic(fmt.Errorf("amount '%v' is not a valid numeric value", amount))
	}

	fee := int64(float64(parsedAmount) * float64(0.2))

	batch := mixer.NewBatch(
		parsedAmount, mixer.Coin(fee), source, recipients)
	mixer := mixer.NewMixer([]*mixer.Batch{batch})
	mixer.Run()
}
