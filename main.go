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
	fmt.Println("   --amount AMOUNT --destination \"ADDRESS1 ADDRESS2 ...ADDRESSN\" --timeout TIMEOUT - Send AMOUNT of Jobcoins to ADDRESSES that you own")
}

func (cli *CLI) Parse() (string, int, []mixer.Address) {
	amount := flag.String("amount", "", "amount of Jobcoin to tumble")
	timeout := flag.Int("timeout", 60, "number of seconds to watch for inbound transfer to tumbler address")
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
	return *amount, *timeout, addresses
}

func main() {
	cli := &CLI{}
	amount, timeout, recipients := cli.Parse()
	parsedAmount, err := mixer.CoinFromString(amount)
	if err != nil {
		log.Panic(fmt.Errorf("amount '%v' is not a valid numeric value", amount))
	}
	fee := mixer.Coin(int64(float64(parsedAmount) * float64(0.2)))

	source := mixer.NewWallet(mixer.CreateAddresses(1)[0])
	fmt.Printf("Send %v Jobcoins to tumbler address: %s\n", amount, source.Address)

	batch := mixer.NewBatch(parsedAmount, fee, source, recipients, timeout)
	mixer := mixer.NewMixer([]*mixer.Batch{batch})
	mixer.Run()
}
