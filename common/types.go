package common

import (
	"github.com/donutnomad/solana-web3/web3"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/text/format"
)

type AccountMeta = web3.AccountMeta
type AccountMetaSlice []*web3.AccountMeta

type PublicKey = web3.PublicKey

func MustPublicKeyFromBase58(input string) PublicKey {
	return web3.MustPublicKey(input)
}

func IsZero(pubkey PublicKey) bool {
	for _, item := range pubkey.Bytes() {
		if item != 0 {
			return false
		}
	}
	return true
}

func As(pubkey PublicKey) solana.PublicKey {
	return solana.PublicKeyFromBytes(pubkey.Bytes())
}

func Meta(
	pubKey PublicKey,
) *web3.AccountMeta {
	return &web3.AccountMeta{
		Pubkey: pubKey,
	}
}

func NewAccountMeta(
	pubKey PublicKey,
	WRITE bool,
	SIGNER bool,
) *AccountMeta {
	return &AccountMeta{
		Pubkey:     pubKey,
		IsWritable: WRITE,
		IsSigner:   SIGNER,
	}
}

func (slice AccountMetaSlice) Get(index int) *AccountMeta {
	if index >= len(slice) {
		return nil
	}
	return slice[index]
}

func (slice *AccountMetaSlice) Append(account *AccountMeta) {
	*slice = append(*slice, account)
}

func (slice *AccountMetaSlice) SetAccounts(accounts []*AccountMeta) error {
	*slice = accounts
	return nil
}

func (slice AccountMetaSlice) GetAccounts() []*AccountMeta {
	out := make([]*AccountMeta, 0, len(slice))
	for i := range slice {
		if slice[i] != nil {
			out = append(out, slice[i])
		}
	}
	return out
}

func FormatMeta(name string, meta *AccountMeta) string {
	var out *solana.AccountMeta
	if meta != nil {
		out = &solana.AccountMeta{
			PublicKey:  solana.PublicKey(meta.Pubkey),
			IsWritable: meta.IsWritable,
			IsSigner:   meta.IsSigner,
		}
	}
	return format.Meta(name, out)
}

func ConvertMeta(input []*solana.AccountMeta) []*AccountMeta {
	var ret []*AccountMeta
	for _, item := range input {
		ret = append(ret, &AccountMeta{
			Pubkey:     PublicKey(item.PublicKey),
			IsSigner:   item.IsSigner,
			IsWritable: item.IsWritable,
		})
	}
	return ret
}

type AccountsSettable interface {
	SetAccounts(accounts []*AccountMeta) error
}

type AccountsGettable interface {
	GetAccounts() (accounts []*AccountMeta)
}
