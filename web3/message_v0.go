package web3

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/donutnomad/solana-web3/web3/utils"
	"github.com/mr-tron/base58"
	"math"
)

type MessageHeader struct {
	// The number of signatures required for this message to be considered valid. The
	// signatures must match the first `numRequiredSignatures` of `accountKeys`.
	NumRequiredSignatures int `json:"numRequiredSignatures,omitempty"`
	// The last `numReadonlySignedAccounts` of the signed keys are read-only accounts
	NumReadonlySignedAccounts int `json:"numReadonlySignedAccounts,omitempty"`
	// The last `numReadonlySignedAccounts` of the unsigned keys are read-only accounts
	NumReadonlyUnsignedAccounts int `json:"numReadonlyUnsignedAccounts,omitempty"`
}

type Base58Bytes []byte

func (c *Base58Bytes) MarshalJSON() ([]byte, error) {
	var str = base58.Encode(*c)
	return json.Marshal(str)
}

func (c *Base58Bytes) UnmarshalJSON(data []byte) (err error) {
	var str string
	err = json.Unmarshal(data, &str)
	if err != nil {
		return err
	}
	if str == "" {
		return nil
	}
	decode, err := base58.Decode(str)
	if err != nil {
		return err
	}
	*c = decode
	return nil
}

// MessageAddressTableLookup An address table lookup used to load additional accounts
type MessageAddressTableLookup struct {
	AccountKey      PublicKey `json:"accountKey,omitempty"`
	WritableIndexes []uint8   `json:"writableIndexes,omitempty"`
	ReadonlyIndexes []uint8   `json:"readonlyIndexes,omitempty"`
}

type AccountKeysFromLookups = LoadedAddresses

type MessageV0 struct {
	Header               MessageHeader               `json:"header"`
	StaticAccountKeys    []PublicKey                 `json:"accountKeys,omitempty"`
	RecentBlockhash      Blockhash                   `json:"recentBlockhash,omitempty"`
	CompiledInstructions []CompiledInstruction       `json:"instructions,omitempty"`
	AddressTableLookups  []MessageAddressTableLookup `json:"addressTableLookups,omitempty"`
}

type CompileV0Args struct {
	PayerKey                   PublicKey
	Instructions               []TransactionInstruction
	RecentBlockhash            Blockhash
	AddressLookupTableAccounts []AddressLookupTableAccount
}

func NewMessage0(args CompileV0Args) (*MessageV0, error) {
	compiledKeys := NewCompileKeys(args.Instructions, args.PayerKey)
	var addressTableLookups []MessageAddressTableLookup
	var accountKeysFromLookups AccountKeysFromLookups

	for _, lookupTable := range args.AddressLookupTableAccounts {
		addressTableLookup, keys, err := compiledKeys.extractTableLookup(lookupTable)
		if err != nil {
			return nil, err
		}
		if addressTableLookup != nil {
			addressTableLookups = append(addressTableLookups, *addressTableLookup)
			accountKeysFromLookups.Writable = append(accountKeysFromLookups.Writable, keys.Writable...)
			accountKeysFromLookups.Readonly = append(accountKeysFromLookups.Readonly, keys.Readonly...)
		}
	}

	header, staticAccountKeys, err := compiledKeys.getMessageComponents()
	if err != nil {
		return nil, err
	}
	accountKeys := MessageAccountKeys{staticAccountKeys, &accountKeysFromLookups}
	compiledInstructions, err := accountKeys.compileInstructions(args.Instructions)
	if err != nil {
		return nil, err
	}
	return &MessageV0{
		Header:               *header,
		StaticAccountKeys:    staticAccountKeys,
		RecentBlockhash:      args.RecentBlockhash,
		CompiledInstructions: compiledInstructions,
		AddressTableLookups:  addressTableLookups,
	}, nil
}

func (m *MessageV0) Version() TransactionVersion {
	return TransactionVersion0
}

func (m *MessageV0) NumAccountKeysFromLookups() int {
	var count = 0
	for _, item := range m.AddressTableLookups {
		count += len(item.ReadonlyIndexes) + len(item.WritableIndexes)
	}
	return count
}

