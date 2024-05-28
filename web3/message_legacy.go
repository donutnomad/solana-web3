package web3

import (
	"bytes"
	"errors"
	"github.com/donutnomad/solana-web3/web3/utils"
	"github.com/mr-tron/base58"
	"sort"
)

type AccountMetaSlice []AccountMeta

func (s AccountMetaSlice) Find(pub PublicKey) int {
	return utils.FindIndex(s, func(meta AccountMeta) bool {
		return meta.Pubkey.Equals(pub)
	})
}

func (s AccountMetaSlice) MoveToFirst(pub PublicKey) AccountMetaSlice {
	ret := utils.RemoveEle(s, func(meta AccountMeta) bool {
		return meta.Pubkey.Equals(pub)
	})
	return utils.AppendToFirst(ret, AccountMeta{
		Pubkey:     pub,
		IsSigner:   true,
		IsWritable: true,
	})
}

func (s AccountMetaSlice) Sort() AccountMetaSlice {
	sort.Sort(ByPriority(s))
	return s
}

func (s AccountMetaSlice) ToHeader() (h MessageHeader) {
	for _, accountMeta := range s {
		if accountMeta.IsSigner {
			h.NumRequiredSignatures += 1
			if !accountMeta.IsWritable {
				h.NumReadonlySignedAccounts += 1
			}
		} else {
			if !accountMeta.IsWritable {
				h.NumReadonlyUnsignedAccounts += 1
			}
		}
	}
	return
}

func (s AccountMetaSlice) ToKeys() (signedKeys []PublicKey, unsignedKeys []PublicKey) {
	for _, item := range s {
		if item.IsSigner {
			signedKeys = append(signedKeys, item.Pubkey)
		} else {
			unsignedKeys = append(unsignedKeys, item.Pubkey)
		}
	}
	return
}

// AccountMeta represents a public key and associated metadata
type AccountMeta struct {
	Pubkey     PublicKey `json:"pubkey"`
	IsSigner   bool      `json:"isSigner"`
	IsWritable bool      `json:"isWritable"`
}

func newAccountMetaWithKey(key PublicKey) AccountMeta {
	return AccountMeta{
		Pubkey: key,
	}
}

func (a *AccountMeta) WRITE() *AccountMeta {
	a.IsWritable = true
	return a
}

func (a *AccountMeta) WriteOR(isWritable bool) {
	a.IsWritable = a.IsWritable || isWritable
}

func (a *AccountMeta) SIGNER() *AccountMeta {
	a.IsSigner = true
	return a
}

func (a *AccountMeta) SignerOR(isSigner bool) {
	a.IsSigner = a.IsSigner || isSigner
}

func (a *AccountMeta) Equals(other AccountMeta) bool {
	return a.Pubkey == other.Pubkey && a.IsSigner == other.IsSigner && a.IsWritable == other.IsWritable
}

type TransactionInstructionSlice []TransactionInstruction

func (s TransactionInstructionSlice) Metas() (out []AccountMeta) {
	for _, ins := range s {
		out = append(out, ins.Keys...)
		out = append(out, newAccountMetaWithKey(ins.ProgramId))
	}
	return
}

// TransactionInstruction represents a transaction instruction
type TransactionInstruction struct {
	// Public keys to include in this transaction
	// Boolean represents whether this pubkey needs to sign the transaction
	Keys []AccountMeta `json:"keys"`
	// Program Id to execute
	ProgramId PublicKey `json:"programId"`
	// Program input
	Data []byte `json:"data"`
}

//type TransactionInstructionJSON struct {
//	Keys []struct {
//		Pubkey     string
//		IsSigner   bool
//		IsWritable bool
//	}
//	ProgramId string
//	Data      []byte
//}

func (ti *TransactionInstruction) Equals(other TransactionInstruction) bool {
	if len(ti.Keys) != len(other.Keys) {
		return false
	}
	if !bytes.Equal(ti.Data, other.Data) {
		return false
	}
	if !ti.ProgramId.Equals(other.ProgramId) {
		return false
	}
	for idx, item := range ti.Keys {
		if !item.Equals(other.Keys[idx]) {
			return false
		}
	}
	return true
}

