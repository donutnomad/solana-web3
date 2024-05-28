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
