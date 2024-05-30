package impl

import (
	"encoding/json"
	"fmt"
	"github.com/donutnomad/solana-web3/web3"
	"testing"
)

func Must[T any](t T, err error) T {
	if err != nil {
		panic(err)
	}
	return t
}

func TestParseTransferTransaction(t *testing.T) {
	connection := Must(web3.NewConnection(web3.MainnetBeta.Url(), &web3.ConnectionConfig{}))

	var signatures = []string{
		// SPL_TOKEN_2022 Transfer
		// https://solscan.io/tx/3mM962VAvY2SFYnYDTMcwWBmGhsDnGaLXVTmqKkox7w1eLdgs3JwpFRToD7hq7bmzL6HnN3LtKXU5n8YBNFbdV2P
		"3mM962VAvY2SFYnYDTMcwWBmGhsDnGaLXVTmqKkox7w1eLdgs3JwpFRToD7hq7bmzL6HnN3LtKXU5n8YBNFbdV2P",
		// SPL_TOKEN Transfer
		// https://solscan.io/tx/2xNAH3qMZtiojzRiuyvFneZEEUJBFtot3nyP8nWDg9RH3SxjwoyBJ7GMgkJLm3U2Dr7fDLJgvFKpfqNs8XxxazmP
		"2xNAH3qMZtiojzRiuyvFneZEEUJBFtot3nyP8nWDg9RH3SxjwoyBJ7GMgkJLm3U2Dr7fDLJgvFKpfqNs8XxxazmP",
		// SOL Transfer
		// Devnet
		// https://explorer.solana.com/tx/2XacUqppRHJ6hnSrwEDDKUTvLLsJPriQMFUpyj87j5iDZfJjXSW1AvPysL2zAbz8UVGyMqHfTeEAyCrac4j4fpfj?cluster=devnet
		"2XacUqppRHJ6hnSrwEDDKUTvLLsJPriQMFUpyj87j5iDZfJjXSW1AvPysL2zAbz8UVGyMqHfTeEAyCrac4j4fpfj",
	}
	for idx, _sig := range signatures {
		fmt.Println("==========>")
		if idx == len(signatures)-1 {
			connection = Must(web3.NewConnection(web3.Devnet.Url(), &web3.ConnectionConfig{}))
		}
		transaction, err := connection.GetTransaction(_sig, web3.GetVersionedTransactionConfig{
			Commitment: &web3.CommitmentFinalized,
		})
		if err != nil {
			panic(err)
		}
		transfer, sig := ParseTransfer(transaction.Transaction.Message, transaction.Meta, transaction.Transaction.Signatures)
		if len(transfer) > 0 {
			fmt.Println(sig)
			bs, _ := json.MarshalIndent(transfer, "", "\t")
			fmt.Println(string(bs))
		}
	}
}
