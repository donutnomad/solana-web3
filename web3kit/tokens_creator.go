package web3kit

import (
	"context"
	"github.com/donutnomad/solana-web3/token_metadata"
	"github.com/donutnomad/solana-web3/web3"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/programs/token"
)

func (t tokenKit) CreateMint(
	ctx context.Context,
	conn *web3.Connection,
	payer web3.Signer,
	mintAuthority web3.ComplexSigner,
	freezeAuthority *web3.PublicKey,
	decimals uint8,
	keypair web3.Signer,
	tokenMetadata token_metadata.TokenMetadata,
	programId web3.PublicKey,
	commitment web3.Commitment,
) (_ web3.TransactionSignature, err error) {
	defer Recover(&err)

	var mint = keypair.PublicKey()
	var signers web3.SignerSlice = append(mintAuthority.Signers(), payer, keypair)
	var mintSize = token.MINT_SIZE
	var lamports = Must1(conn.GetMinimumBalanceForRentExemption(mintSize, &commitment))

	var builder = NewTransactionBuilder()
	builder.AddInstructions(system.NewCreateAccountInstruction(
		lamports,
		uint64(mintSize),
		programId.D(),
		payer.PublicKey().D(),
		mint.D(),
	).Build())
	builder.AddInstructions2(token.NewInitializeMint2Instruction(decimals, mintAuthority.PublicKey.D(), freezeAuthority.D(), mint.D()).ValidateAndBuild())
	builder.AddInsBuilder(MetaPlex.GetCreateIns(
		tokenMetadata.Name,
		tokenMetadata.Symbol,
		tokenMetadata.Uri,
		payer.PublicKey(),
		mint,
		mintAuthority.PublicKey,
		mintAuthority.PublicKey,
		programId,
	))

	transaction := Must1(builder.SetFeePayer(payer.PublicKey()).Build())
	return conn.SendAndConfirmTransaction(ctx, *transaction, signers, web3.ConfirmOptions{
		SkipPreflight:       web3.Ref(false),
		PreflightCommitment: &commitment,
		Commitment:          &commitment,
	})
}
