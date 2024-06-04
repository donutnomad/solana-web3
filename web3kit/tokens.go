package web3kit

import (
	"context"
	ata "github.com/donutnomad/solana-web3/associated_token_account"
	spltoken2022 "github.com/donutnomad/solana-web3/spl_token_2022"
	"github.com/donutnomad/solana-web3/web3"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/programs/token"
)

var Token = tokenKit{}

type tokenKit struct {
}

// Transfer executes a token transfer transaction on the Solana blockchain.
//
// This method facilitates transferring tokens from one wallet address to another. The method supports token transfers
// as well as SOL transfers when the mint is set to `web3.PublicKey{}`.
//
// Parameters:
// - ctx: The context to manage the lifecycle of the request and provide cancellation capabilities.
// - connection: A reference to the Solana web3 connection instance used to interact with the Solana blockchain.
// - payer: The signer responsible for paying the transaction fees.
// - from: The signer representing the sender's wallet address. This is not the associated token address.
// - to: The recipient's wallet address. This is not the associated token address.
// - mint: The mint address of the token being transferred. Use `web3.PublicKey{}` for SOL transfers (SystemProgram).
// - amount: The amount of tokens to transfer.
// - programId: The program ID that handles the token transfers, typically the token program ID.
// - confirm: A boolean indicating whether to wait for transaction confirmation.
// - options: Options for transaction confirmation, such as commitment level and preflight commitment.
//
// Returns:
// - web3.TransactionSignature: The signature of the transaction upon successful submission.
// - error: An error object in case of failure during transaction processing.
func (t tokenKit) Transfer(
	ctx context.Context,
	connection *web3.Connection,
	payer web3.Signer,
	from web3.Signer, // wallet address, not associated token address
	to web3.PublicKey, // wallet address, not associated token address
	mint web3.PublicKey,
	amount uint64,
	programId web3.PublicKey,
	confirm bool,
	options web3.ConfirmOptions,
	multiSigners ...web3.Signer,
) (web3.TransactionSignature, error) {
	var instructions, err = t.GetTransferInstructions(connection, payer.PublicKey(), from.PublicKey(), to, mint, amount, programId, Map(multiSigners, func(i int, t web3.Signer) web3.PublicKey {
		return t.PublicKey()
	})...)
	if err != nil {
		return "", err
	}
	blockhash, err := connection.GetLatestBlockhash(web3.GetLatestBlockhashConfig{
		Commitment: &web3.CommitmentFinalized,
	})
	if err != nil {
		return "", err
	}
	var transaction = web3.NewTransactionWithBlock(blockhash.Blockhash, blockhash.LastValidBlockHeight)
	transaction.SetFeePayer(payer.PublicKey())
	for _, ins := range instructions {
		transaction.AddInstruction2(ins)
	}
	if confirm {
		return connection.SendAndConfirmTransaction(ctx, *transaction, []web3.Signer{payer, from}, options)
	} else {
		return connection.SendTransaction(transaction, []web3.Signer{payer, from}, web3.SendOptions{
			SkipPreflight:       options.SkipPreflight,
			PreflightCommitment: options.PreflightCommitment,
			MaxRetries:          options.MaxRetries,
			MinContextSlot:      options.MinContextSlot,
		})
	}
}

// GetTransferInstructions Get a transfer instructions for token
// @param payer Pay the fee
// @param from Source (wallet address)
// @param to Destination (wallet address)
// @param amount Transferred
// @param programId web3.TokenProgramID,web3.TokenProgram2022ID,web3.SystemProgramID
func (tokenKit) GetTransferInstructions(
	connection *web3.Connection,
	payer web3.PublicKey, from web3.PublicKey, to web3.PublicKey,
	mint web3.PublicKey, amount uint64, programId web3.PublicKey,
	multiSigners ...web3.PublicKey,
) (_ []web3.TransactionInstruction, err error) {
	defer Recover(&err)

	var tx = web3.Transaction{}

	if programId == web3.SystemProgramID {
		ins := Must1(system.NewTransferInstruction(amount, solana.PublicKey(from), solana.PublicKey(to)).ValidateAndBuild())
		Must(tx.AddInstruction3(ins))
		return tx.ExportIns(), nil
	}

	associatedFrom := Must1(ata.FindAssociatedTokenAddress(from, mint, programId))
	associatedTo := Must1(ata.FindAssociatedTokenAddress(to, mint, programId))
	for _, accounts := range [][]web3.PublicKey{{associatedFrom, from}, {associatedTo, to}} {
		tokenAccountInfo := Must1(connection.GetAccountInfo(accounts[0], web3.GetAccountInfoConfig{}))
		if tokenAccountInfo != nil {
			continue
		}
		ins := Must1(ata.NewCreateInstruction(payer, accounts[0], accounts[1], mint, web3.SystemProgramID, programId).ValidateAndBuild())
		Must(tx.AddInstruction4(ins))
	}
	if programId == web3.TokenProgram2022ID {
		ins := Must1(spltoken2022.NewTransferInstruction(amount, associatedFrom, associatedTo, from).SetAuthorityAccount(from, multiSigners...).ValidateAndBuild())
		Must(tx.AddInstruction4(ins))
	} else {
		ins := Must1(token.NewTransferInstruction(amount, solana.PublicKey(associatedFrom), solana.PublicKey(associatedTo), solana.PublicKey(from), Map(multiSigners, func(i int, t web3.PublicKey) solana.PublicKey {
			return solana.PublicKey(t)
		})).ValidateAndBuild())
		Must(tx.AddInstruction3(ins))
	}
	return tx.ExportIns(), nil
}
