package impl

import (
	"encoding/binary"
	"fmt"
	"github.com/donutnomad/solana-web3/spl_token_2022"
	"github.com/donutnomad/solana-web3/web3"
	ag_binary "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
	spltoken "github.com/gagliardetto/solana-go/programs/token"
	"log"
	"reflect"
)

type TokenType string

const (
	TokenTypeSOL     TokenType = "SOL"
	TokenTypeSPL     TokenType = "SPL"
	TokenTypeSPL2022 TokenType = "SPL2022"
)

type ParseTransferRet struct {
	From             web3.PublicKey `json:"from"`
	To               web3.PublicKey `json:"to"`
	FromOwner        web3.PublicKey `json:"fromOwner"`
	Token            web3.PublicKey `json:"token"`
	TokenType        TokenType      `json:"tokenType"`
	Amount           uint64         `json:"amount"`
	InstructionIndex int            `json:"instructionIndex"`
}

func (p *ParseTransferRet) IsValid() bool {
	var zero = web3.PublicKey{}
	if p.TokenType == TokenTypeSPL || p.TokenType == TokenTypeSPL2022 {
		return p.Token != zero
	} else {
		return true
	}
}

func metaConverter(input []*web3.AccountMeta) []*solana.AccountMeta {
	var ret = make([]*solana.AccountMeta, len(input))
	for idx, t := range input {
		ret[idx] = &solana.AccountMeta{
			PublicKey:  solana.PublicKey(t.Pubkey),
			IsWritable: t.IsWritable,
			IsSigner:   t.IsSigner,
		}
	}
	return ret
}

func parseSystemToken(data []byte, accounts []*web3.AccountMeta) *ParseTransferRet {
	if len(data) <= 4 {
		return nil
	}
	typId := binary.LittleEndian.Uint32(data)
	if typId == system.Instruction_Transfer {
		data = data[4:]
		var inst = new(system.Transfer)
		if err := ag_binary.NewBinDecoder(data).Decode(inst); err != nil {
			return nil
		}
		inst.AccountMetaSlice = metaConverter(accounts)
		if inst.Lamports == nil {
			return nil
		}
		return &ParseTransferRet{
			From:      web3.PublicKey(inst.GetFundingAccount().PublicKey),
			To:        web3.PublicKey(inst.GetRecipientAccount().PublicKey),
			FromOwner: web3.PublicKey(inst.GetFundingAccount().PublicKey),
			Token:     web3.PublicKey{},
			TokenType: TokenTypeSOL,
			Amount:    *inst.Lamports,
		}
	}
	return nil
}

func parseSplToken(data []byte, accounts []*web3.AccountMeta) *ParseTransferRet {
	instruction, err := spltoken.DecodeInstruction(metaConverter(accounts), data)
	if err != nil {
		return nil
	}
	ins := reflect.ValueOf(instruction.Impl).Elem().Interface()
	if data[0] == spltoken.Instruction_Transfer {
		var c = ins.(spltoken.Transfer)
		if c.Amount == nil {
			return nil
		}
		return &ParseTransferRet{
			From:      web3.PublicKey(c.GetSourceAccount().PublicKey),
			To:        web3.PublicKey(c.GetDestinationAccount().PublicKey),
			FromOwner: web3.PublicKey(c.GetOwnerAccount().PublicKey),
			Token:     web3.PublicKey{}, // update outside
			TokenType: TokenTypeSPL,
			Amount:    *c.Amount,
		}
	} else if data[0] == spltoken.Instruction_TransferChecked {
		var c = ins.(spltoken.TransferChecked)
		if c.Amount == nil {
			return nil
		}
		return &ParseTransferRet{
			From:      web3.PublicKey(c.GetSourceAccount().PublicKey),
			To:        web3.PublicKey(c.GetDestinationAccount().PublicKey),
			FromOwner: web3.PublicKey(c.GetOwnerAccount().PublicKey),
			Token:     web3.PublicKey(c.GetMintAccount().PublicKey),
			TokenType: TokenTypeSPL,
			Amount:    *c.Amount,
		}
	}
	return nil
}

func parseSplToken2022(data []byte, accounts []*web3.AccountMeta) *ParseTransferRet {
	typ := data[0]
	if typ != spl_token_2022.Instruction_Transfer && typ != spl_token_2022.Instruction_TransferChecked {
		return nil
	}
	instruction, err := spl_token_2022.DecodeInstruction(accounts, data)
	if err != nil {
		return nil
	}
	ins := reflect.ValueOf(instruction.Impl).Elem().Interface()
	if typ == spl_token_2022.Instruction_Transfer {
		var c = ins.(spl_token_2022.Transfer)
		return &ParseTransferRet{
			From:      c.GetSourceAccount().Pubkey,
			To:        c.GetDestinationAccount().Pubkey,
			FromOwner: c.GetAuthorityAccount().Pubkey,
			Token:     web3.PublicKey{},
			TokenType: TokenTypeSPL2022,
			Amount:    *c.Amount,
		}
	} else {
		var c = ins.(spl_token_2022.TransferChecked)
		return &ParseTransferRet{
			From:      c.GetSourceAccount().Pubkey,
			To:        c.GetDestinationAccount().Pubkey,
			FromOwner: c.GetAuthorityAccount().Pubkey,
			Token:     c.GetMintAccount().Pubkey,
			TokenType: TokenTypeSPL2022,
			Amount:    *c.Amount,
		}
	}
}

