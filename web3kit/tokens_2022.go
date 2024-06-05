package web3kit

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	ata "github.com/donutnomad/solana-web3/associated_token_account"
	. "github.com/donutnomad/solana-web3/spl_token_2022"
	"github.com/donutnomad/solana-web3/spl_token_2022/extension/cpi_guard"
	"github.com/donutnomad/solana-web3/spl_token_2022/extension/default_account_state"
	"github.com/donutnomad/solana-web3/spl_token_2022/extension/immutable_owner"
	"github.com/donutnomad/solana-web3/spl_token_2022/extension/metadata_pointer"
	"github.com/donutnomad/solana-web3/spl_token_2022/extension/mint_close_authority"
	"github.com/donutnomad/solana-web3/spl_token_2022/extension/non_transferable"
	"github.com/donutnomad/solana-web3/spl_token_2022/extension/permanent_delegate"
	"github.com/donutnomad/solana-web3/spl_token_2022/extension/transfer_fee"
	"github.com/donutnomad/solana-web3/spl_token_2022/extension/transfer_hook"
	"github.com/donutnomad/solana-web3/web3"
)

var Token2022 = tokenKit2022{}

type tokenKit2022 struct {
}

var ACCOUNT_TYPE_SIZE = 1
var MULTISIG_SIZE = 355

func (t tokenKit2022) MEMO_TRANSFER_SIZE() int {
	return 1
}
func (t tokenKit2022) INTEREST_BEARING_MINT_CONFIG_STATE_SIZE() int {
	return 52
}
func (t tokenKit2022) TRANSFER_FEE_AMOUNT_SIZE() int {
	return 8
}
func (t tokenKit2022) LENGTH_SIZE() int {
	return 2
}
func (t tokenKit2022) TYPE_SIZE() int {
	return 2
}

func (t tokenKit2022) AddTypeAndLengthToLen(l int) uint64 {
	return t.AddTypeAndLengthToLen2(uint64(l))
}

func (t tokenKit2022) AddTypeAndLengthToLen2(l uint64) uint64 {
	return l + uint64(t.TYPE_SIZE()) + uint64(t.LENGTH_SIZE())
}

func (t tokenKit2022) GetMintLen(extensionTypes []ExtensionType) (uint64, error) {
	return t.GetLen(extensionTypes, MINT_SIZE)
}

func (t tokenKit2022) GetLen(extensionTypes []ExtensionType, baseSize int) (uint64, error) {
	if len(extensionTypes) == 0 {
		return uint64(baseSize), nil
	} else {
		var accountLength = uint64(ACCOUNT_SIZE + ACCOUNT_TYPE_SIZE)
		for _, ext := range extensionTypes {
			ret, err := t.GetTypeLen(ext)
			if err != nil {
				return 0, err
			}
			accountLength += t.AddTypeAndLengthToLen(ret)
		}
		if accountLength == uint64(MULTISIG_SIZE) {
			return accountLength + uint64(t.TYPE_SIZE()), nil
		} else {
			return accountLength, nil
		}
	}
}

func (t tokenKit2022) GetExtensionType(tlvData []byte) ([]ExtensionType, error) {
	var extensionTypes []ExtensionType
	var extensionTypeIndex uint64 = 0
	for {
		if extensionTypeIndex < uint64(len(tlvData)) {
			entryType, err := t.readUint16LE(tlvData, extensionTypeIndex)
			if err != nil {
				return nil, err
			}
			extensionTypes = append(extensionTypes, ExtensionType(entryType))
			entryLength, err := t.readUint16LE(tlvData, extensionTypeIndex+uint64(t.TYPE_SIZE()))
			if err != nil {
				return nil, err
			}
			extensionTypeIndex += t.AddTypeAndLengthToLen2(uint64(entryLength))
		} else {
			break
		}
	}
	return extensionTypes, nil
}

func (t tokenKit2022) GetExtensionData(extension ExtensionType, tlvData []byte) ([]byte, error) {
	var extensionTypeIndex uint64 = 0
	for {
		if t.AddTypeAndLengthToLen2(extensionTypeIndex) <= uint64(len(tlvData)) {
			entryType, err := t.readUint16LE(tlvData, extensionTypeIndex)
			if err != nil {
				return nil, err
			}
			entryLength, err := t.readUint16LE(tlvData, extensionTypeIndex+uint64(t.TYPE_SIZE()))
			if err != nil {
				return nil, err
			}
			var typeIndex = t.AddTypeAndLengthToLen2(extensionTypeIndex)
			if entryType == uint16(extension) {
				return tlvData[typeIndex : typeIndex+uint64(entryLength)], nil
			}
			extensionTypeIndex = typeIndex + uint64(entryLength)
		} else {
			break
		}
	}
	return nil, nil
}

