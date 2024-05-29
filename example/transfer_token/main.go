package main

import (
	"context"
	"fmt"
	"github.com/donutnomad/solana-web3/web3"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/token"
	"github.com/mr-tron/base58"
)

func mustGetAssociatedTokenAddress(owner, mint web3.PublicKey) web3.PublicKey {
	address, seed, err := solana.FindAssociatedTokenAddress(solana.PublicKey(owner), solana.PublicKey(mint))
	if err != nil {
		panic(err)
	}
	_ = seed
	return web3.PublicKey(address)
}

func main() {
	var commitment = web3.CommitmentConfirmed
	client, err := web3.NewConnection(web3.Devnet.Url(), &web3.ConnectionConfig{
		Commitment: &commitment,
	})
	if err != nil {
		panic(err)
	}
	// create a token with command: `spl-token create-token`
	var mint = web3.MustPublicKey("3S2EY2KBqKfYB6rkSN6LLNsTnVbSZ8U9bFqXzEpZZnYp")
	var amount uint64 = 10

	var privateKey = "your private key"
	decode, err := base58.Decode(privateKey)
	if err != nil {
		panic(err)
	}
	var owner = web3.NewSigner(decode)

	var from = mustGetAssociatedTokenAddress(owner.PublicKey(), mint) // token account

	// create a token account with command: `spl-token create-account --owner <WALLET_ADDRESS> <MINT_ADDRESS>`
	// make sure your token account is created, otherwise, the transfer will fail.
	var toOwner = web3.MustPublicKey("jCXxmo3KRiMFkS9MwVXbD7qb3U4iH6Y5LxzD1VkB4Lc")
	var to = mustGetAssociatedTokenAddress(toOwner, mint) // token account

	var transaction = web3.Transaction{}
	transfer := token.NewTransferInstruction(amount, solana.PublicKey(from), solana.PublicKey(to), solana.PublicKey(owner.PublicKey()), nil)
	err = transaction.AddInstruction3(transfer.Build())
	if err != nil {
		panic(err)
	}
	transaction.SetFeePayer(owner.PublicKey())

	// send and confirm transaction
	signature, err := client.SendAndConfirmTransaction(context.Background(), transaction,
		[]web3.Signer{owner}, web3.ConfirmOptions{
			SkipPreflight:       web3.Ref(false),
			PreflightCommitment: &web3.CommitmentProcessed,
			Commitment:          &web3.CommitmentProcessed,
		})
	if err != nil {
		panic(err)
	}
	fmt.Println("Signature:", signature)
}
