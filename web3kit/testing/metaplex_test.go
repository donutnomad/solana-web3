package testing

import (
	"context"
	"fmt"
	"github.com/donutnomad/solana-web3/web3"
	"github.com/donutnomad/solana-web3/web3kit"
	"testing"
)

func TestGetTokenName(t *testing.T) {
	var client = web3kit.Must1(web3.NewConnection(web3.Devnet.Url(), nil))
	var mint = web3.MustPublicKey("EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v")

	fmt.Println(web3kit.Token.GetTokenName(context.Background(),
		client,
		mint,
		&web3.CommitmentProcessed,
	))
	// nil

	mint = web3.MustPublicKey("GVTL9CwHurEhXE4WohoNV3KJKvnMNHivq6Ah9kgz8jiA")
	fmt.Println(web3kit.Token.GetTokenName(context.Background(),
		client,
		mint,
		&web3.CommitmentProcessed,
	))
}