//// ToJSON converts TransactionInstruction to JSON representation
//func (ti *TransactionInstruction) ToJSON() TransactionInstructionJSON {
//	return TransactionInstructionJSON{
//		Keys: utils.Map(ti.Keys, func(t AccountMeta) struct {
//			Pubkey     string
//			IsSigner   bool
//			IsWritable bool
//		} {
//			return struct {
//				Pubkey     string
//				IsSigner   bool
//				IsWritable bool
//			}{Pubkey: t.Pubkey.ToJSON(), IsSigner: t.IsSigner, IsWritable: t.IsWritable}
//		}),
//		ProgramId: ti.ProgramId.ToJSON(),
//		Data:      ti.Data,
//	}
//}

// NonceInformation Nonce information to be used to build an offline Transaction.
type NonceInformation struct {
	// The current blockhash stored in the nonce
	Nonce Blockhash
	// AdvanceNonceAccount Instruction
	NonceInstruction TransactionInstruction
}

type CompiledInstruction struct {
	//  Index into the transaction keys array indicating the program account that executes this instruction
	ProgramIdIndex uint8 `json:"programIdIndex,omitempty"`
	// Ordered indices into the transaction keys array indicating which accounts to pass to the program
	Accounts []uint8 `json:"accounts,omitempty"`
	// The program input data encoded as base58
	Data Base58Bytes `json:"data,omitempty"`
}

func newCompiledInstruction(programIdIndex int, accounts []int, data Base58Bytes) (*CompiledInstruction, error) {
	if programIdIndex < 0 {
		return nil, errors.New("programIdIndex < 0")
	}
	for _, idx := range accounts {
		if idx < 0 {
			return nil, errors.New("keyIndex < 0")
		}
	}
	return &CompiledInstruction{
		ProgramIdIndex: uint8(programIdIndex),
		Accounts:       utils.MapInt[int, uint8](accounts),
		Data:           data,
	}, nil
}

type Message struct {
	Header          MessageHeader         `json:"header"`
	AccountKeys     []PublicKey           `json:"accountKeys,omitempty"`
	RecentBlockhash Blockhash             `json:"recentBlockhash,omitempty"`
	Instructions    []CompiledInstruction `json:"instructions,omitempty"`

	indexToProgramIds map[int]PublicKey
}

type CompileLegacyArgs struct {
	PayerKey        PublicKey
	Instructions    []TransactionInstruction
	RecentBlockhash Blockhash
}

func NewMessage(args CompileLegacyArgs) (*Message, error) {
	compiledKeys := NewCompileKeys(args.Instructions, args.PayerKey)
	header, staticAccountKeys, err := compiledKeys.getMessageComponents()
	if err != nil {
		return nil, err
	}
	accountKeys := MessageAccountKeys{staticAccountKeys, nil}
	instructions, err := accountKeys.compileInstructions(args.Instructions)
	if err != nil {
		return nil, err
	}
	var indexToProgramIds = make(map[int]PublicKey)
	for _, ix := range instructions {
		indexToProgramIds[int(ix.ProgramIdIndex)] = staticAccountKeys[ix.ProgramIdIndex]
	}
	return &Message{
		Header:            *header,
		AccountKeys:       staticAccountKeys,
		RecentBlockhash:   args.RecentBlockhash,
		Instructions:      instructions,
		indexToProgramIds: indexToProgramIds,
	}, nil
}

func (m *Message) inflate() {
	var indexToProgramIds = make(map[int]PublicKey)
	for _, ix := range m.Instructions {
		indexToProgramIds[int(ix.ProgramIdIndex)] = m.AccountKeys[ix.ProgramIdIndex]
	}
	m.indexToProgramIds = indexToProgramIds
}

func (m *Message) Version() TransactionVersion {
	return TransactionVersionLegacy
}

