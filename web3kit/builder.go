package web3kit

import (
	"errors"
	"github.com/donutnomad/solana-web3/web3"
	"github.com/gagliardetto/solana-go"
)

type TransactionBuilder struct {
	builder *web3.Transaction
	err     error
}

func NewTransactionBuilder() *TransactionBuilder {
	return &TransactionBuilder{
		builder: &web3.Transaction{},
	}
}

// AddInstructions Support web3.Instruction and solana.Instruction
func (b *TransactionBuilder) AddInstructions(ins ...any) *TransactionBuilder {
	if b.err != nil {
		return b
	}
	for _, ins_ := range ins {
		b.AddInstructions2(ins_, nil)
		if b.err != nil {
			return b
		}
	}
	return b
}

func (b *TransactionBuilder) AddInstructions2(ins any, err error) *TransactionBuilder {
	if b.err != nil {
		return b
	}
	if err != nil {
		b.err = err
		return b
	}
	switch v := ins.(type) {
	case web3.Instruction:
		err := b.builder.AddInstructionAny(v)
		if err != nil {
			b.err = err
			return b
		}
	case solana.Instruction:
		err := b.builder.AddInstructionAny(v)
		if err != nil {
			b.err = err
			return b
		}
	case []web3.Instruction:
		for _, ins_ := range v {
			b.AddInstructions2(ins_, nil)
			if b.err != nil {
				return b
			}
		}
	case []solana.Instruction:
		for _, ins_ := range v {
			b.AddInstructions2(ins_, nil)
			if b.err != nil {
				return b
			}
		}
	default:
		b.err = errors.New("invalid params of ins")
		return b
	}
	return b
}

func (b *TransactionBuilder) AddInsBuilder(builder interface {
	Validate() error
}) *TransactionBuilder {
	if b.err != nil {
		return b
	}
	err := b.builder.AddInsBuilder(builder)
	if err != nil {
		b.err = err
	}
	return b
}

func (b *TransactionBuilder) SetFeePayer(feePayer web3.PublicKey) *TransactionBuilder {
	if b.err != nil {
		return b
	}
	b.builder.SetFeePayer(feePayer)
	return b
}

func (b *TransactionBuilder) SetRecentBlockHash(connection *web3.Connection, commitment web3.Commitment) *TransactionBuilder {
	if b.err != nil {
		return b
	}
	blockHash, err := connection.GetLatestBlockhash(web3.GetLatestBlockhashConfig{
		Commitment: &commitment,
	})
	if err != nil {
		b.err = err
		return b
	}
	b.builder.RecentBlockhash = &blockHash.Blockhash
	b.builder.LastValidBlockHeight = &blockHash.LastValidBlockHeight
	return b
}

func (b *TransactionBuilder) Build() (*web3.Transaction, error) {
	if b.err != nil {
		return nil, b.err
	}
	return b.builder, nil
}
