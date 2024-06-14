package main

import (
	"context"
	"fmt"
	"github.com/donutnomad/solana-web3/example/common"
	"github.com/donutnomad/solana-web3/irys"
	"github.com/donutnomad/solana-web3/test"
	"github.com/donutnomad/solana-web3/web3"
	"github.com/donutnomad/solana-web3/web3kit"
)

func main() {
	var args = web3kit.CreateTokenArgs2022{
		BasicMetadata: web3kit.BasicMetadata{
			Name:        "AA",
			Symbol:      "BB",
			Description: "A TOKEN 2022",
			Image:       test.TestingLogo(),
			Additional: map[string]string{
				"additional": "test",
			},
		},
		Decimals:            9,
		InitialSupply:       web3.Ref(uint64(9)),
		EnableWhitelist:     false,
		EnableBlacklist:     true,
		EnableForceTransfer: true,
		Fee:                 web3.Ref(2 * uint16(100)), // 2%
		MaximumFee:          0,
	}
	var ctx = context.Background()
	var payer = test.GetYourPrivateKey()
	var owner = web3.NewComplexSigner(payer)
	var connection = web3kit.Must1(web3.NewConnection(web3.Devnet.Url(), nil))

	provider := common.NewIrysProvider(irys.DEV, "https://arweave.net/")
	sig, mint := web3kit.Must2(web3kit.Token2022.CreateToken(
		ctx, connection, payer, owner, args, provider, web3kit.MetadataPlex, web3.CommitmentProcessed,
	))
	fmt.Println("Signature:", sig)
	fmt.Println("Mint:", mint)
	fmt.Printf("Explorer: https://explorer.solana.com/address/%s/metadata?cluster=devnet\n", mint.String())
}
