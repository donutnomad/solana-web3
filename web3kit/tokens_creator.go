package web3kit

import (
	"context"
	ata "github.com/donutnomad/solana-web3/associated_token_account"
	"github.com/donutnomad/solana-web3/token_metadata"
	"github.com/donutnomad/solana-web3/web3"
	"github.com/gagliardetto/solana-go"
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
	initialSupply *uint64,
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
	// initial supply
	if initialSupply != nil && *initialSupply > uint64(0) {
		owner := mintAuthority.PublicKey
		associatedTokenProgramId := web3.SPLAssociatedTokenAccountProgramID
		associatedToken := Must1(ata.FindAssociatedTokenAddress(owner, mint, programId))
		builder.AddInsBuilder(ata.NewCreateInstruction(
			payer.PublicKey(),
			associatedToken,
			owner,
			mint,
			web3.SystemProgramID,
			programId,
		).SetProgramId(&associatedTokenProgramId))
		multiSigners := Map(mintAuthority.Addresses(), func(i int, t web3.PublicKey) solana.PublicKey {
			return solana.PublicKey(t)
		})
		builder.AddInsBuilder(token.NewMintToInstruction(*initialSupply, mint.D(), associatedToken.D(), mintAuthority.PublicKey.D(), multiSigners))
	}

	transaction := Must1(builder.SetFeePayer(payer.PublicKey()).Build())
	return conn.SendAndConfirmTransaction(ctx, *transaction, signers, web3.ConfirmOptions{
		SkipPreflight:       web3.Ref(false),
		PreflightCommitment: &commitment,
		Commitment:          &commitment,
	})
}

type CreateTokenArgs struct {
	BasicMetadata
	Decimals      uint8
	InitialSupply *uint64
}

func (t tokenKit) CreateToken(
	ctx context.Context,
	connection *web3.Connection,
	payer web3.Signer,
	owner web3.ComplexSigner,
	args CreateTokenArgs,
	metaProvider FileProvider,
	commitment web3.Commitment,
) (web3.TransactionSignature, web3.PublicKey, error) {
	var mintAuthority = owner.PublicKey
	var freezeAuthority = &mintAuthority

	metadataURI, err := metaProvider.MetadataURI(ctx, connection, payer, args.BasicMetadata)
	if err != nil {
		return "", web3.PublicKey{}, err
	}

	tokenKeypair := web3.Keypair.Generate()
	var additionalMetadata []struct {
		Key   string
		Value string
	}
	for key, value := range args.BasicMetadata.Additional {
		additionalMetadata = append(additionalMetadata, struct {
			Key   string
			Value string
		}{Key: key, Value: value})
	}
	tokenMetadata := token_metadata.TokenMetadata{
		UpdateAuthority:    mintAuthority,
		Mint:               tokenKeypair.PublicKey(),
		Name:               args.Name,
		Symbol:             args.Symbol,
		Uri:                metadataURI,
		AdditionalMetadata: additionalMetadata,
	}
	sig, err := Token.CreateMint(
		ctx,
		connection,
		payer,
		owner,
		freezeAuthority,
		args.Decimals,
		tokenKeypair,
		args.InitialSupply,
		tokenMetadata,
		web3.TokenProgramID,
		commitment,
	)
	if err != nil {
		return "", web3.PublicKey{}, err
	}
	return sig, tokenKeypair.PublicKey(), nil
}
