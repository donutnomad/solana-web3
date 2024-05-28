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
	balance, err := client.GetBalance(web3.SystemProgramID, web3.GetBalanceConfig{})
	if err != nil {
		panic(err)
	}
	fmt.Println("balance: ", balance)
}
