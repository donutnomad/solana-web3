package web3kit

import (
	"context"
	"errors"
	"fmt"
	ata "github.com/donutnomad/solana-web3/associated_token_account"
	mtm "github.com/donutnomad/solana-web3/mpl_token_metadata"
	. "github.com/donutnomad/solana-web3/spl_token_2022"
	. "github.com/donutnomad/solana-web3/spl_token_2022/extension"
	"github.com/donutnomad/solana-web3/token_metadata"
	"github.com/donutnomad/solana-web3/web3"
	"github.com/gagliardetto/solana-go/programs/system"
)

type MetadataProvider int

const (
	MetadataPlex MetadataProvider = iota
	SplToken2022
)

func (t tokenKit2022) CreateMint(
	ctx context.Context,
	conn *web3.Connection,
	payer web3.Signer,
	mintAuthority web3.ComplexSigner,
	freezeAuthority *web3.PublicKey,
	decimals uint8,
	keypair web3.Signer,
	extensions []ExtensionInitializationParams,
	initialSupply *uint64,
	tokenMetadata token_metadata.TokenMetadata,
	metadataProvider MetadataProvider,
	programId web3.PublicKey,
	commitment web3.Commitment,
) (_ web3.TransactionSignature, err error) {
	defer Recover(&err)

	var mint = keypair.PublicKey()
	var signers web3.SignerSlice = append(mintAuthority.Signers(), payer, keypair)

	// add metadata extension
	if ExistsExtensionType(extensions, ExtensionTypeMetadataPointer) {
		return "", errors.New("the extension type 'MetadataPointer' already exists")
	}
	if metadataProvider == SplToken2022 {
		// the metadata address of spl-token-2022 is the mint address
		_mintAuthority := mintAuthority.PublicKey
		extensions = append([]ExtensionInitializationParams{NewMetadataPointerParamsInitialize(&_mintAuthority, &mint, mint)}, extensions...)
	} else if metadataProvider == MetadataPlex {
		// metaplex-token-metadata create_mint:
		///       When creating a mint with spl-token-2022, the following extensions are enabled:
		///       - mint close authority extension enabled and set to the metadata account
		///       - metadata pointer extension enabled and set to the metadata account
		if ExistsExtensionType(extensions, ExtensionTypeMintCloseAuthority) {
			return "", errors.New("the extension type 'MintCloseAuthority' already exists")
		}
		// the authority must be `none`
		// the metadata address must be pda
		metadataPDA := Must1(mtm.FindAssociatedAddress(mint))
		extensions = append([]ExtensionInitializationParams{NewMetadataPointerParamsInitialize(nil, &metadataPDA, mint)}, extensions...)
		// the close authority must be pda
		extensions = append([]ExtensionInitializationParams{NewMintCloseAuthorityParams(&metadataPDA)}, extensions...)
	}

	extensions = DeDupBy(extensions, func(t ExtensionInitializationParams) ExtensionType {
		return t.ExtensionType()
	})
	mintSize := Must1(Token2022.GetMintLen(Map(extensions, func(_ int, t ExtensionInitializationParams) ExtensionType {
		return t.ExtensionType()
	})))
	var addDataLength uint64 = 0
	if metadataProvider == SplToken2022 {
		addDataLength = Token2022.AddTypeAndLengthToLen(GetSize(&tokenMetadata))
	}
	lamports := Must1(conn.GetMinimumBalanceForRentExemption(int(mintSize+addDataLength), &commitment))

	var builder = NewTransactionBuilder()
	if mintAuthority.IsMultiSig() {
		// check threshold
		count := len(mintAuthority.Signers())
		if count < 1 || count > 11 {
			return "", errors.New("invalid threshold")
		}
		// check multi sig account
		accountInfo := Must1(conn.GetAccountInfo(mintAuthority.PublicKey, web3.GetAccountInfoConfig{
			Commitment: &web3.CommitmentFinalized,
		}))
		if accountInfo == nil {
			return "", errors.New("multi sig account not found")
		}
		if len(accountInfo.Data.Content) != MULTISIG_SIZE {
			return "", errors.New("invalid account data size of multi sig")
		}
	}
	builder.AddInstructions(system.NewCreateAccountInstruction(
		lamports,
		mintSize,
		programId.D(),
		payer.PublicKey().D(),
		mint.D(),
	).Build())

	var defaultAccountIsFrozen = false
	for _, extension := range extensions {
		if extension.ExtensionType() == ExtensionTypeDefaultAccountState {
			p := extension.(DefaultAccountStateParams)
			defaultAccountIsFrozen = p.State == AccountStateFrozen
		}
		ret := Must1(ExtensionInitializationParamsToInstruction(extension, mint, programId))
		if ret == nil {
			return "", errors.New("not supported extension type")
		}
		builder.AddInstructions(ret)
	}
	builder.AddInstructions2(NewInitializeMint2Instruction(decimals, mintAuthority.PublicKey, freezeAuthority, mint).SetProgramId(&programId).ValidateAndBuild())

	// add metadata
	if metadataProvider == SplToken2022 {
		if tokenMetadata.UpdateAuthority.IsZero() && len(tokenMetadata.AdditionalMetadata) > 0 {
			fmt.Println("metadata updateAuthority is zero, ignore additionalMetadata")
		}
		builder.AddInstructions2(TokenMeta.GetCreateIns(
			tokenMetadata.Name,
			tokenMetadata.Symbol,
			tokenMetadata.Uri,
			mint,
			tokenMetadata.UpdateAuthority,
			mintAuthority.PublicKey,
			programId,
			tokenMetadata.AdditionalMetadata,
		))
	} else if metadataProvider == MetadataPlex {
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
	} else {
		return "", errors.New("metadata provider is not support")
	}

	// initial supply
	if initialSupply != nil && *initialSupply > uint64(0) {
		owner := mintAuthority.PublicKey
		associatedTokenProgramId := web3.SPLAssociatedTokenAccountProgramID
		associatedToken := Must1(ata.FindAssociatedTokenAddress(owner, mint, web3.TokenProgram2022ID))
		builder.AddInsBuilder(ata.NewCreateInstruction(
			payer.PublicKey(),
			associatedToken,
			owner,
			mint,
			web3.SystemProgramID,
			programId,
		).SetProgramId(&associatedTokenProgramId))

		if defaultAccountIsFrozen {
			builder.AddInsBuilder(NewThawAccountInstruction(associatedToken, mint, owner).SetProgramId(&programId))
		}

		builder.AddInsBuilder(NewMintToInstruction(*initialSupply, mint, associatedToken, web3.PublicKey{}).
			SetAuthorityAccount(mintAuthority.PublicKey, mintAuthority.Addresses()...).
			SetProgramId(&programId))
	}
	transaction := Must1(builder.SetFeePayer(payer.PublicKey()).Build())

	return conn.SendAndConfirmTransaction(ctx, *transaction, signers, web3.ConfirmOptions{
		SkipPreflight:       web3.Ref(false),
		PreflightCommitment: &commitment,
		Commitment:          &commitment,
	})
}