func (m *MessageV0) IsAccountSigner(index int) bool {
	return index < m.Header.NumRequiredSignatures
}

func (m *MessageV0) IsAccountWritable(index int) bool {
	numSignedAccounts := m.Header.NumRequiredSignatures
	numStaticAccountKeys := len(m.StaticAccountKeys)
	if index >= numStaticAccountKeys {
		var lookupAccountKeysIndex = index - numStaticAccountKeys
		numWritableLookupAccountKeys := utils.SumInt(m.AddressTableLookups, func(o MessageAddressTableLookup) int {
			return len(o.WritableIndexes)
		})
		return lookupAccountKeysIndex < numWritableLookupAccountKeys
	} else if index >= m.Header.NumRequiredSignatures {
		var unsignedAccountIndex = index - numSignedAccounts
		var numUnsignedAccounts = numStaticAccountKeys - numSignedAccounts
		var numWritableUnsignedAccounts = numUnsignedAccounts - m.Header.NumReadonlyUnsignedAccounts
		return unsignedAccountIndex < numWritableUnsignedAccounts
	} else {
		var numWritableSignedAccounts = numSignedAccounts - m.Header.NumReadonlySignedAccounts
		return index < numWritableSignedAccounts
	}
}

type GetAccountKeysArgs struct {
	AccountKeysFromLookups     *AccountKeysFromLookups
	AddressLookupTableAccounts []AddressLookupTableAccount
}

type MessageAccountKeys struct {
	staticAccountKeys      []PublicKey
	accountKeysFromLookups *AccountKeysFromLookups
}

func (m MessageAccountKeys) KeySegments() [][]PublicKey {
	keySegments := [][]PublicKey{m.staticAccountKeys}
	if m.accountKeysFromLookups != nil {
		keySegments = append(keySegments, m.accountKeysFromLookups.Writable)
		keySegments = append(keySegments, m.accountKeysFromLookups.Readonly)
	}
	return keySegments
}

func (m MessageAccountKeys) FlatKeySegments() []PublicKey {
	var ret []PublicKey
	for _, keySegment := range m.KeySegments() {
		for _, key := range keySegment {
			ret = append(ret, key)
		}
	}
	return ret
}

func (m MessageAccountKeys) Get(index int) *PublicKey {
	for _, keySegment := range m.KeySegments() {
		if index < len(keySegment) {
			return &keySegment[index]
		} else {
			index -= len(keySegment)
		}
	}
	return nil
}

func (m MessageAccountKeys) Length() int {
	return len(m.FlatKeySegments())
}

func (m MessageAccountKeys) compileInstructions(instructions []TransactionInstruction) ([]CompiledInstruction, error) {
	var mErr error
	// Bail early if any account indexes would overflow a u8
	if m.Length() > math.MaxUint8+1 {
		return nil, errors.New("account index overflow encountered during compilation")
	}
	keyIndexMap := make(map[string]uint8)
	for index, key := range m.FlatKeySegments() {
		keyIndexMap[key.Base58()] = uint8(index)
	}
	findKeyIndex := func(key PublicKey) uint8 {
		keyIndex, ok := keyIndexMap[key.Base58()]
		if !ok {
			mErr = errors.New("encountered an unknown instruction account key during compilation")
			return 0
		}
		return keyIndex
	}
	var ret = utils.Map(instructions, func(instruction TransactionInstruction) CompiledInstruction {
		return CompiledInstruction{
			ProgramIdIndex: findKeyIndex(instruction.ProgramId),
			Accounts: utils.Map(instruction.Keys, func(meta AccountMeta) uint8 {
				return findKeyIndex(meta.Pubkey)
			}),
			Data: instruction.Data,
		}
	})
	if mErr != nil {
		return nil, mErr
	}
	return ret, nil
}

var MessageVersion0Prefix = byte(1 << 7)

