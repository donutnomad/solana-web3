package web3

import (
	"errors"
	"fmt"
	"github.com/donutnomad/solana-web3/web3/utils"
	solana "github.com/gagliardetto/solana-go"
	"github.com/mr-tron/base58"
	"log"
)

type SignaturePubkeyPair struct {
	Signature Signature
	PublicKey PublicKey
}

// Transaction class
type Transaction struct {
	// signatures for the transaction.  Typically created by invoking the
	// `sign()` method
	signatures   []SignaturePubkeyPair
	instructions []TransactionInstruction
	feePayer     *PublicKey

	//// A
	// the last block chain can advance to before tx is declared expired
	LastValidBlockHeight *uint64
	// A recent transaction id. Must be populated by the caller
	RecentBlockhash *Blockhash

	//// B
	// If this is a nonce transaction this represents the minimum slot from which
	// to evaluate if the nonce has advanced when attempting to confirm the
	// transaction. This protects against a case where the transaction confirmation
	// logic loads the nonce account from an old slot and assumes the mismatch in
	// nonce value implies that the nonce has been advanced.
	MinNonceContextSlot *uint64
	// Optional Nonce information. If populated, transaction will use a durable
	// Nonce hash instead of a recentBlockhash. Must be populated by the caller
	NonceInfo *NonceInformation
}

func NewTransactionWithBlock(recentBlockhash Blockhash, lastValidBlockHeight uint64) *Transaction {
	return &Transaction{
		LastValidBlockHeight: &lastValidBlockHeight,
		RecentBlockhash:      &recentBlockhash,
	}
}

func (t *Transaction) AddInstruction(keys []AccountMeta, programId PublicKey, data []byte) {
	t.instructions = append(t.instructions, TransactionInstruction{
		Keys:      keys,
		ProgramId: programId,
		Data:      data,
	})
}

func (t *Transaction) AddInstruction2(ins TransactionInstruction) {
	t.instructions = append(t.instructions, ins)
}

func (t *Transaction) AddInstruction3(ins solana.Instruction) error {
	var keys []AccountMeta
	for _, item := range ins.Accounts() {
		keys = append(keys, AccountMeta{
			Pubkey:     PublicKey(item.PublicKey),
			IsSigner:   item.IsSigner,
			IsWritable: item.IsWritable,
		})
	}
	data, err := ins.Data()
	if err != nil {
		return err
	}
	t.AddInstruction(keys, PublicKey(ins.ProgramID()), data)
	return nil
}

func (t *Transaction) AddInstruction4(ins Instruction) error {
	var keys []AccountMeta
	for _, item := range ins.Accounts() {
		keys = append(keys, *item)
	}
	data, err := ins.Data()
	if err != nil {
		return err
	}
	t.AddInstruction(keys, ins.ProgramID(), data)
	return nil
}

func (t *Transaction) SetFeePayer(payer PublicKey) {
	t.feePayer = &payer
}

func (t *Transaction) Sign(signers ...Signer) error {
	// dedupe signers
	seen := make(map[string]bool)
	var uniqueSigners []Signer
	for _, signer := range signers {
		k := signer.PublicKey().String()
		if _, ok := seen[k]; ok {
			continue
		} else {
			seen[k] = true
			uniqueSigners = append(uniqueSigners, signer)
		}
	}
	t.signatures = utils.Map(uniqueSigners, func(t Signer) SignaturePubkeyPair {
		return SignaturePubkeyPair{
			PublicKey: t.PublicKey(),
		}
	})
	message, err := t.compile()
	if err != nil {
		return err
	}
	return t._partialSign(message, uniqueSigners...)
}

