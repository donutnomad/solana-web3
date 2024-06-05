package web3kit

import (
	"encoding/binary"
	"github.com/donutnomad/solana-web3/spl_token_2022"
	"github.com/donutnomad/solana-web3/web3"
)

// GetProgramAccountsOption represents options for filtering program accounts.
// When a field is not nil, it indicates that the corresponding condition is enabled.
// Setting multiple fields to non-nil values represents an AND relationship, meaning
// accounts must meet all specified conditions to be included in the results.
type GetProgramAccountsOption struct {
	Mint            *web3.PublicKey
	Owner           *web3.PublicKey
	Amount          *uint64
	Delegate        *web3.PublicKey
	State           *spl_token_2022.AccountState
	IsNative        *uint64
	DelegatedAmount *uint64
	CloseAuthority  *web3.PublicKey
}

func GetProgramAccountFilters(option GetProgramAccountsOption) []web3.GetProgramAccountsFilter {
	var filters = make([]web3.GetProgramAccountsFilter, 0, 8)
	var addF = func(offset uint64, bytes []byte) {
		filters = append(filters, web3.GetProgramAccountsFilter{
			Memcmp: &web3.RPCFilterMemcmp{
				Offset: offset,
				Bytes:  bytes,
			},
		})
	}
	var uint64ToBsLE = func(input uint64) []byte {
		var bs [8]byte
		binary.LittleEndian.PutUint64(bs[:], input)
		return bs[:]
	}
	var offset uint64 = 0
	// mint (0)
	{
		if option.Mint != nil {
			addF(offset, (*option.Mint).Bytes())
		}
		offset += 32
	}
	// owner (32)
	{
		if option.Owner != nil {
			addF(offset, (*option.Owner).Bytes())
		}
		offset += 32
	}
	// amount (64)
	{
		if option.Amount != nil {
			addF(offset, uint64ToBsLE(*option.Amount))
		}
		offset += 8
	}
	// delegate (72)
	{
		offset += 4 // optional
		if option.Delegate != nil {
			addF(offset, (*option.Delegate).Bytes())
		}
		offset += 32
	}
	// state (108)
	{
		if option.State != nil {
			addF(offset, []byte{uint8(*option.State)})
		}
		offset += 1
	}
	// isNative (109)
	{
		offset += 4 // optional
		if option.IsNative != nil {
			addF(offset, uint64ToBsLE(*option.IsNative))
		}
		offset += 8
	}
	// delegated amount (121)
	{
		if option.DelegatedAmount != nil {
			addF(offset, uint64ToBsLE(*option.DelegatedAmount))
		}
		offset += 8
	}
	// close authority (129)
	{
		offset += 4 // optional
		if option.CloseAuthority != nil {
			addF(offset, (*option.CloseAuthority).Bytes())
		}
		offset += 32
	}
	//  (165)
	if offset != spl_token_2022.ACCOUNT_SIZE {
		panic("invalid offset")
	}
	return filters
}
