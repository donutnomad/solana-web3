# Solana Web3 Library (Golang)

[![Go Reference](https://pkg.go.dev/badge/github.com/donutnomad/solana-web3.svg)](https://pkg.go.dev/github.com/donutnomad/solana-web3)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

This is a Go library for interacting with the Solana blockchain, inspired by the
official [@solana/web3.js](https://github.com/solana-labs/solana-web3.js) JavaScript library. It provides a simple and
easy-to-use API for building applications on the Solana network using Go.

## Features

- [Solana RPC Methods](https://solana.com/docs/rpc/http/)
- Transfer SOL, Token, Token2022
- Token2022 Extensions
- Metaplex Token Metadata
- Transaction creation and signing
- Easy to create a Token

## Installation

To install the library, use `go get`:

```sh
go get -u github.com/donutnomad/solana-web3
```

## Web3

### Usage

```go
package main

import "context"
import "github.com/donutnomad/solana-web3/web3"
import "github.com/donutnomad/solana-web3/web3kit"

func main() {
	client, err := web3.NewConnection(web3.Devnet.Url(), &web3.ConnectionConfig{
		Commitment: &web3.CommitmentConfirmed,
	})
	if err != nil {
		panic(err)
	}
	// generate a random key
	var keypair = web3.Keypair.Generate()
	_ = keypair
	// get minimum balance for rent exemption
	rent, err := client.GetMinimumBalanceForRentExemption(0, nil)
	if err != nil {
		panic(err)
	}
	_ = rent
	// transfer sol/token/token2022
	// More: https://github.com/donutnomad/solana-web3/tree/main/example/transfer_sol
	signature, err := web3kit.Token.Transfer(context.Background(), client,
		owner, owner, toOwner, mint, amount, tokenProgramId, true, web3.ConfirmOptions{
			SkipPreflight:       web3.Ref(false),
			PreflightCommitment: &commitment,
			Commitment:          &commitment,
		})
	if err != nil {
		panic(err)
	}
	// get token name,symbol,logo
	var mint = web3.MustPublicKey("FH3i2zWEZRKQVkdqKknkfXzYgrGSTcc48VnwoJduf2o1")
	name, symbol, logo, err := web3kit.Token.GetTokenName(context.Background(),
		client, mint, &web3.CommitmentProcessed,
	)
	if err != nil {
		panic(err)
	}
	_ = name
	_ = symbol
	_ = logo
}

```

### Examples

This table provides an overview of the examples available in
the [examples directory](https://github.com/donutnomad/solana-web3/tree/main/example/).

| Example                                                                                                | Description                                                          |
|--------------------------------------------------------------------------------------------------------|----------------------------------------------------------------------|
| [balance](https://github.com/donutnomad/solana-web3/tree/main/example/balance)                         | An example to retrieve the SOL balance of a user.                    |
| [parse](https://github.com/donutnomad/solana-web3/tree/main/example/parse)                             | An example to parse SOL/Token/Token2022 transfers on the blockchain. |
| [transfer_sol](https://github.com/donutnomad/solana-web3/tree/main/example/transfer_sol)               | An example to send SOL tokens.                                       |
| [transfer_token](https://github.com/donutnomad/solana-web3/tree/main/example/transfer_token)           | An example to send SPL TOKEN tokens.                                 |
| [transfer_token_2022](https://github.com/donutnomad/solana-web3/tree/main/example/transfer_token_2022) | An example to send SPL TOKEN 2022 tokens.                            |
| [create_token](https://github.com/donutnomad/solana-web3/tree/main/example/create_token)               | An example to create solana token with metadata.                     |
| [create_token_2022](https://github.com/donutnomad/solana-web3/tree/main/example/create_token_2022)     | An example to create solana token 2022 with metadata.                |

### Programs

| Program                                                                                                  | Description                                                                                                                                                                                                        |
|----------------------------------------------------------------------------------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| [Metaplex Token Metadata](https://github.com/donutnomad/solana-web3/tree/main/mpl_token_metadata)        | Metaplex Token Metadata https://github.com/metaplex-foundation/mpl-token-metadata                                                                                                                                  |
| [Associated Token Account](https://github.com/donutnomad/solana-web3/tree/main/associated_token_account) | Solana Token Associated Token Account https://github.com/solana-labs/solana-program-library/tree/master/associated-token-account/program                                                                           |
| [Token Program 2022](https://github.com/donutnomad/solana-web3/tree/main/spl_token_2022)                 | Solana Token Program 2022. https://github.com/solana-labs/solana-program-library/tree/master/token/program-2022 <br/>Supported Extensions:cpi_guard,default_account_state...[More](#Token Program 2022 Extensions) |
| Token Program                                                                                            | Solana Token Program https://github.com/solana-labs/solana-program-library/tree/master/token/program                                                                                                               |
| [Token Program 2022 Metadata](https://github.com/donutnomad/solana-web3/tree/main/token_metadata)        | Token Metadata for Token Program 2022 https://github.com/solana-labs/solana-program-library/tree/master/token-metadata                                                                                             |

## Token Program 2022 Extensions

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
- [ ] `confidential_transfer`
- [ ] `confidential_transfer_fee`

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

This library was inspired by and based on the official [@solana/web3.js](https://github.com/solana-labs/solana-web3.js)
library and the [github.com/gagliardetto](https://github.com/gagliardetto/solana-go) repository.
