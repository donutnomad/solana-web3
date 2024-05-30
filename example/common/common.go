package common

import (
	"fmt"
	"github.com/donutnomad/solana-web3/associated_token_account"
	"github.com/donutnomad/solana-web3/web3"
	"github.com/mr-tron/base58"
)

func GetYourPrivateKey() web3.Signer {
	var privateKey = "your private key"
	decode, err := base58.Decode(privateKey)
	if err != nil {
		panic(err)
	}
	return web3.NewSigner(decode)
	//return example.GetEnvPrivateKey()
}

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
	address, seed, err := associated_token_account.FindAssociatedTokenAddressAndBumpSeed(
		owner, mint, programId)
	if err != nil {
		panic(err)
	}
	_ = seed
	return address
}
