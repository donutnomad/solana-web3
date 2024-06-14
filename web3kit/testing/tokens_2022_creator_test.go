package testing

import (
	"context"
	"fmt"
	"github.com/donutnomad/solana-web3/irys"
	"github.com/donutnomad/solana-web3/test"
	"github.com/donutnomad/solana-web3/web3"
	"github.com/donutnomad/solana-web3/web3kit"
	"testing"
)

type IrysProvider struct {
	endpoint irys.Endpoint
	gateway  string
}

func NewIrysProvider(endpoint irys.Endpoint, gateway string) *IrysProvider {
	return &IrysProvider{endpoint: endpoint, gateway: gateway}
}

func (i IrysProvider) MetadataURI(ctx context.Context, connection *web3.Connection, payer web3.Signer, basic web3kit.BasicMetadata) (string, error) {
	if i.endpoint != irys.DEV && i.endpoint != irys.NODE1 && i.endpoint != irys.NODE2 {
		return "", nil
	}
	return irys.UploadLogoAndMetadata(
		ctx,
		connection,
		irys.NewIrys(i.endpoint),
		payer,
		i.gateway,
		basic.Name,
		basic.Symbol,
		basic.Description,
		basic.Additional,
		basic.Image,
	)
}

func TestCreateToken2022(t *testing.T) {
	var args = web3kit.CreateTokenArgs2022{
		BasicMetadata: web3kit.BasicMetadata{
			Name:        "AA",
			Symbol:      "BB",
			Description: "A TOKEN",
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

	provider := NewIrysProvider(irys.DEV, "https://arweave.net/")
	sig, mint := web3kit.Must2(web3kit.Token2022.CreateToken(
		ctx, connection, payer, owner, args, provider, web3kit.MetadataPlex, web3.CommitmentProcessed,
	))
	fmt.Println("Signature:", sig)
	fmt.Println("Mint:", mint)
	fmt.Printf("Explorer: https://explorer.solana.com/address/%s/metadata?cluster=devnet\n", mint.String())
}
