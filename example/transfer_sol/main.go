package main

import (
	"context"
	"fmt"
	"github.com/donutnomad/solana-web3/example/common"
	"github.com/donutnomad/solana-web3/web3"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
)

func main() {
	var commitment = web3.CommitmentConfirmed
	client, err := web3.NewConnection(web3.Devnet.Url(), &web3.ConnectionConfig{
		Commitment: &commitment,
	})
	if err != nil {
		panic(err)
	}

	var from = common.GetYourPrivateKey()
	// generate a random public key
	var to = web3.Keypair.Generate().PublicKey()
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

	fmt.Println("Transfer SOL")
	fmt.Printf("From: %s, To: %s,  Amount: %d\n", from, to, amount)
	fmt.Println("Signature:", signature)
}