func (m *Message) StaticAccountKeys() []PublicKey {
	return m.AccountKeys
}

func (m *Message) CompiledInstructions() []CompiledInstruction {
	return m.Instructions
}

func (m *Message) AddressTableLookups() []MessageAddressTableLookup {
	return nil
}

func (m *Message) Serialize() []byte {
	// header
	buf := []byte{
		uint8(m.Header.NumRequiredSignatures),
		uint8(m.Header.NumReadonlySignedAccounts),
		uint8(m.Header.NumReadonlyUnsignedAccounts),
	}

	// accountKeys
	utils.EncodeLength(&buf, len(m.AccountKeys))
	for _, key := range m.AccountKeys {
		buf = append(buf, key[:]...)
	}

	// recentBlockhash
	decode, err := base58.Decode(m.RecentBlockhash)
	if err != nil {
		panic(err)
	}
	buf = append(buf, decode...)

	// instructions
	utils.EncodeLength(&buf, len(m.Instructions))
	for _, instruction := range m.Instructions {
		// programIdIndex
		buf = append(buf, instruction.ProgramIdIndex)
		// accounts
		utils.EncodeLength(&buf, len(instruction.Accounts))
		buf = append(buf, instruction.Accounts...)

		// data
		utils.EncodeLength(&buf, len(instruction.Data))
		buf = append(buf, instruction.Data...)
	}

	return buf
}

func (m *Message) Deserialize(data []byte) error {
	buf := bytes.NewBuffer(data)
	var bs [3]byte
	_, err := buf.Read(bs[:])
	if err != nil {
		return err
	}
	header := MessageHeader{
		NumRequiredSignatures:       int(bs[0]),
		NumReadonlySignedAccounts:   int(bs[1]),
		NumReadonlyUnsignedAccounts: int(bs[2]),
	}

	var accountKeys []PublicKey
	accountKeysLength, size, err := utils.DecodeLength(buf.Bytes())
	if err != nil {
		return err
	}
	buf.Next(size)
	for i := 0; i < accountKeysLength; i++ {
		var k [PUBLIC_KEY_LENGTH]byte
		_, err = buf.Read(k[:])
		if err != nil {
			return err
		}
		accountKeys = append(accountKeys, k)
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

	m.Header = header
	m.AccountKeys = accountKeys
	m.RecentBlockhash = recentBlockhash
	m.Instructions = compiledInstructions

	m.inflate()

	return nil
}

func (m *Message) GetAccountKeys() MessageAccountKeys {
	return MessageAccountKeys{
		m.StaticAccountKeys(), nil,
	}
}

func (m *Message) IsAccountSigner(index int) bool {
	return index < m.Header.NumRequiredSignatures
}

func (m *Message) IsAccountWritable(index int) bool {
	var numSignedAccounts = m.Header.NumRequiredSignatures
	if index >= m.Header.NumRequiredSignatures {
		var unsignedAccountIndex = index - numSignedAccounts
		var numUnsignedAccounts = len(m.AccountKeys) - numSignedAccounts
		var numWritableUnsignedAccounts = numUnsignedAccounts - m.Header.NumReadonlyUnsignedAccounts
		return unsignedAccountIndex < numWritableUnsignedAccounts
	} else {
		var numWritableSignedAccounts = numSignedAccounts - m.Header.NumReadonlySignedAccounts
		return index < numWritableSignedAccounts
	}
}

func (m *Message) IsProgramId(index int) bool {
	_, ok := m.indexToProgramIds[index]
	return ok
}

func (m *Message) ProgramIds() []PublicKey {
	var ret []PublicKey
	for _, value := range m.indexToProgramIds {
		ret = append(ret, value)
	}
	return ret
}

func (m *Message) NonProgramIds() []PublicKey {
	var ret []PublicKey
	for index, key := range m.AccountKeys {
		if !m.IsProgramId(index) {
			ret = append(ret, key)
		}
	}
	return ret
}
