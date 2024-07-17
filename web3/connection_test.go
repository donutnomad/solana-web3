package web3

import (
	"fmt"
	"github.com/gagliardetto/solana-go"
	"testing"
)

func TestConnection(t *testing.T) {
	//conn := Must1(NewConnection("http://127.0.0.1:8899", &ConnectionConfig{
	//	Commitment: &CommitmentProcessed,
	//}))
	conn2 := Must1(NewConnection("https://api.devnet.solana.com", &ConnectionConfig{
		Commitment: &CommitmentProcessed,
	}))
	conn := conn2
	address := MustPublicKey("G7gLJ333oxdVJWXHShSvaMsEkp3MxyzCx2nDxq55h663")
	mint := MustPublicKey("8YLk6K77BVa6gWk5B4Cf6BPM4pR1uVyMBW98uiosbbhp")

	t.Run("GetBalanceAndContext", func(t *testing.T) {
		ret := Must1(conn.GetBalanceAndContext(address, GetBalanceConfig{
			Commitment: &CommitmentFinalized,
		}))
		fmt.Println(ret)
	})

	t.Run("GetFirstAvailableBlock", func(t *testing.T) {
		fmt.Println(Must1(conn.GetFirstAvailableBlock()))
	})

	t.Run("GetBlockTime", func(t *testing.T) {
		fmt.Println(Must1(conn.GetBlockTime(180462)))
	})

	t.Run("GetSupply", func(t *testing.T) {
		ret := Must1(conn.GetSupply(GetSupplyConfig{
			ExcludeNonCirculatingAccountsList: true,
		}))
		fmt.Println(ret)
	})

	t.Run("GetTokenSupply", func(t *testing.T) {
		ret := Must1(conn.GetTokenSupply(mint, nil))
		fmt.Println(ret)
		fmt.Println(ret.Value.UIAmount)
	})

	t.Run("GetTokenAccountsByOwner", func(t *testing.T) {
		ret := Must1(conn.GetTokenAccountsByOwner(address, NewTokenAccountsFilter(mint), GetTokenAccountsByOwnerConfig{}))
		for _, item := range ret.Value {
			fmt.Println(item.Pubkey.String())
			fmt.Println(item.Account.Owner.String())
		}
	})
	t.Run("GetParsedTokenAccountsByOwner", func(t *testing.T) {
		ret := Must1(conn.GetParsedTokenAccountsByOwner(address, NewTokenAccountsFilter(mint), nil))
		for _, item := range ret.Value {
			fmt.Println(item.Pubkey.String())
			fmt.Println(item.Account.Owner.String())
		}
	})

	t.Run("GetLargestAccounts", func(t *testing.T) {
		ret := Must1(conn.GetLargestAccounts(GetLargestAccountsConfig{
			//Filter: &LargestAccountsFilterNonCirculating,
		}))
		fmt.Println(ret)
	})
	t.Run("GetTokenLargestAccounts", func(t *testing.T) {
		ret := Must1(conn.GetTokenLargestAccounts(mint, nil))
		fmt.Println(ret)
	})
	t.Run("GetAccountInfoAndContext", func(t *testing.T) {
		//var i uint64 = 229000
		ret := Must1(conn.GetAccountInfoAndContext(mint, GetAccountInfoConfig{
			//MinContextSlot: 229000,
		}))
		fmt.Println(ret)
	})
	t.Run("GetParsedAccountInfo", func(t *testing.T) {
		ret := Must1(conn.GetParsedAccountInfo(UniquePublicKey(), GetAccountInfoConfig{
			//MinContextSlot: 229000,
		}))
		fmt.Println(ret)
	})
	t.Run("GetAccountInfo", func(t *testing.T) {
		ret := Must1(conn.GetAccountInfo(mint, GetAccountInfoConfig{
			//MinContextSlot: 229000,
		}))
		fmt.Println(ret)
	})
	t.Run("GetMultipleParsedAccounts", func(t *testing.T) {
		ret := Must1(conn.GetMultipleParsedAccounts([]PublicKey{mint, address}, GetMultipleAccountsConfig{
			//MinContextSlot: 229000,
		}))
		fmt.Println(ret)
	})
	t.Run("GetMultipleAccountsInfoAndContext", func(t *testing.T) {
		ret := Must1(conn.GetMultipleAccountsInfoAndContext([]PublicKey{mint, address}, GetMultipleAccountsConfig{
			//MinContextSlot: 229000,
		}))
		fmt.Println(ret)
	})
	t.Run("GetProgramAccounts", func(t *testing.T) {
		ret := Must1(conn.GetProgramAccounts(MustPublicKey("TokenzQdBNbLqP5VEhdkAS6EPFLC1PHnBqCXEpPxuEb"), GetProgramAccountsConfig{
			//MinContextSlot: 229000,
		}))
		_ = ret
		//fmt.Println(ret)
	})
	t.Run("GetParsedProgramAccounts", func(t *testing.T) {
		ret := Must1(conn.GetParsedProgramAccounts(MustPublicKey("TokenzQdBNbLqP5VEhdkAS6EPFLC1PHnBqCXEpPxuEb"), GetParsedProgramAccountsConfig{
			//MinContextSlot: 229000,
		}))
		fmt.Println(ret)
	})
	t.Run("GetBlockHeight", func(t *testing.T) {
		ret := Must1(conn.GetBlockHeight(GetBlockHeightConfig{
			MinContextSlot: 229000,
		}))
		fmt.Println(ret)
	})
	t.Run("GetVoteAccounts", func(t *testing.T) {
		ret := Must1(conn.GetVoteAccounts(nil))
		fmt.Println(ret)
	})
	t.Run("GetSlot", func(t *testing.T) {
		ret := Must1(conn.GetSlot(GetSlotConfig{
			//MinContextSlot: 229000,
		}))
		fmt.Println(ret)
	})
	t.Run("GetClusterNodes", func(t *testing.T) {
		ret := Must1(conn.GetClusterNodes())
		fmt.Println(ret)
	})
	t.Run("GetSlotLeader", func(t *testing.T) {
		ret := Must1(conn.GetSlotLeader(GetSlotLeaderConfig{}))
		fmt.Println(ret)
	})
	t.Run("GetSlotLeaders", func(t *testing.T) {
		ret := Must1(conn.GetSlotLeaders(0, 10))
		fmt.Println(ret)
	})
	t.Run("GetTransactionCount", func(t *testing.T) {
		ret := Must1(conn.GetTransactionCount(GetTransactionCountConfig{
			//MinContextSlot: 229000,
		}))
		fmt.Println(ret)
	})
	t.Run("GetTransactionCount", func(t *testing.T) {
		ret := Must1(conn.GetTotalSupply(nil))
		fmt.Println(ret)
	})
	t.Run("GetInflationGovernor", func(t *testing.T) {
		ret := Must1(conn.GetInflationGovernor(nil))
		fmt.Println(ret)
	})
	t.Run("GetInflationReward", func(t *testing.T) {
		ret := Must1(conn.GetInflationReward(
			[]PublicKey{mint, address},
			GetInflationRewardConfig{
				Commitment: &CommitmentConfirmed,
			},
		))
		fmt.Println(ret)
	})
	t.Run("GetInflationRate", func(t *testing.T) {
		ret := Must1(conn.GetInflationRate())
		fmt.Println(ret)
	})
	t.Run("GetEpochInfo", func(t *testing.T) {
		ret := Must1(conn.GetEpochInfo(GetEpochInfoConfig{}))
		fmt.Println(ret)
	})
	_ = conn2
	t.Run("GetEpochSchedule", func(t *testing.T) {
		ret := Must1(conn.GetEpochSchedule())
		fmt.Println(ret)
	})
	t.Run("GetLeaderSchedule", func(t *testing.T) {
		ret := Must1(conn.GetLeaderSchedule())
		_ = ret
	})
	t.Run("GetMinimumBalanceForRentExemption", func(t *testing.T) {
		ret := Must1(conn.GetMinimumBalanceForRentExemption(100, nil))
		fmt.Println(ret)
	})
	t.Run("GetRecentPerformanceSamples", func(t *testing.T) {
		ret := Must1(conn.GetRecentPerformanceSamples(2))
		fmt.Println(ret)
	})
	t.Run("GetRecentPrioritizationFees", func(t *testing.T) {
		ret := Must1(conn.GetRecentPrioritizationFees(GetRecentPrioritizationFeesConfig{
			LockedWritableAccounts: []PublicKey{address},
		}))
		fmt.Println(ret)
	})
	t.Run("getLatestBlockhash", func(t *testing.T) {
		ret := Must1(conn.GetLatestBlockhash(GetLatestBlockhashConfig{}))
		fmt.Println(ret)
	})
	t.Run("GetVersion", func(t *testing.T) {
		ret := Must1(conn.GetVersion())
		fmt.Println(ret)
	})
	t.Run("IsBlockhashValid", func(t *testing.T) {
		ret := Must1(conn.GetLatestBlockhash(GetLatestBlockhashConfig{}))
		ret2 := Must1(conn.IsBlockhashValid(ret.Blockhash, IsBlockhashValidConfig{}))
		fmt.Println(ret2)
	})
	t.Run("GetGenesisHash", func(t *testing.T) {
		ret := Must1(conn.GetGenesisHash())
		fmt.Println(ret)
	})
	t.Run("GetBlockProduction", func(t *testing.T) {
		ret := Must1(conn.GetBlockProduction(GetBlockProductionConfig{}))
		fmt.Println(ret)
	})
	t.Run("RequestAirdrop", func(t *testing.T) {
		ret := Must1(conn.RequestAirdrop(address, solana.LAMPORTS_PER_SOL))
		fmt.Println(ret)
	})
	t.Run("RequestAirdrop", func(t *testing.T) {
		ret := Must1(conn.GetStakeMinimumDelegation(GetStakeMinimumDelegationConfig{}))
		fmt.Println(ret)
	})
	t.Run("GetBlocks", func(t *testing.T) {
		ret := Must1(conn.GetBlocks(0, Ref[uint64](10), &CommitmentConfirmed))
		fmt.Println(ret)
	})

	t.Run("GetBlockSignatures", func(t *testing.T) {
		ret := Must1(conn.GetBlockSignatures(298300, &CommitmentFinalized))
		fmt.Println(ret)
	})

	t.Run("GetNonceAndContext", func(t *testing.T) {
		ret := Must1(conn.GetNonceAndContext(MustPublicKey("GX9Y7gaf6aHSXoBjpYozdrN89eqNosBjW7P1anU5FvBX"), GetNonceAndContextConfig{}))
		fmt.Println(ret)
	})
	t.Run("GetNonce", func(t *testing.T) {
		ret := Must1(conn.GetNonce(MustPublicKey("GX9Y7gaf6aHSXoBjpYozdrN89eqNosBjW7P1anU5FvBX"), GetNonceAndContextConfig{}))
		fmt.Println(ret)
	})

	t.Run("GetTransaction", func(t *testing.T) {
		ret := Must1(conn.GetTransaction("q4C1xjwCvgiBcSwjctxtu7gz8dASHrZ1ktTWemUb2hWdtZBm4JBG8TjQY635mFYGH91zZTRLBpjhqyTmo6tgvte", GetVersionedTransactionConfig{
			Commitment: &CommitmentFinalized,
		}))
		if ret != nil {
			fmt.Println(ret)
			fmt.Println(ret.Transaction.Message.Version())
		}
	})

	t.Run("GetBlock", func(t *testing.T) {
		response := Must1(conn.GetBlock(Must1(conn.GetSlot(GetSlotConfig{}))-40, GetBlockConfig{}))
		fmt.Println(response)
	})
}

func Must(err error) {
	if err != nil {
		panic(err)
	}
}

func Must1[T any](obj T, err error) T {
	if err != nil {
		panic(err)
	}
	return obj
}
