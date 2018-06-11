[![Build Status](https://travis-ci.com/philangist/apollo.svg?branch=master)](https://travis-ci.com/philangist/apollo)  [![codecov](https://codecov.io/gh/philangist/apollo/branch/master/graph/badge.svg)](https://codecov.io/gh/philangist/apollo)

Apollo
==

Cryptocurrency tumbler for the future

![Apollo Mission](https://dprbcn.files.wordpress.com/2014/02/space_bit.gif)


__Running it locally__

Run from source

```bash
$ go get github.com/philangist/apollo
$ cd $GOPATH/src/github.com/philangist/apollo
$ go run main.go -amount=1 -timeout=120 -destination="Alice Bob Charles Daniel Elizabeth Francine George Harrris Ida"                           0 < 07:00:01
Send 1.00 Jobcoins to tumbler address: 1528717278-2229040884-0
Address is  Pool-2018-June-7-11
b.StartTime: 2018-06-11 07:41:18.631204126 -0400 EDT m=+0.003202583
Polling address: 1528717278-2229040884-0
New txn seen: &{2018-06-11 11:41:58.912 +0000 UTC Address-1 1528717278-2229040884-0 100}
Sending amount '1.00' to recipient 'Pool-2018-June-7-11'
Sending amount '0.18' to recipient 'Alice'
...
```

Run as binary

```bash
$ git clone github.com/philangist/apollo
$ cd apollo
$ ./build/apollo -amount=10 -timeout=30 -destination="Julio Keanna Leo"
Send 10.00 Jobcoins to tumbler address: 1528717636-2347662004-0
Address is  Pool-2018-June-7-11
b.StartTime: 2018-06-11 07:47:16.708110045 -0400 EDT m=+0.016500152
Polling address: 1528717636-2347662004-0
New txn seen: &{2018-06-11 11:47:27.821 +0000 UTC Address-1 1528717636-2347662004-0 1000}
Sending amount '10.00' to recipient 'Pool-2018-June-7-11'
...
```

Tests:
```bash
$ go test -v ./...
```

__Design & Rationale__  
Architecture:
- The core data structures in Apollo are `Address`, `Coin`, and `Transaction`. Both `Transaction` and `Coin` are used to read/write data representations accross application boundaries to the user and Jobcoin blockchain.

- The pooling logic is handled by `Batch` and `Mixer`. `Mixer` follows a `PoolStrategy` which is a function that returns a pool `Address`. Apollo's default pooling strategy is to generate a new central pool every hour.

- For polling of new transactions I chose I chose to just use the `FETCH_TXNS_URL` endpoint (http://jobcoin.gemini.com/victory/api/transactions) because it `Wallet.GetTransactions` to only use `Transaction`s for parsing reponses and I would've had to write a specialized container type for the ADDRESS INFO endpoint http://jobcoin.gemini.com/victory/api/addresses/{address}. This behavior is also more consistent with how polling a real blockchain would work.

- There's obviously a performance hit for making the same request every time `GetTransactions` is called and performing an O(n) scan for each `Transaction` that meets the filter criteria -- timestamp > cutoff and recipient == w.Address. The upside is that it simplified the development process and realistically this solution could easily scale to several tens-hundreds of thousands of transaction records being returned per call without any problems. It's not perfect, but it's a reasonable tradeoff.

- If performance started to be an issue some possible solutions are to:


1. Only request `Transaction`s past a certain time window. Real blockchain nodes typically support this by allowing peers to only ask for blocks above a certain height.
2. Create an in-memory cache of previously requested txns (since they're immutable) and append newly seen txns to the cache record for all relevant users. Cache entries could also have a simple bit representing their processed state so previously tumbled transactions can be safely evicted. An example cache structure:
```javascript
{
  "Alice-TXNS" : [(TXN_1, 1), (TXN_2, 1), .. (TXN_N, 0)],
  "Bob-TXNS" :   [(TXN_1, 1), (TXN_3, 1), (TXN_7, 1), .. (TXN_N + 1, 0)],
}
```
`Wallet`s would then read `Transaction`s from the cache, instead of making network requests

3. Use the ADDRESS INFO endpoint to read from the Jobcoin API on a per-`Wallet` basis

Language:
- I choose to use Go to implement the solution because of it's simplicity and batteries included standard library. The strong type system, built-in static analyzer (go vet) and testing framework also mean that as a developer it's easy to reason about the guarantees that Apollo provides. Lastly, the liberal use of interfaces within the standard library also provides a lot of control with regards to how Apollo's data is represented by the language (JSON serialization/deserialization for the `Coin` type is a great example of this)

- The language is very strict and explicit which I personally like. The downside is that it can make certain test patterns like mocking and monkeypatching that are straightforward in other languages cumbersome to implement. The `TestMixerRun` in `mixer/mixer_test.go` is a case where the needed functionality could've been tested more succintly and extensively in a dynamic language.

- Although I started off with a concurrency-heavy approach (exhibit A: https://github.com/philangist/apollo/commit/fb83b2682a98d105933dee10d20136cdcb5a2b22), my final solution was much simpler and didn't need to take heavy advantage of Go's runtime speed, or builtin concurrency and networking primitives. I lost some of the languages best features and also lost the rapid development speed and flexibility  dynamic languages provide. That being said, once I put in the time to write solid tests with good coverage I felt like I could trust that my code's invariants held a lot more than when I've written code with Python or Perl. I actually wrote some of the serialization/deserialization code on a plane flight to San Francisco without wi-fi and when I landed there was zero issue with running the changes against live data. Using Apollo with the provided endpoints did not result in any weird type errors I hadn't thought to account for, and I'm certain that would've happened with the other mentioned languages.
