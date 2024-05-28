# Solana Go Web3 Library

[![Go Reference](https://pkg.go.dev/badge/github.com/donutnomad/solana-web3.svg)](https://pkg.go.dev/github.com/donutnomad/solana-web3)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

This is a Go library for interacting with the Solana blockchain, inspired by the official [@solana/web3.js](https://github.com/solana-labs/solana-web3.js) JavaScript library. It provides a simple and easy-to-use API for building applications on the Solana network using Go.

## Features

- Account creation and management
- Sending and receiving SOL
- Interacting with programs and smart contracts
- Transaction creation and signing
- Querying blockchain state and accounts

## Installation

To install the library, use `go get`:

```sh
go get github.com/donutnomad/solana-web3/
```

## Web3

### Examples

#### Getting Account Balance
```go
package main
import (
	"fmt"
	"github.com/donutnomad/solana-web3/web3"
)
func main() {
	client, err := web3.NewConnection(web3.Devnet.Url(), &web3.ConnectionConfig{
		Commitment: &web3.CommitmentConfirmed,
	})
	if err != nil {
		panic(err)
	}
	var account = web3.MustPublicKey("your public key")
	balance, err := client.GetBalance(account, web3.GetBalanceConfig{})
	if err != nil {
		panic(err)
	}
	fmt.Println("balance: ", balance)
}
```

#### Transferring SOL
```go
package main

import (
	"context"
	"fmt"
	"github.com/donutnomad/solana-web3/web3"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/mr-tron/base58"
)

// LamportsToSol Approximately convert fractional native tokens (lamports) into native tokens (SOL)
func LamportsToSol(lamports uint64) float64 {
	return float64(lamports) / float64(web3.LAMPORTS_PER_SOL)
}

// SolToLamports Approximately convert native tokens (SOL) into fractional native tokens (lamports)
func SolToLamports(sol float64) uint64 {
	return uint64(sol * float64(web3.LAMPORTS_PER_SOL))
}

func LamportsToString(lamports uint64) string {
	return fmt.Sprintf("%.9f", LamportsToSol(lamports))
}

func main() {
	var commitment = web3.CommitmentConfirmed
	client, err := web3.NewConnection(web3.Devnet.Url(), &web3.ConnectionConfig{
		Commitment: &commitment,
	})
	if err != nil {
		panic(err)
	}
	var privateKey = "your private key"
	decode, err := base58.Decode(privateKey)
	if err != nil {
		panic(err)
	}
	var from = web3.NewSigner(decode)
	// generate a random public key
	var to = web3.Keypair.Generate().PublicKey()
	var amount uint64 = web3.LAMPORTS_PER_SOL

	// check balance
	{
		balance, err := client.GetBalance(to, web3.GetBalanceConfig{Commitment: &commitment})
		if err != nil {
			panic(err)
		}
		rent, err := client.GetMinimumBalanceForRentExemption(0, &commitment)
		if err != nil {
			panic(err)
		}
		if amount < rent-balance {
			// Error: Transaction simulation failed: Transaction results in an account (1) with insufficient funds for rent
			// The amount needs to be greater than rent
			fmt.Printf("warn: insufficient funds for rent, balance: %s SOL, rent exemption: %s SOL, amount: %s SOL\n", LamportsToString(balance), LamportsToString(rent), LamportsToString(amount))
		}
	}

	// transaction
	var ins = system.NewTransferInstruction(
		amount, solana.PublicKey(from.PublicKey()), solana.PublicKey(to),
	).Build()
	var transaction = web3.Transaction{}
	err = transaction.AddInstruction3(ins)
	if err != nil {
		panic(err)
	}
	transaction.SetFeePayer(from.PublicKey())

	// send and confirm transaction
	signature, err := client.SendAndConfirmTransaction(context.Background(), transaction,
		[]web3.Signer{from}, web3.ConfirmOptions{
			SkipPreflight:       web3.Ref(false),
			PreflightCommitment: &web3.CommitmentProcessed,
			Commitment:          &web3.CommitmentProcessed,
		})
	if err != nil {
		panic(err)
	}
	fmt.Println("Signature:", signature)
}
```


## SPL_TOKEN_2022

### Extensions
- [x] `cpi_guard`
- [x] `default_account_state`
- [x] `group_member_pointer`
- [x] `group_pointer`
- [x] `immutable_owner`
- [x] `interest_bearing_mint`
- [x] `memo_transfer`
- [x] `metadata_pointer`
- [x] `mint_close_authority`
- [x] `non_transferable`
- [x] `permanent_delegate`
- [x] `token_group`
- [x] `transfer_fee`
- [x] `transfer_hook`

## MPL_TOKEN_METADATA

## TOKEN_METADATA

## Documentation

Full documentation is available on [pkg.go.dev](https://pkg.go.dev/github.com/donutnomad/solana-web3).

## Contributing

Contributions are welcome! Please open an issue or submit a pull request on GitHub.

1. Fork the repository
2. Create a new branch (`git checkout -b feature/your-feature`)
3. Commit your changes (`git commit -am 'Add some feature'`)
4. Push to the branch (`git push origin feature/your-feature`)
5. Create a new Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgements

This library was inspired by and based on the official [@solana/web3.js](https://github.com/solana-labs/solana-web3.js) library and the [github.com/gagliardetto](https://github.com/gagliardetto/solana-go) repository.
