package web3kit

import (
	"context"
	"github.com/donutnomad/solana-web3/spl_token_2022"
	"github.com/donutnomad/solana-web3/spl_token_2022/extension/default_account_state"
	"github.com/donutnomad/solana-web3/spl_token_2022/extension/transfer_fee"
	"github.com/donutnomad/solana-web3/token_metadata"
	"github.com/donutnomad/solana-web3/web3"
	binary "github.com/gagliardetto/binary"
)

// ParseDefaultAccountState Extension: default_account_state
func (t tokenKit2022) ParseDefaultAccountState(data []byte) (*default_account_state.DefaultAccountState, error) {
	return parseExtension[*default_account_state.DefaultAccountState](spl_token_2022.ExtensionTypeDefaultAccountState, data)
}

// ParseTransferFeeConfig Extension: transfer_fee
func (t tokenKit2022) ParseTransferFeeConfig(data []byte) (*transfer_fee.TransferFeeConfig, error) {
	return parseExtension[*transfer_fee.TransferFeeConfig](spl_token_2022.ExtensionTypeTransferFeeConfig, data)
}

// GetTokenMetadata Extension: token_metadata
func (t tokenKit2022) GetTokenMetadata(
	ctx context.Context,
	connection *web3.Connection,
	mint, programId web3.PublicKey,
	config web3.GetAccountInfoConfig,
) (*token_metadata.TokenMetadata, error) {
	mintInfo, err := t.GetMint(ctx, connection, mint, programId, config)
	if err != nil {
		return nil, err
	}
	return t.ParseTokenMetadata(mintInfo.TlvData)
}

// ParseTokenMetadata Extension: token_metadata
func (t tokenKit2022) ParseTokenMetadata(data []byte) (*token_metadata.TokenMetadata, error) {
	return parseExtension[*token_metadata.TokenMetadata](spl_token_2022.ExtensionTypeTokenMetadata, data)
}

func parseExtension[T binary.BinaryUnmarshaler](extension spl_token_2022.ExtensionType, data []byte) (T, error) {
	var zero T
	extensionData, err := Token2022.GetExtensionData(extension, data)
	if err != nil {
		return zero, err
	}
	return decodeObject[T](extensionData)
}
