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
