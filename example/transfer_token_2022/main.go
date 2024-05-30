package main

import (
	"context"
	"fmt"
	"github.com/donutnomad/solana-web3/associated_token_account"
	"github.com/donutnomad/solana-web3/example/common"
	spltoken2022 "github.com/donutnomad/solana-web3/spl_token_2022"
	"github.com/donutnomad/solana-web3/web3"
)

func main() {
	var commitment = web3.CommitmentConfirmed
	//var endpoint = web3.Devnet.Url()
	var endpoint = "http://127.0.0.1:8899"
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

	var owner = common.GetYourPrivateKey()

	var fromOwner = owner.PublicKey()
	var from = common.MustGetAssociatedTokenAddress(fromOwner, mint, tokenProgramId) // token account

	var toOwner = web3.Keypair.Generate().PublicKey()
	var to = common.MustGetAssociatedTokenAddress(toOwner, mint, tokenProgramId) // token account

	info, err := client.GetAccountInfo(from, web3.GetAccountInfoConfig{})
	if err != nil {
		return
	}
	if info == nil {
		// exe: `spl-token create-account -p TokenzQdBNbLqP5VEhdkAS6EPFLC1PHnBqCXEpPxuEb <MINT>`
		// exe: `spl-token mint <MINT> 100`
		panic("token account of `from` is not exists")
	}

	info, err = client.GetAccountInfo(to, web3.GetAccountInfoConfig{})
	if err != nil {
		return
	}

	var transaction = web3.Transaction{}

	// instruction 1
	if info == nil || info.Owner != web3.SPLAssociatedTokenAccountProgramID {
		fmt.Println("create associated token account", to)
		ins, err := associated_token_account.NewCreateInstruction(
			fromOwner, to, toOwner, mint, web3.SystemProgramID, tokenProgramId).ValidateAndBuild()
		if err != nil {
			panic(err)
		}
		if err = transaction.AddInstruction4(ins); err != nil {
			panic(err)
		}
	}

	// instruction 2
	transfer := spltoken2022.NewTransferInstruction(amount, from, to, fromOwner)
	ins, err := transfer.ValidateAndBuild()
	if err != nil {
		panic(err)
	}
	if err = transaction.AddInstruction4(ins); err != nil {
		panic(err)
	}

	// send and confirm transaction
	transaction.SetFeePayer(fromOwner)
	signature, err := client.SendAndConfirmTransaction(context.Background(), transaction,
		[]web3.Signer{owner}, web3.ConfirmOptions{
			SkipPreflight:       web3.Ref(false),
			PreflightCommitment: &web3.CommitmentProcessed,
			Commitment:          &web3.CommitmentProcessed,
		})
	if err != nil {
		panic(err)
	}
	fmt.Println("Transfer SPL_TOKEN_2022")
	fmt.Printf("FromOwner: %s, ToOwner: %s, From: %s, To: %s, Amount: %d\n", fromOwner, toOwner, from, to, amount)
	fmt.Println("Mint: ", mint)
	fmt.Println("Signature:", signature)
}