func (t *Transaction) _partialSign(message *Message, signers ...Signer) error {
	signData := message.Serialize()
	for _, signer := range signers {
		sign, err := signer.Sign(signData)
		if err != nil {
			return err
		}
		err = t._addSignature(signer.PublicKey(), sign)
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *Transaction) _addSignature(pubkey PublicKey, signature Signature) error {
	index := utils.FindIndex(t.signatures, func(pair SignaturePubkeyPair) bool {
		return pubkey.Equals(pair.PublicKey)
	})
	if index < 0 {
		return fmt.Errorf("unknown signer %s", pubkey)
	}
	t.signatures[index].Signature = signature
	return nil
}

func (t *Transaction) compile() (*Message, error) {
	message, err := t.compileMessage()
	if err != nil {
		return nil, err
	}
	signedKeys := message.AccountKeys[0:message.Header.NumRequiredSignatures]

	if len(t.signatures) == len(signedKeys) {
		var valid = true
		for index, sig := range t.signatures {
			if !signedKeys[index].Equals(sig.PublicKey) {
				valid = false
			}
		}
		if valid {
			return message, nil
		}
	}

	t.signatures = utils.Map(signedKeys, func(t PublicKey) SignaturePubkeyPair {
		return SignaturePubkeyPair{
			PublicKey: t,
		}
	})
	return message, nil
}

func (t *Transaction) compileMessage() (*Message, error) {
	var recentBlockhash = t.RecentBlockhash
	var instructions TransactionInstructionSlice = t.instructions

	if t.NonceInfo != nil {
		recentBlockhash = &t.NonceInfo.Nonce
		if !t.instructions[0].Equals(t.NonceInfo.NonceInstruction) {
			instructions = utils.AppendToFirst(instructions, t.NonceInfo.NonceInstruction)
		}
	}
	if recentBlockhash == nil {
		return nil, errors.New("transaction recentBlockhash required")
	}
	if len(instructions) < 1 {
		log.Println("No instructions provided")
	}

	var feePayer PublicKey
	if t.feePayer != nil {
		feePayer = *t.feePayer
	} else if len(t.signatures) > 0 && len(t.signatures) > 0 {
		// Use implicit fee payer
		feePayer = t.signatures[0].PublicKey
	} else {
		return nil, errors.New("transaction fee payer required")
	}

	// Cull duplicate account metas
	var uniqueMap = utils.NewKVMap[string, AccountMeta]()
	for _, m := range instructions.Metas() {
		uniqueMap.InsertOrUpdate(m.Pubkey.String(), m, func(a *AccountMeta) {
			a.WriteOR(m.IsWritable)
			a.SignerOR(m.IsSigner)
		})
	}
	// Sort. Prioritizing first by signer, then by writable
	// Move fee payer to the front
	var uniqueMetas = AccountMetaSlice(uniqueMap.Values()).Sort().MoveToFirst(feePayer)

	// Disallow unknown signers
	{
		for _, signature := range t.signatures {
			idx := uniqueMetas.Find(signature.PublicKey)
			if idx == -1 {
				return nil, fmt.Errorf("unknown signer: %s", signature.PublicKey)
			}
			if !uniqueMetas[idx].IsSigner {
				uniqueMetas[idx].IsSigner = true
				log.Println("Transaction references a signature that is unnecessary, only the fee payer and instruction signer accounts should sign a transaction. This behavior is deprecated and will throw an error in the next major version release.")
			}
		}
	}

	var message = Message{
		Header:          uniqueMetas.ToHeader(),
		AccountKeys:     utils.MergeList(uniqueMetas.ToKeys()),
		RecentBlockhash: *recentBlockhash,
	}

	compiledInstructions, err := utils.MapWithError(instructions, func(ins TransactionInstruction) (CompiledInstruction, error) {
		return utils.DeRef2(newCompiledInstruction(
			utils.FindIndexByValue(message.AccountKeys, ins.ProgramId),
			utils.Map(ins.Keys, func(t AccountMeta) int {
				return utils.FindIndexByValue(message.AccountKeys, t.Pubkey)
			}),
			ins.Data,
		))
	})
	if err != nil {
		return nil, err
	}
	message.Instructions = compiledInstructions
	return &message, nil
}

// Signature The first (payer) Transaction signature
func (t *Transaction) Signature() Signature {
	if len(t.signatures) > 0 {
		return t.signatures[0].Signature
	}
	return [64]byte{}
}

func (t *Transaction) Serialize() ([]byte, error) {
	message, err := t.compile()
	if err != nil {
		return nil, err
	}
	signData := message.Serialize()
	// verifySignatures
	{
		sigErrors := t.getMessageSignednessErrors(signData, true)
		if sigErrors != nil {
			errorMessage := "Signature verification failed."
			if len(sigErrors.Invalid) > 0 {
				errorMessage += fmt.Sprintf("\nInvalid signature for public key%s [`%s`].",
					suffix(len(sigErrors.Invalid)), joinBase58(sigErrors.Invalid))
			}
			if len(sigErrors.Missing) > 0 {
				errorMessage += fmt.Sprintf("\nMissing signature for public key%s [`%s`].",
					suffix(len(sigErrors.Missing)), joinBase58(sigErrors.Missing))
			}
			return nil, errors.New(errorMessage)
		}
	}
	return t.serialize(signData)
}

func PopulateTransaction(message Message, signatures []string) (*Transaction, error) {
	transaction := Transaction{}
	transaction.RecentBlockhash = &message.RecentBlockhash
	if message.Header.NumRequiredSignatures > 0 {
		transaction.SetFeePayer(message.AccountKeys[0])
	}
	for index, signature := range signatures {
		decode, err := base58.Decode(signature)
		if err != nil {
			return nil, err
		}
		transaction.signatures = append(transaction.signatures, SignaturePubkeyPair{
			Signature: solana.SignatureFromBytes(decode),
			PublicKey: message.AccountKeys[index],
		})
	}
	for _, ins := range message.Instructions {
		var keys []AccountMeta
		for _, account := range ins.Accounts {
			pubkey := message.AccountKeys[account]
			issigner := false
			for _, s := range transaction.signatures {
				if s.PublicKey.String() == pubkey.String() {
					issigner = true
					break
				}
			}
			if !issigner {
				issigner = message.IsAccountSigner(int(account))
			}
			keys = append(keys, AccountMeta{
				Pubkey:     pubkey,
				IsSigner:   issigner,
				IsWritable: message.IsAccountWritable(int(account)),
			})
		}
		transaction.AddInstruction(keys, message.AccountKeys[ins.ProgramIdIndex], ins.Data)
	}
	return &transaction, nil
}

func suffix(count int) string {
	if count == 1 {
		return ""
	}
	return "(s)"
}

func joinBase58(keys []PublicKey) string {
	var result string
	for i, key := range keys {
		result += key.Base58()
		if i < len(keys)-1 {
			result += "`, `"
		}
	}
	return result
}

func (t *Transaction) serialize(signData []byte) ([]byte, error) {
	if len(t.signatures) >= 256 {
		return nil, errors.New("signatures count is too long")
	}
	var buf []byte
	utils.EncodeLength(&buf, len(t.signatures))
	for _, sig := range t.signatures {
		buf = append(buf, sig.Signature[:]...)
	}
	buf = append(buf, signData...)
	if len(buf) > PacketDataSize {
		return nil, fmt.Errorf("transaction too large: %d > %d", len(buf), PacketDataSize)
	}
	return buf, nil
}

func (t *Transaction) getMessageSignednessErrors(message []byte, requireAllSignatures bool) *MessageSignednessErrors {
	ret := &MessageSignednessErrors{}
	for _, entry := range t.signatures {
		if entry.Signature.IsZero() {
			if requireAllSignatures {
				ret.addMissing(entry.PublicKey)
			}
		} else {
			if !entry.PublicKey.Verify(message, entry.Signature) {
				ret.addInvalid(entry.PublicKey)
			}
		}
	}
	if ret.hasErrors() {
		return ret
	}
	return nil
}

type ByPriority []AccountMeta

func (a ByPriority) Len() int      { return len(a) }
func (a ByPriority) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

func (a ByPriority) Less(i, j int) bool {
	if a[i].IsSigner != a[j].IsSigner {
		// Signers always come before non-signers
		return a[i].IsSigner
	}
	if a[i].IsWritable != a[j].IsWritable {
		// Writable accounts always come before read-only accounts
		return a[i].IsWritable
	}
	// Otherwise, sort by pubkey, stringwise.
	return a[i].Pubkey.Base58() < a[j].Pubkey.Base58()
}

type MessageSignednessErrors struct {
	Missing []PublicKey
	Invalid []PublicKey
}

func (m *MessageSignednessErrors) addMissing(publicKey PublicKey) {
	m.Missing = append(m.Missing, publicKey)
}

func (m *MessageSignednessErrors) addInvalid(publicKey PublicKey) {
	m.Invalid = append(m.Invalid, publicKey)
}

func (m *MessageSignednessErrors) hasErrors() bool {
	return len(m.Missing) > 0 || len(m.Invalid) > 0
}
