package web3

import (
	"github.com/donutnomad/solana-web3/web3/utils"
	"github.com/pkg/errors"
)

type CompiledKeyMeta struct {
	IsSigner   bool
	IsWritable bool
	IsInvoked  bool
}

type KeyMetaMap = map[string]*CompiledKeyMeta

type CompiledKeys struct {
	Payer      PublicKey
	KeyMetaMap KeyMetaMap
}

func NewCompileKeys(instructions []TransactionInstruction, payer PublicKey) CompiledKeys {
	keyMetaMap := make(KeyMetaMap)
	getOrInsertDefault := func(pubkey PublicKey) *CompiledKeyMeta {
		address := pubkey.Base58()
		keyMeta, ok := keyMetaMap[address]
		if !ok {
			keyMeta = &CompiledKeyMeta{
				IsSigner:   false,
				IsWritable: false,
				IsInvoked:  false,
			}
			keyMetaMap[address] = keyMeta
		}
		return keyMeta
	}

	payerKeyMeta := getOrInsertDefault(payer)
	payerKeyMeta.IsSigner = true
	payerKeyMeta.IsWritable = true

	for _, ix := range instructions {
		getOrInsertDefault(ix.ProgramId).IsInvoked = true
		for _, accountMeta := range ix.Keys {
			keyMeta := getOrInsertDefault(accountMeta.Pubkey)
			if keyMeta.IsSigner == false {
				keyMeta.IsSigner = accountMeta.IsSigner
			}
			if keyMeta.IsWritable == false {
				keyMeta.IsWritable = accountMeta.IsWritable
			}
		}
	}

	return CompiledKeys{
		payer,
		keyMetaMap,
	}
}

func (c CompiledKeys) getMessageComponents() (*MessageHeader, []PublicKey, error) {
	if len(c.KeyMetaMap) > 256 {
		return nil, nil, errors.New("max static account keys length exceeded")
	}
	var writableSigners []PublicKey
	var readonlySigners []PublicKey
	var writableNonSigners []PublicKey
	var readonlyNonSigners []PublicKey
	for key_, value := range c.KeyMetaMap {
		key := MustPublicKey(key_)
		if value.IsSigner && value.IsWritable {
			writableSigners = append(writableSigners, key)
		}
		if value.IsSigner && !value.IsWritable {
			readonlySigners = append(readonlySigners, key)
		}
		if !value.IsSigner && value.IsWritable {
			writableNonSigners = append(writableNonSigners, key)
		}
		if !value.IsSigner && !value.IsWritable {
			readonlyNonSigners = append(readonlyNonSigners, key)
		}
	}

	if len(writableSigners) <= 0 {
		return nil, nil, errors.New("expected at least one writable signer key")
	}
	if !writableSigners[0].Equals(c.Payer) {
		return nil, nil, errors.New("expected first writable signer key to be the fee payer")
	}
	var accounts []PublicKey
	accounts = append(accounts, writableSigners...)
	accounts = append(accounts, readonlySigners...)
	accounts = append(accounts, writableNonSigners...)
	accounts = append(accounts, readonlyNonSigners...)
	return &MessageHeader{
		NumRequiredSignatures:       len(writableSigners) + len(readonlySigners),
		NumReadonlySignedAccounts:   len(readonlySigners),
		NumReadonlyUnsignedAccounts: len(readonlyNonSigners),
	}, accounts, nil
}

func (c CompiledKeys) extractTableLookup(lookupTable AddressLookupTableAccount) (*MessageAddressTableLookup, *AccountKeysFromLookups, error) {
	writableIndexes, drainedWritableKeys, err := c.drainKeysFoundInLookupTable(lookupTable.State.Addresses, func(keyMeta *CompiledKeyMeta) bool {
		return !keyMeta.IsSigner && !keyMeta.IsInvoked && keyMeta.IsWritable
	})
	if err != nil {
		return nil, nil, err
	}
	readonlyIndexes, drainedReadonlyKeys, err := c.drainKeysFoundInLookupTable(lookupTable.State.Addresses, func(keyMeta *CompiledKeyMeta) bool {
		return !keyMeta.IsSigner && !keyMeta.IsInvoked && !keyMeta.IsWritable
	})
	// Don't extract lookup if no keys were found
	if len(writableIndexes) == 0 && len(readonlyIndexes) == 0 {
		return nil, nil, nil
	}
	return &MessageAddressTableLookup{
			AccountKey:      lookupTable.Key,
			WritableIndexes: writableIndexes,
			ReadonlyIndexes: readonlyIndexes,
		}, &AccountKeysFromLookups{
			Writable: drainedWritableKeys,
			Readonly: drainedReadonlyKeys,
		}, nil
}

func (c CompiledKeys) drainKeysFoundInLookupTable(lookupTableEntries []PublicKey, keyMetaFilter func(keyMeta *CompiledKeyMeta) bool) (lookupTableIndexes []uint8, drainedKeys []PublicKey, err error) {
	for address, keyMeta := range c.KeyMetaMap {
		if keyMetaFilter(keyMeta) {
			key := MustPublicKey(address)
			lookupTableIndex := utils.FindIndex(lookupTableEntries, func(entry PublicKey) bool {
				return PublicKey(entry).Equals(key)
			})
			if lookupTableIndex >= 0 {
				if lookupTableIndex > 256 {
					return nil, nil, errors.New("Max lookup table index exceeded")
				}
				lookupTableIndexes = append(lookupTableIndexes, uint8(lookupTableIndex))
				drainedKeys = append(drainedKeys, key)
				delete(c.KeyMetaMap, address)
			}
		}
	}
	return lookupTableIndexes, drainedKeys, nil
}