func (t tokenKit2022) GetTypeLen(e ExtensionType) (int, error) {
	switch e {
	case ExtensionTypeUninitialized:
		return 0, nil
	case ExtensionTypeTransferFeeConfig:
		return transfer_fee.TRANSFER_FEE_CONFIG_SIZE, nil
	case ExtensionTypeTransferFeeAmount:
		return t.TRANSFER_FEE_AMOUNT_SIZE(), nil
	case ExtensionTypeMintCloseAuthority:
		return mint_close_authority.MINT_CLOSE_AUTHORITY_SIZE, nil
	case ExtensionTypeConfidentialTransferMint:
		return 97, nil
	case ExtensionTypeConfidentialTransferAccount:
		return 286, nil
	case ExtensionTypeCpiGuard:
		return cpi_guard.CPI_GUARD_SIZE, nil
	case ExtensionTypeDefaultAccountState:
		return default_account_state.DEFAULT_ACCOUNT_STATE_SIZE, nil
	case ExtensionTypeImmutableOwner:
		return immutable_owner.IMMUTABLE_OWNER_SIZE, nil
	case ExtensionTypeMemoTransfer:
		return t.MEMO_TRANSFER_SIZE(), nil
	case ExtensionTypeMetadataPointer:
		return metadata_pointer.METADATA_POINTER_SIZE, nil
	case ExtensionTypeNonTransferable:
		return non_transferable.NON_TRANSFERABLE_SIZE, nil
	case ExtensionTypeInterestBearingConfig:
		return t.INTEREST_BEARING_MINT_CONFIG_STATE_SIZE(), nil
	case ExtensionTypePermanentDelegate:
		return permanent_delegate.PERMANENT_DELEGATE_SIZE, nil
	case ExtensionTypeNonTransferableAccount:
		return non_transferable.NON_TRANSFERABLE_ACCOUNT_SIZE, nil
	case ExtensionTypeTransferHook:
		return transfer_hook.TRANSFER_HOOK_SIZE, nil
	case ExtensionTypeTransferHookAccount:
		return transfer_hook.TRANSFER_HOOK_ACCOUNT_SIZE, nil
	case ExtensionTypeTokenMetadata:
		return 0, fmt.Errorf("cannot get type length for variable extension type:%v", e)
	default:
		return 0, fmt.Errorf("unknown extension type: %v", e)
	}
}

func (t tokenKit2022) IsMintExtension(e ExtensionType) bool {
	switch e {
	case ExtensionTypeTransferFeeConfig:
		fallthrough
	case ExtensionTypeMintCloseAuthority:
		fallthrough
	case ExtensionTypeConfidentialTransferMint:
		fallthrough
	case ExtensionTypeDefaultAccountState:
		fallthrough
	case ExtensionTypeNonTransferable:
		fallthrough
	case ExtensionTypeInterestBearingConfig:
		fallthrough
	case ExtensionTypePermanentDelegate:
		fallthrough
	case ExtensionTypeTransferHook:
		fallthrough
	case ExtensionTypeMetadataPointer:
		fallthrough
	case ExtensionTypeTokenMetadata:
		return true
	case ExtensionTypeUninitialized:
		fallthrough
	case ExtensionTypeTransferFeeAmount:
		fallthrough
	case ExtensionTypeConfidentialTransferAccount:
		fallthrough
	case ExtensionTypeImmutableOwner:
		fallthrough
	case ExtensionTypeMemoTransfer:
		fallthrough
	case ExtensionTypeCpiGuard:
		fallthrough
	case ExtensionTypeNonTransferableAccount:
		fallthrough
	case ExtensionTypeTransferHookAccount:
		return false
	default:
		panic(fmt.Sprintf("Unknown extension type: %v", e))
	}
}

var InvalidAccountSizeErr = errors.New("InvalidAccountSizeErr")
var TokenAccountNotFoundErr = errors.New("TokenAccountNotFoundErr")
var TokenInvalidAccountOwnerErr = errors.New("TokenInvalidAccountOwnerErr")
var TokenInvalidMintErr = errors.New("TokenInvalidMintErr")

type Mint2022 struct {
	Mint
	Address web3.PublicKey // address of the mint
	TlvData []byte         // Additional data for extension
}

func (t tokenKit2022) GetMint(
	ctx context.Context,
	connection *web3.Connection,
	mint, programId web3.PublicKey,
	config web3.GetAccountInfoConfig,
) (*Mint2022, error) {
	_ = ctx
	info, err := connection.GetAccountInfo(mint, config)
	if err != nil {
		return nil, err
	}
	return t.UnpackMint(mint, info, programId)
}

