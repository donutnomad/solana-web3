package mpl_token_metadata

import "github.com/donutnomad/solana-web3/web3"

func FindAssociatedAddress(
	mint web3.PublicKey,
) (web3.PublicKey, error) {
	k, _, err := FindAssociatedAddressAndBumpSeed(mint)
	return k, err
}

func FindAssociatedAddressAndBumpSeed(
	mint web3.PublicKey,
) (web3.PublicKey, uint8, error) {
	return web3.FindProgramAddress([][]byte{
		[]byte("metadata"),
		ProgramID[:],
		mint[:],
	}, ProgramID)
}
