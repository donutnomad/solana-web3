package web3kit

import (
	"github.com/donutnomad/solana-web3/token_metadata"
	"github.com/donutnomad/solana-web3/web3"
)

var TokenMeta = &tokenMeta{}

type tokenMeta struct {
}

func (t tokenMeta) GetCreateIns(name, symbol, uri string,
	mint, updateAuthority, mintAuthority, programId web3.PublicKey,
	additionalMetadata []struct {
		Key   string
		Value string
	},
) (ret []web3.Instruction, err error) {
	defer Recover(&err)

	ins := Must1(token_metadata.NewInitializeInstruction(
		name,
		symbol,
		uri,
		mint,
		updateAuthority,
		mint,
		mintAuthority,
	).SetProgramId(&programId).ValidateAndBuild())
	ret = append(ret, ins)

	if !updateAuthority.IsZero() && len(additionalMetadata) > 0 {
		for _, item := range additionalMetadata {
			ins := Must1(token_metadata.NewUpdateFieldInstruction(token_metadata.NewField_Key(item.Key), item.Value, mint, updateAuthority).SetProgramId(&programId).ValidateAndBuild())
			ret = append(ret, ins)
		}
	}

	return
}
