package main

import (
	"context"
	"fmt"
	"github.com/donutnomad/solana-web3/example/common"
	"github.com/donutnomad/solana-web3/test"
	"github.com/donutnomad/solana-web3/web3"
	"github.com/donutnomad/solana-web3/web3kit"
)

func main() {
	var commitment = web3.CommitmentProcessed
	var endpoint = web3.Devnet.Url()
	var tokenProgramId = web3.TokenProgram2022ID

	client, err := web3.NewConnection(endpoint, &web3.ConnectionConfig{
		Commitment: &commitment,
	})
	if err != nil {
		panic(err)
	}
	// create a token with command: `spl-token create-token -p TokenzQdBNbLqP5VEhdkAS6EPFLC1PHnBqCXEpPxuEb`
	var mint = web3.MustPublicKey("FzVc5VPXfSQrPwEXMWFspaSeri682bHQps8easnJ1C92")
	var amount uint64 = 10

	var owner = test.GetYourPrivateKey()
	var toOwner = web3.Keypair.Generate().PublicKey()

	// send and confirm transaction
	signature, err := web3kit.Token.Transfer(context.Background(), client,
		owner, owner, toOwner, mint, amount, tokenProgramId, true, web3.ConfirmOptions{
			SkipPreflight:       web3.Ref(false),
			PreflightCommitment: &commitment,
			Commitment:          &commitment,
		})
	if err != nil {
		panic(err)
	}

	var from = common.MustGetAssociatedTokenAddress(owner.PublicKey(), mint, tokenProgramId) // token account
	var to = common.MustGetAssociatedTokenAddress(toOwner, mint, tokenProgramId)             // token account
	fmt.Println("Transfer SPL_TOKEN_2022")
	fmt.Printf("FromOwner: %s, ToOwner: %s, From: %s, To: %s, Amount: %d\n", owner.PublicKey(), toOwner, from, to, amount)
	fmt.Println("Mint: ", mint)
	fmt.Println("Signature:", signature)
}
