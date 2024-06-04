package main

import (
	"context"
	"fmt"
	"github.com/donutnomad/solana-web3/example/common"
	"github.com/donutnomad/solana-web3/web3"
	"github.com/donutnomad/solana-web3/web3kit"
)

func main() {
	var commitment = web3.CommitmentProcessed
	client, err := web3.NewConnection(web3.Devnet.Url(), &web3.ConnectionConfig{
		Commitment: &commitment,
	})
	if err != nil {
		panic(err)
	}

	var from = common.GetYourPrivateKey()
	var to = web3.Keypair.Generate().PublicKey() // generate a random public key
	var amount = web3.LAMPORTS_PER_SOL

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
			fmt.Printf("warn: insufficient funds for rent, balance: %s SOL, rent exemption: %s SOL, amount: %s SOL\n", common.LamportsToString(balance), common.LamportsToString(rent), common.LamportsToString(amount))
		}
	}

	signature, err := web3kit.Token.Transfer(context.Background(), client, from, from, to, web3.PublicKey{}, amount, web3.SystemProgramID, true, web3.ConfirmOptions{
		SkipPreflight:       web3.Ref(false),
		PreflightCommitment: &commitment,
		Commitment:          &commitment,
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("Transfer SOL")
	fmt.Printf("From: %s, To: %s,  Amount: %d\n", from, to, amount)
	fmt.Println("Signature:", signature)
}
