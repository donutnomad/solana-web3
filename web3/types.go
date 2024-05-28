package web3

type LoadedAddresses struct {
	Writable []PublicKey `json:"writable,omitempty"`
	Readonly []PublicKey `json:"readonly,omitempty"`
}

// LAMPORTS_PER_SOL There are 1-billion lamports in one SOL
const LAMPORTS_PER_SOL uint64 = 1000000000

type Instruction interface {
	ProgramID() PublicKey     // the programID the instruction acts on
	Accounts() []*AccountMeta // returns the list of accounts the instructions requires
	Data() ([]byte, error)    // the binary encoded instructions
}

// Meta intializes a new AccountMeta with the provided pubKey.
func Meta(
	pubKey PublicKey,
) *AccountMeta {
	return &AccountMeta{
		Pubkey: pubKey,
	}
}