func getAccountsFromTx(keys *web3.MessageAccountKeys, accounts []uint8) ([]*web3.AccountMeta, error) {
	return Map(accounts, func(i int, idx uint8) *web3.AccountMeta {
		pubKey := keys.Get(int(idx))
		var k web3.PublicKey
		if pubKey != nil {
			k = *pubKey
		} else {
			log.Printf("warning, tx: , public key is nil")
		}
		return &web3.AccountMeta{
			Pubkey: k,
		}
	}), nil
}

func ParseTransfer(message web3.VersionedMessage, meta *web3.ConfirmedTransactionMeta, signatures []string) (ret []ParseTransferRet, sig string) {
	msg := message
	if meta.Err != nil {
		return nil, ""
	}
	if len(signatures) == 0 {
		return nil, ""
	}
	sig = signatures[0]
	for insIndex, ins := range msg.CompiledInstructions() {
		if len(ins.Data) == 0 {
			continue
		}
		keys, err := message.GetAccountKeys(web3.GetAccountKeysArgs{
			AccountKeysFromLookups: meta.LoadedAddresses,
		})
		if err != nil {
			log.Println("GetAccountKeys failed; ", err)
			continue
		}
		accounts, err := getAccountsFromTx(keys, ins.Accounts)
		if err != nil {
			log.Println("getAccountsFromTx failed; ", err)
			continue
		}
		program := keys.Get(int(ins.ProgramIdIndex))
		if program == nil {
			continue
		}
		var transfer *ParseTransferRet
		switch *program {
		case web3.SystemProgramID:
			transfer = parseSystemToken(ins.Data, accounts)
		case web3.TokenProgramID:
			transfer = parseSplToken(ins.Data, accounts)
		case web3.TokenProgram2022ID:
			transfer = parseSplToken2022(ins.Data, accounts)
		default:
			continue
		}
		if transfer == nil {
			continue
		}
		transfer.InstructionIndex = insIndex
		if *program != web3.SystemProgramID && transfer.Token.IsZero() {
			for _, item := range meta.PreTokenBalances {
				account := keys.Get(int(item.AccountIndex))
				if account != nil && *account == transfer.From {
					transfer.Token = item.Mint
					break
				}
			}
			if transfer.Token.IsZero() {
				fmt.Printf("ParseTransferRet, accountToMint: %s\n", transfer.From)
				continue
			}
		}
		ret = append(ret, *transfer)
	}
	return
}

type GetMintRet struct {
	TokenAccount      web3.PublicKey
	TokenAccountOwner web3.PublicKey
	Mint              web3.PublicKey
	TokenProgram      web3.PublicKey
}

func GetMint(accountOfToken []web3.PublicKey, connection *web3.Connection, commitment *web3.Commitment) ([]GetMintRet, error) {
	accountOfToken = DeDup(accountOfToken)
	group := Chunk(accountOfToken, 100)
	var ret = make([]GetMintRet, 0, len(accountOfToken))
	for _, publicKeys := range group {
		info, err := connection.GetMultipleAccountsInfo(publicKeys, web3.GetMultipleAccountsConfig{
			Commitment: commitment,
			DataSlice: &web3.DataSlice{
				Offset: web3.Ref[uint64](0),
				Length: web3.Ref[uint64](64),
			},
		})
		if err != nil {
			return nil, err
		}
		for idx, item := range info {
			if item == nil || len(item.Data.Content) < 64 {
				continue
			}
			d := item.Data.Content
			ret = append(ret, GetMintRet{
				TokenAccount:      publicKeys[idx],
				TokenAccountOwner: web3.NewPublicKeyFromBs(d[32:]),
				Mint:              web3.NewPublicKeyFromBs(d[:32]),
				TokenProgram:      item.Owner,
			})
		}
	}
	return ret, nil
}

func Map[T any, O any, TS ~[]T](input TS, mapper func(int, T) O) []O {
	var output = make([]O, len(input))
	for i, data := range input {
		output[i] = mapper(i, data)
	}
	return output
}

func BoolToInt(input bool) int {
	if !input {
		return 0
	}
	return 1
}

func DeDup[T comparable, TS ~[]T](input TS) []T {
	var m = make(map[T]bool, len(input))
	var n = make([]T, 0, len(m))
	for _, item := range input {
		if _, ok := m[item]; !ok {
			m[item] = true
			n = append(n, item)
		}
	}

	return n
}

func Chunk[T any](arr []T, chunkSize int) [][]T {
	var l = len(arr)
	if l == 0 {
		return nil
	}
	if l <= chunkSize {
		return [][]T{arr}
	}
	var chunks = make([][]T, 0, (l/chunkSize)+BoolToInt(l%chunkSize != 0))
	for i := 0; i < l; i += chunkSize {
		chunks = append(chunks, arr[i:min(i+chunkSize, l)])
	}
	return chunks
}
