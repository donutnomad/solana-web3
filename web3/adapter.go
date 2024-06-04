package web3

import (
	"errors"
	"github.com/donutnomad/solana-web3/web3/utils"
	"github.com/gagliardetto/solana-go"
	"reflect"
)

var invalidMethod = errors.New("method ValidateAndBuild is invalid")

func validateAndBuild[T any](target any) (*T, error) {
	method := reflect.ValueOf(target).MethodByName("ValidateAndBuild")
	if !method.IsValid() {
		return nil, invalidMethod
	}
	results := method.Call(nil)
	if len(results) != 2 {
		return nil, invalidMethod
	}
	errValue := results[1]
	if !errValue.IsNil() {
		err, ok := errValue.Interface().(error)
		if !ok {
			err = invalidMethod
		}
		return nil, err
	}
	instruction, ok := results[0].Interface().(T)
	if !ok {
		return nil, invalidMethod
	}
	return &instruction, nil
}

func instructionToTransactionInstruction(ins Instruction) (*TransactionInstruction, error) {
	var keys = utils.Map(ins.Accounts(), func(t *AccountMeta) AccountMeta {
		return *t
	})
	data, err := ins.Data()
	if err != nil {
		return nil, err
	}
	return &TransactionInstruction{
		Keys:      keys,
		ProgramId: ins.ProgramID(),
		Data:      data,
	}, nil
}

func instructionToTransactionInstruction2(ins solana.Instruction) (*TransactionInstruction, error) {
	var keys = utils.Map(ins.Accounts(), func(t *solana.AccountMeta) AccountMeta {
		return AccountMeta{
			Pubkey:     PublicKey(t.PublicKey),
			IsSigner:   t.IsSigner,
			IsWritable: t.IsWritable,
		}
	})
	data, err := ins.Data()
	if err != nil {
		return nil, err
	}
	return &TransactionInstruction{
		Keys:      keys,
		ProgramId: PublicKey(ins.ProgramID()),
		Data:      data,
	}, nil
}

func adapterInstructionTo(target any) (*TransactionInstruction, error) {
	{
		ins, err := validateAndBuild[Instruction](target)
		if err == nil {
			return instructionToTransactionInstruction(*ins)
		}
		if !errors.Is(err, invalidMethod) {
			return nil, err
		}
	}
	ins, err := validateAndBuild[solana.Instruction](target)
	if err != nil {
		return nil, err
	}
	return instructionToTransactionInstruction2(*ins)
}
