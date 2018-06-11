package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/philangist/apollo/mixer"
)

type CLI struct{}

func (cli *CLI) Usage() {
	fmt.Println("Usage:")
	fmt.Println("   --amount AMOUNT --destination \"ADDRESS1 ADDRESS2 ...ADDRESSN\" --timeout TIMEOUT - Send AMOUNT of Jobcoins to ADDRESSES that you own")
}

func (cli *CLI) Parse() (mixer.Coin, int, []mixer.Address) {
	amount := flag.String("amount", "", "amount of Jobcoin to tumble")
	timeout := flag.Int("timeout", 60, "number of seconds to watch for inbound transfer to tumbler address")
	destination := flag.String("destination", "", "amount of Jobcoin to tumble")

	flag.Parse()

	parsedAmount, err := mixer.CoinFromString(*amount)
	if err != nil {
		fmt.Println(fmt.Errorf("Amount '%v' is not a valid numeric value", amount))
		cli.Usage()
		os.Exit(1)
	}

	if parsedAmount < 0 {
		fmt.Println("Amount must be a non-negative value")
		cli.Usage()
		os.Exit(1)
	}

	if *timeout < 0 {
		fmt.Println("Timeout must be a non-negative value")
		cli.Usage()
		os.Exit(1)
	}

	if len(*destination) == 0 {
		cli.Usage()
		os.Exit(1)
	}

	var addresses []mixer.Address
	for _, address := range strings.Split(*destination, " ") {
		if address == "" {
			continue
		}
		addresses = append(addresses, mixer.Address(address))
	}
	if len(addresses) == 0 {
		fmt.Println("No valid addresses seen. Addresses must be non-empty strings")
		cli.Usage()
		os.Exit(1)
	}

	return parsedAmount, *timeout, addresses
}

func main() {
	cli := &CLI{}
	amount, timeout, recipients := cli.Parse()

	fee := mixer.Coin(int64(float64(amount) * float64(0.2)))
	source := mixer.NewWallet(mixer.NewAddresses(1)[0])
	fmt.Printf("Send %v Jobcoins to tumbler address: %s\n", amount.ToString(), source.Address)

	batch := mixer.NewBatch(amount, fee, source, recipients, timeout)
	mixer := mixer.NewMixer([]*mixer.Batch{batch})
	mixer.Run()
}
