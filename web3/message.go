package web3

import (
	"errors"
	"fmt"
)

type TransactionMessage struct {
	payerKey        PublicKey
	instructions    []TransactionInstruction
	recentBlockhash Blockhash
}

type DecompileArgs struct {
	AccountKeysFromLookups     *AccountKeysFromLookups
	AddressLookupTableAccounts []AddressLookupTableAccount
}

func NewTransactionMessageFrom(message VersionedMessage, args *DecompileArgs) (*TransactionMessage, error) {
	header := message.Header()
	compiledInstructions := message.CompiledInstructions()
	recentBlockhash := message.RecentBlockhash()
	numRequiredSignatures := header.NumRequiredSignatures
	numReadonlySignedAccounts := header.NumReadonlySignedAccounts
	numReadonlyUnsignedAccounts := header.NumReadonlyUnsignedAccounts

	numWritableSignedAccounts := numRequiredSignatures - numReadonlySignedAccounts
	if numWritableSignedAccounts <= 0 {
		return nil, errors.New("message header is invalid")
	}
	var numWritableUnsignedAccounts = len(message.StaticAccountKeys()) -
		numRequiredSignatures -
		numReadonlyUnsignedAccounts
	if numWritableUnsignedAccounts < 0 {
		return nil, errors.New("message header is invalid")
	}
	var args1 GetAccountKeysArgs
	if args != nil {
		args1.AccountKeysFromLookups = args.AccountKeysFromLookups
		args1.AddressLookupTableAccounts = args.AddressLookupTableAccounts
	}
	accountKeys, err := message.GetAccountKeys(args1)
	if err != nil {
		return nil, err
	}
	payerKey := accountKeys.Get(0)
	if payerKey == nil {
		return nil, errors.New("failed to decompile message because no account keys were found")
	}

	var instructions []TransactionInstruction
	for _, compiledIx := range compiledInstructions {
		var keys []AccountMeta

		for _, keyIndex_ := range compiledIx.Accounts {
			keyIndex := int(keyIndex_)
			pubkey := accountKeys.Get(keyIndex)
			if pubkey == nil {
				return nil, fmt.Errorf("failed to find key for account key index %d", keyIndex)
			}
			isSigner := keyIndex < numRequiredSignatures
			isWritable := false
			if isSigner {
				isWritable = keyIndex < numWritableSignedAccounts
			} else if keyIndex < len(accountKeys.staticAccountKeys) {
				isWritable = keyIndex-numRequiredSignatures < numWritableUnsignedAccounts
			} else {
				// accountKeysFromLookups cannot be undefined because we already found a pubkey for this index above
				isWritable = keyIndex-len(accountKeys.staticAccountKeys) < len(accountKeys.accountKeysFromLookups.Writable)
			}

			keys = append(keys, AccountMeta{
				Pubkey:     *pubkey,
				IsSigner:   keyIndex < header.NumRequiredSignatures,
				IsWritable: isWritable,
			})
		}

		programId := accountKeys.Get(int(compiledIx.ProgramIdIndex))
		if programId == nil {
			return nil, fmt.Errorf("failed to find program id for program id index %d", compiledIx.ProgramIdIndex)
		}

		instructions = append(instructions, TransactionInstruction{
			ProgramId: *programId,
			Data:      compiledIx.Data,
			Keys:      keys,
		})
	}

	return &TransactionMessage{
		payerKey:        *payerKey,
		instructions:    instructions,
		recentBlockhash: recentBlockhash,
	}, nil

}

func (m TransactionMessage) Instructions() []TransactionInstruction {
	return m.instructions
}

func (m TransactionMessage) CompileToLegacyMessage() (*Message, error) {
	return NewMessage(CompileLegacyArgs{
		PayerKey:        m.payerKey,
		RecentBlockhash: m.recentBlockhash,
		Instructions:    m.instructions,
	})
}

func (m TransactionMessage) CompileToV0Message(addressLookupTableAccounts []AddressLookupTableAccount) (*MessageV0, error) {
	return NewMessage0(CompileV0Args{
		PayerKey:                   m.payerKey,
		RecentBlockhash:            m.recentBlockhash,
		Instructions:               m.instructions,
		AddressLookupTableAccounts: addressLookupTableAccounts,
	})
}