func (m *MessageV0) Serialize() []byte {
	var buf []byte

	// prefix
	buf = append(buf, MessageVersion0Prefix)
	// header
	buf = append(buf, uint8(m.Header.NumRequiredSignatures))
	buf = append(buf, uint8(m.Header.NumReadonlySignedAccounts))
	buf = append(buf, uint8(m.Header.NumReadonlyUnsignedAccounts))
	// staticAccountKeysLength
	utils.EncodeLength(&buf, len(m.StaticAccountKeys))
	// staticAccount
	for _, account := range m.StaticAccountKeys {
		buf = append(buf, account.Bytes()...)
	}
	// recentBlockhash
	decode, err := base58.Decode(m.RecentBlockhash)
	if err != nil {
		panic(err)
	}
	buf = append(buf, decode...)
	// instructions
	{
		// instructionsLength
		utils.EncodeLength(&buf, len(m.CompiledInstructions))
		var bufInner []byte
		for _, instruction := range m.CompiledInstructions {
			// programIdIndex
			bufInner = append(bufInner, instruction.ProgramIdIndex)
			// accountKeyIndexes
			utils.EncodeLength(&bufInner, len(instruction.Accounts))
			bufInner = append(bufInner, instruction.Accounts...)
			// data
			utils.EncodeLength(&bufInner, len(instruction.Data))
			bufInner = append(bufInner, instruction.Data...)
		}
		buf = append(buf, bufInner...)
	}
	// AddressTableLookups
	{
		// addressTableLookupsLength
		utils.EncodeLength(&buf, len(m.AddressTableLookups))
		var bufInner []byte
		for _, lookup := range m.AddressTableLookups {
			// accountKey
			bufInner = append(bufInner, lookup.AccountKey.Bytes()...)
			// encodedWritableIndexesLength
			utils.EncodeLength(&bufInner, len(lookup.WritableIndexes))
			// writableIndexes
			bufInner = append(bufInner, lookup.WritableIndexes...)
			// encodedReadonlyIndexesLength
			utils.EncodeLength(&bufInner, len(lookup.ReadonlyIndexes))
			// readonlyIndexes
			bufInner = append(bufInner, lookup.ReadonlyIndexes...)
		}
		buf = append(buf, bufInner...)
	}
	return buf
}

func (m *MessageV0) Deserialize(data []byte) error {
	buf := bytes.NewBuffer(data)
	prefix, err := buf.ReadByte()
	if err != nil {
		return err
	}
	maskedPrefix := prefix & versionPrefixMask
	if prefix == maskedPrefix {
		return errors.New("expected versioned message but received legacy message")
	}
	version := maskedPrefix
	if version != 0 {
		return fmt.Errorf("expected versioned message with version 0 but found version %d", version)
	}
	var bs [3]byte
	_, err = buf.Read(bs[:])
	if err != nil {
		return err
	}
	header := MessageHeader{
		NumRequiredSignatures:       int(bs[0]),
		NumReadonlySignedAccounts:   int(bs[1]),
		NumReadonlyUnsignedAccounts: int(bs[2]),
	}

	var staticAccountKeys []PublicKey
	staticAccountKeysLength, size, err := utils.DecodeLength(buf.Bytes())
	if err != nil {
		return err
	}
	buf.Next(size)
	for i := 0; i < staticAccountKeysLength; i++ {
		var k [PUBLIC_KEY_LENGTH]byte
		_, err = buf.Read(k[:])
		if err != nil {
			return err
		}
		staticAccountKeys = append(staticAccountKeys, k)
	}

	var k [PUBLIC_KEY_LENGTH]byte
	_, err = buf.Read(k[:])
	if err != nil {
		return err
	}
	recentBlockhash := base58.Encode(k[:])

	instructionCount, size, err := utils.DecodeLength(buf.Bytes())
	buf.Next(size)
	var compiledInstructions []CompiledInstruction
	for i := 0; i < instructionCount; i++ {
		programIdIndex, err := buf.ReadByte()
		if err != nil {
			return err
		}
		accountKeyIndexesLength, size, err := utils.DecodeLength(buf.Bytes())
		if err != nil {
			return err
		}
		buf.Next(size)
		var accountKeyIndexes = make([]byte, accountKeyIndexesLength)
		_, err = buf.Read(accountKeyIndexes[:])
		if err != nil {
			return err
		}
		dataLength, size, err := utils.DecodeLength(buf.Bytes())
		if err != nil {
			return err
		}
		buf.Next(size)
		var insData = make([]byte, dataLength)
		_, err = buf.Read(insData[:])
		if err != nil {
			return err
		}
		compiledInstructions = append(compiledInstructions, CompiledInstruction{
			programIdIndex,
			accountKeyIndexes,
			insData,
		})
	}

	addressTableLookupsCount, size, err := utils.DecodeLength(buf.Bytes())
	buf.Next(size)
	var addressTableLookups []MessageAddressTableLookup
	for i := 0; i < addressTableLookupsCount; i++ {
		var k [PUBLIC_KEY_LENGTH]byte
		_, err = buf.Read(k[:])
		if err != nil {
			return err
		}
		writableIndexesLength, size, err := utils.DecodeLength(buf.Bytes())
		if err != nil {
			return err
		}
		buf.Next(size)
		var writableIndexes = make([]byte, writableIndexesLength)
		_, err = buf.Read(writableIndexes[:])
		if err != nil {
			return err
		}
		readonlyIndexesLength, size, err := utils.DecodeLength(buf.Bytes())
		if err != nil {
			return err
		}
		buf.Next(size)
		var readonlyIndexes = make([]byte, readonlyIndexesLength)
		_, err = buf.Read(readonlyIndexes[:])
		if err != nil {
			return err
		}
		addressTableLookups = append(addressTableLookups, MessageAddressTableLookup{
			k, writableIndexes, readonlyIndexes,
		})
	}

	m.Header = header
	m.StaticAccountKeys = staticAccountKeys
	m.RecentBlockhash = recentBlockhash
	m.CompiledInstructions = compiledInstructions
	m.AddressTableLookups = addressTableLookups

	return nil
}

