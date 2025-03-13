package common

import (
	"context"
	"fmt"
	"github.com/donutnomad/solana-web3/associated_token_account"
	"github.com/donutnomad/solana-web3/irys"
	"github.com/donutnomad/solana-web3/web3"
	"github.com/donutnomad/solana-web3/web3kit"
)

// LamportsToSol Approximately convert fractional native tokens (lamports) into native tokens (SOL)
func LamportsToSol(lamports uint64) float64 {
	return float64(lamports) / float64(web3.LAMPORTS_PER_SOL)
}

// SolToLamports Approximately convert native tokens (SOL) into fractional native tokens (lamports)
func SolToLamports(sol float64) uint64 {
	return uint64(sol * float64(web3.LAMPORTS_PER_SOL))
}

func LamportsToString(lamports uint64) string {
	return fmt.Sprintf("%.9f", LamportsToSol(lamports))
}

func MustGetAssociatedTokenAddress(owner, mint web3.PublicKey, programId web3.PublicKey) web3.PublicKey {
	address, err := associated_token_account.FindAssociatedTokenAddress(
		owner, mint, programId)
	if err != nil {
		panic(err)
	}
	return address
}

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

type StubProvider struct {
}

func (i StubProvider) MetadataURI(ctx context.Context, connection *web3.Connection, payer web3.Signer, basic web3kit.BasicMetadata) (string, error) {
	return "", nil
}
