package web3

import (
	binary "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go/programs/system"
)

// DurableNonce A durable nonce is a 32 byte value encoded as a base58 string.
type DurableNonce = string

type NonceAccount = system.NonceAccount

// NonceAccountFromAccountData Deserialize NonceAccount from the account data.
func NonceAccountFromAccountData(buffer []byte) (*NonceAccount, error) {
	decoder := binary.NewDecoderWithEncoding(buffer, binary.EncodingBorsh)
	var account system.NonceAccount
	if err := account.UnmarshalWithDecoder(decoder); err != nil {
		return nil, err
	}
	return &account, nil
}