type CreateTokenArgs2022 struct {
	BasicMetadata
	Decimals            uint8
	InitialSupply       *uint64
	EnableWhitelist     bool
	EnableBlacklist     bool
	EnableForceTransfer bool
	Fee                 *uint16 // max: 100_00(100%)
	MaximumFee          uint64
}

type BasicMetadata struct {
	Image       []byte
	Name        string
	Symbol      string
	Description string
	Additional  map[string]string
}

type FileProvider interface {
	MetadataURI(ctx context.Context, connection *web3.Connection, payer web3.Signer, basic BasicMetadata) (url string, err error)
}

func (t tokenKit2022) CreateToken(
	ctx context.Context,
	connection *web3.Connection,
	payer web3.Signer,
	owner web3.ComplexSigner,
	args CreateTokenArgs2022,
	metaProvider FileProvider,
	metaType MetadataProvider,
	commitment web3.Commitment,
) (web3.TransactionSignature, web3.PublicKey, error) {
	var mintAuthority = owner.PublicKey
	var freezeAuthority = &mintAuthority
	var permanentDelegate = owner.PublicKey
	var transferFeeConfigAuthority = &mintAuthority
	var withdrawWithheldAuthority = &mintAuthority

	metadataURI, err := metaProvider.MetadataURI(ctx, connection, payer, args.BasicMetadata)
	if err != nil {
		return "", web3.PublicKey{}, err
	}

	tokenKeypair := web3.Keypair.Generate()
	var extensions_ []ExtensionInitializationParams

	if args.EnableWhitelist {
		var state = AccountStateInitialized
		if args.EnableWhitelist {
			state = AccountStateFrozen
		}
		extensions_ = append(extensions_, NewDefaultAccountStateParams(state))
	} else if args.EnableBlacklist {
		// pass
	} else {
		freezeAuthority = nil
	}
	if args.EnableForceTransfer {
		extensions_ = append(extensions_, NewPermanentDelegateParams(permanentDelegate))
	}
	if args.Fee != nil {
		extensions_ = append(extensions_, NewTransferFeeConfigParams(
			max(100_00 /*100%*/, *args.Fee),
			max(0, args.MaximumFee),
			transferFeeConfigAuthority,
			withdrawWithheldAuthority,
		))
	}
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
	sig, err := Token2022.CreateMint(
		ctx,
		connection,
		payer,
		owner,
		freezeAuthority,
		args.Decimals,
		tokenKeypair,
		extensions_,
		args.InitialSupply,
		tokenMetadata,
		metaType,
		web3.TokenProgram2022ID,
		commitment,
	)
	if err != nil {
		return "", web3.PublicKey{}, err
	}
	return sig, tokenKeypair.PublicKey(), nil
}
