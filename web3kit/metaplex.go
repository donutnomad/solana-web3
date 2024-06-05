package web3kit

import (
	"context"
	mtm "github.com/donutnomad/solana-web3/mpl_token_metadata"
	"github.com/donutnomad/solana-web3/web3"
)

// MetaPlex Metaplex https://www.metaplex.com/
var MetaPlex = metaPlex{}

type metaPlex struct {
}

func (m metaPlex) ParseMetadata(data []byte) (*mtm.Metadata, error) {
	return decodeObject[*mtm.Metadata](data)
}

func (m metaPlex) GetMetadata(ctx context.Context, connection *web3.Connection, mint web3.PublicKey, commitment *web3.Commitment) (*mtm.Metadata, error) {
	_ = ctx
	var address, err = mtm.FindAssociatedAddress(mint)
	if err != nil {
		return nil, err
	}
	info, err := connection.GetAccountInfo(address, web3.GetAccountInfoConfig{
		Commitment: commitment,
	})
	if err != nil {
		return nil, err
	}
	return m.ParseMetadata(info.Data.Content)
}