func (t tokenKit2022) ParseMint(data []byte) (*Mint, error) {
	if len(data) < MINT_SIZE {
		return nil, InvalidAccountSizeErr
	}
	return decodeObject[*Mint](data[0:MINT_SIZE])
}

func (t tokenKit2022) UnpackMint(mintAddress web3.PublicKey, info *web3.AccountInfoD, programId web3.PublicKey) (*Mint2022, error) {
	if info == nil {
		return nil, TokenAccountNotFoundErr
	}
	if info.Owner != programId {
		return nil, TokenInvalidAccountOwnerErr
	}
	data := info.Data.Content
	raw, err := t.ParseMint(data)
	if err != nil {
		return nil, err
	}
	var ret = &Mint2022{
		Address: mintAddress,
		Mint:    *raw,
	}
	if len(data) > MINT_SIZE {
		if len(data) <= ACCOUNT_SIZE {
			return nil, InvalidAccountSizeErr
		}
		if len(data) == MULTISIG_SIZE {
			return nil, InvalidAccountSizeErr
		}
		if data[ACCOUNT_SIZE] != 1 /*AccountType.Mint*/ {
			return nil, TokenInvalidMintErr
		}
		ret.TlvData = data[ACCOUNT_SIZE+ACCOUNT_TYPE_SIZE:]
	}
	return ret, nil
}

type TokenAccount2022 struct {
	Account
	Address web3.PublicKey // address of the token
	TlvData []byte         // Additional data for extension
}

func (t tokenKit2022) GetTokenAccount(
	ctx context.Context,
	connection *web3.Connection,
	account, programId web3.PublicKey,
	config web3.GetAccountInfoConfig,
) (*TokenAccount2022, error) {
	_ = ctx
	info, err := connection.GetAccountInfo(account, config)
	if err != nil {
		return nil, err
	}
	return t.UnpackTokenAccount(account, info, programId)
}

func (t tokenKit2022) ParseTokenAccount(data []byte) (*Account, error) {
	if len(data) < ACCOUNT_SIZE {
		return nil, InvalidAccountSizeErr
	}
	return decodeObject[*Account](data[0:ACCOUNT_SIZE])
}

func (t tokenKit2022) UnpackTokenAccount(tokenAccount web3.PublicKey, info *web3.AccountInfoD, programId web3.PublicKey) (*TokenAccount2022, error) {
	if info == nil {
		return nil, TokenAccountNotFoundErr
	}
	if info.Owner != programId {
		return nil, TokenInvalidAccountOwnerErr
	}
	data := info.Data.Content
	raw, err := t.ParseTokenAccount(data)
	if err != nil {
		return nil, err
	}
	var ret = &TokenAccount2022{
		Address: tokenAccount,
		Account: *raw,
	}
	if len(data) > ACCOUNT_SIZE {
		if len(data) == MULTISIG_SIZE {
			return nil, InvalidAccountSizeErr
		}
		if data[ACCOUNT_SIZE] != 2 /*AccountType.Account*/ {
			return nil, TokenInvalidMintErr
		}
		ret.TlvData = data[ACCOUNT_SIZE+ACCOUNT_TYPE_SIZE:]
	}
	return ret, nil
}

type ProgramAccount struct {
	TokenAccount web3.PublicKey
	Info         web3.AccountInfoD
	Owner        web3.PublicKey
}

func (t tokenKit2022) FindTokenAccounts(
	ctx context.Context,
	connection *web3.Connection,
	mint, programId web3.PublicKey,
	option GetProgramAccountsOption,
	commitment web3.Commitment,
) ([]ProgramAccount, error) {
	_ = ctx
	option.Mint = web3.Ref(mint)
	response, err := connection.GetProgramAccounts(programId, web3.GetProgramAccountsConfig{
		Commitment: &commitment,
		DataSlice: &web3.DataSlice{
			Offset: web3.Ref(uint64(0)),
			Length: web3.Ref(uint64(64)), // only get mint and owner
		},
		Filters: GetProgramAccountFilters(option),
	})
	if err != nil {
		return nil, err
	}
	var ret []ProgramAccount
	for _, item := range response {
		owner := web3.NewPublicKeyFromBs(item.Account.Data.Content[32:64])
		associated, err := ata.FindAssociatedTokenAddress(owner, mint, programId)
		if err != nil {
			continue
		}
		if !item.Pubkey.Equals(associated) {
			continue
		}
		ret = append(ret, ProgramAccount{
			Info:         item.Account,
			TokenAccount: item.Pubkey,
			Owner:        owner,
		})
	}
	return ret, nil
}

func (t tokenKit2022) readUint16LE(data []byte, index uint64) (uint16, error) {
	var value uint16
	err := binary.Read(bytes.NewReader(data[index:]), binary.LittleEndian, &value)
	if err != nil {
		return 0, err
	}
	return value, nil
}
