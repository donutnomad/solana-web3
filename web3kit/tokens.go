package web3kit

import (
	"context"
	"errors"
	ata "github.com/donutnomad/solana-web3/associated_token_account"
	spltoken2022 "github.com/donutnomad/solana-web3/spl_token_2022"
	"github.com/donutnomad/solana-web3/web3"
	"github.com/donutnomad/solana-web3/web3kit/solanatokenlist"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/programs/token"
	"strings"
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
	transaction.AddInstructions(instructions...)
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
		Must(tx.AddInsBuilder(system.NewTransferInstruction(amount, from.D(), to.D())))
		return tx.ExportIns(), nil
	}

	associatedFrom := Must1(ata.FindAssociatedTokenAddress(from, mint, programId))
	associatedTo := Must1(ata.FindAssociatedTokenAddress(to, mint, programId))
	for _, accounts := range [][]web3.PublicKey{{associatedFrom, from}, {associatedTo, to}} {
		tokenAccountInfo := Must1(connection.GetAccountInfo(accounts[0], web3.GetAccountInfoConfig{}))
		if tokenAccountInfo != nil {
			continue
		}
		Must(tx.AddInsBuilder(
			ata.NewCreateInstruction(payer, accounts[0], accounts[1], mint, web3.SystemProgramID, programId),
		))
	}
	if programId == web3.TokenProgram2022ID {
		Must(tx.AddInsBuilder(
			spltoken2022.NewTransferInstruction(amount, associatedFrom, associatedTo, from).SetAuthorityAccount(from, multiSigners...),
		))
	} else {
		Must(tx.AddInsBuilder(
			token.NewTransferInstruction(amount, associatedFrom.D(), associatedTo.D(), from.D(), Map(multiSigners, convertPublicKey))),
		)
	}
	return tx.ExportIns(), nil
}

func (t tokenKit) GetMint(
	ctx context.Context,
	connection *web3.Connection,
	mint, programId web3.PublicKey,
	config web3.GetAccountInfoConfig,
) (*MintInfo, error) {
	return Token2022.GetMint(ctx, connection, mint, programId, config)
}

func (t tokenKit) ParseMint(data []byte) (*spltoken2022.Mint, error) {
	return Token2022.ParseMint(data)
}

func (t tokenKit) UnpackMint(mintAddress web3.PublicKey, info *web3.AccountInfoD, programId web3.PublicKey) (*MintInfo, error) {
	return Token2022.UnpackMint(mintAddress, info, programId)
}

func (t tokenKit) GetTokenAccount(
	ctx context.Context,
	connection *web3.Connection,
	account, programId web3.PublicKey,
	config web3.GetAccountInfoConfig,
) (*TokenAccount, error) {
	return Token2022.GetTokenAccount(ctx, connection, account, programId, config)
}

func (t tokenKit) ParseTokenAccount(data []byte) (*spltoken2022.Account, error) {
	return Token2022.ParseTokenAccount(data)
}

func (t tokenKit) UnpackTokenAccount(tokenAccount web3.PublicKey, info *web3.AccountInfoD, programId web3.PublicKey) (*TokenAccount, error) {
	return Token2022.UnpackTokenAccount(tokenAccount, info, programId)
}

func (t tokenKit) FindTokenAccounts(
	ctx context.Context,
	connection *web3.Connection,
	mint, programId web3.PublicKey,
	option GetProgramAccountsOption,
	commitment web3.Commitment,
) ([]ProgramAccount, error) {
	return Token2022.FindTokenAccounts(ctx, connection, mint, programId, option, commitment)
}

func (t tokenKit) GetTokenName(ctx context.Context, connection *web3.Connection, mint web3.PublicKey, commitment *web3.Commitment) (name, symbol, uri string, err error) {
	defer Recover(&err)

	name, symbol, uri, ok := solanatokenlist.GetTokenInfo(mint.Base58())
	if ok {
		return
	}

	metaplexMetadata := Must1(MetaPlex.GetMetadata(ctx, connection, mint, commitment))
	clear_ := func(input string) string {
		return strings.TrimRight(input, "\u0000")
	}
	if metaplexMetadata != nil {
		d := metaplexMetadata.Data
		return clear_(d.Name), clear_(d.Symbol), clear_(d.Uri), nil
	} else {
		metadata, err := Token2022.GetTokenMetadata(ctx, connection, mint, web3.TokenProgram2022ID, web3.GetAccountInfoConfig{
			Commitment: commitment,
		})
		if err != nil {
			if !(errors.Is(err, TokenAccountNotFoundErr) || errors.Is(err, TokenInvalidAccountOwnerErr)) {
				return "", "", "", err
			}
		}
		if metadata != nil {
			return clear_(metadata.Name), clear_(metadata.Symbol), clear_(metadata.Uri), nil
		}
	}
	return "", "", "", nil
}

var convertPublicKey = func(i int, t web3.PublicKey) solana.PublicKey {
	return t.D()
}