func (m *MessageV0) GetAccountKeys(args GetAccountKeysArgs) (*MessageAccountKeys, error) {
	var accountKeysFromLookups *AccountKeysFromLookups
	if args.AccountKeysFromLookups != nil {
		if m.NumAccountKeysFromLookups() != len(args.AccountKeysFromLookups.Writable)+len(args.AccountKeysFromLookups.Readonly) {
			return nil, errors.New("failed to get account keys because of a mismatch in the number of account keys from lookups")
		}
		accountKeysFromLookups = args.AccountKeysFromLookups
	} else if len(args.AddressLookupTableAccounts) > 0 {
		lookups, err := m.resolveAddressTableLookups(args.AddressLookupTableAccounts)
		if err != nil {
			return nil, err
		}
		accountKeysFromLookups = lookups
	} else if len(m.AddressTableLookups) > 0 {
		return nil, errors.New("failed to get account keys because address table lookups were not resolved")
	}
	return &MessageAccountKeys{
		m.StaticAccountKeys, accountKeysFromLookups,
	}, nil
}

func (m *MessageV0) resolveAddressTableLookups(addressLookupTableAccounts []AddressLookupTableAccount) (*AccountKeysFromLookups, error) {
	accountKeysFromLookups := AccountKeysFromLookups{}
	for _, tableLookup := range m.AddressTableLookups {
		tableAccount, founded := utils.Find(addressLookupTableAccounts, func(account AddressLookupTableAccount) bool {
			return account.Key.Equals(tableLookup.AccountKey)
		})
		if !founded {
			return nil, fmt.Errorf("failed to find address lookup table account for table key %s", tableLookup.AccountKey)
		}

		for _, index := range tableLookup.WritableIndexes {
			if int(index) < len(tableAccount.State.Addresses) {
				accountKeysFromLookups.Writable = append(accountKeysFromLookups.Writable, PublicKey(tableAccount.State.Addresses[index]))
			} else {
				return nil, fmt.Errorf("failed to find address for index %d in address lookup table %s", index, tableLookup.AccountKey)
			}
		}

		for _, index := range tableLookup.ReadonlyIndexes {
			if int(index) < len(tableAccount.State.Addresses) {
				accountKeysFromLookups.Readonly = append(accountKeysFromLookups.Readonly, PublicKey(tableAccount.State.Addresses[index]))
			} else {
				return nil, fmt.Errorf("failed to find address for index %d in address lookup table %s", index, tableLookup.AccountKey)
			}
		}
	}
	return &accountKeysFromLookups, nil
}
