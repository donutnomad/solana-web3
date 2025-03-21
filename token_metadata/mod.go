// This code was AUTOGENERATED using the library.
// Please DO NOT EDIT THIS FILE.

package token_metadata

import (
	"bytes"
	"fmt"
	spew "github.com/davecgh/go-spew/spew"
	binary "github.com/donutnomad/solana-web3/binary"
	common "github.com/donutnomad/solana-web3/common"
	solanago "github.com/gagliardetto/solana-go"
	text "github.com/gagliardetto/solana-go/text"
	treeout "github.com/gagliardetto/treeout"
)

var ProgramID common.PublicKey = common.MustPublicKeyFromBase58("11111111111111111111111111111111")

func SetProgramID(pubkey common.PublicKey) {
	ProgramID = pubkey
	if !common.IsZero(ProgramID) {
		solanago.RegisterInstructionDecoder(common.As(ProgramID), registryDecodeInstruction)
	}
}

const ProgramName = "token_metadata"

func init() {
	if !common.IsZero(ProgramID) {
		solanago.RegisterInstructionDecoder(common.As(ProgramID), registryDecodeInstruction)
	}
}

func btou32(b bool) uint32 {
	if b {
		return 1
	}
	return 0
}

var (
	Instruction_Initialize      = binary.TypeID([8]byte{210, 225, 30, 162, 88, 184, 77, 141})
	Instruction_UpdateField     = binary.TypeID([8]byte{221, 233, 49, 45, 181, 202, 220, 200})
	Instruction_RemoveKey       = binary.TypeID([8]byte{234, 18, 32, 56, 89, 141, 37, 181})
	Instruction_UpdateAuthority = binary.TypeID([8]byte{215, 228, 166, 228, 84, 100, 86, 123})
	Instruction_Emit            = binary.TypeID([8]byte{250, 166, 180, 250, 13, 12, 184, 70})
)

var InstructionImplDef = binary.NewVariantDefinitionAnchorType([]binary.VariantTypeHash{
	{
		"initialize", "spl_token_metadata_interface:initialize_account", (*Initialize)(nil),
	},
	{
		"update_field", "spl_token_metadata_interface:updating_field", (*UpdateField)(nil),
	},
	{
		"remove_key", "spl_token_metadata_interface:remove_key_ix", (*RemoveKey)(nil),
	},
	{
		"update_authority", "spl_token_metadata_interface:update_the_authority", (*UpdateAuthority)(nil),
	},
	{
		"emit", "spl_token_metadata_interface:emitter", (*Emit)(nil),
	},
})

// InstructionIDToName returns the name of the instruction given its ID.
func InstructionIDToName(id binary.TypeID) string {
	switch id {
	case Instruction_Initialize:
		return "Initialize"
	case Instruction_UpdateField:
		return "UpdateField"
	case Instruction_RemoveKey:
		return "RemoveKey"
	case Instruction_UpdateAuthority:
		return "UpdateAuthority"
	case Instruction_Emit:
		return "Emit"
	default:
		return ""
	}
}

func registryDecodeInstruction(accounts []*solanago.AccountMeta, data []byte) (interface{}, error) {
	obj, err := DecodeInstruction(common.ConvertMeta(accounts), data)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func DecodeInstruction(accounts []*common.AccountMeta, data []byte) (*Instruction, error) {
	obj := new(Instruction)
	if err := binary.NewBorshDecoder(data).Decode(obj); err != nil {
		return nil, fmt.Errorf("unable to decode instruction: %w", err)
	}
	if v, ok := obj.Impl.(common.AccountsSettable); ok {
		err := v.SetAccounts(accounts)
		if err != nil {
			return nil, fmt.Errorf("unable to set accounts for instruction: %w", err)
		}
	}
	return obj, nil
}

type Instruction struct {
	binary.BaseVariant
	programId *common.PublicKey
	typeIdLen uint8
}

func (obj *Instruction) EncodeToTree(parent treeout.Branches) {
	if enToTree, ok := obj.Impl.(text.EncodableToTree); ok {
		enToTree.EncodeToTree(parent)
	} else {
		parent.Child(spew.Sdump(obj))
	}
}

func (obj *Instruction) ProgramID() common.PublicKey {
	if obj.programId != nil {
		return *obj.programId
	}
	return ProgramID
}

func (obj *Instruction) Accounts() (out []*common.AccountMeta) {
	return obj.Impl.(common.AccountsGettable).GetAccounts()
}

func (obj *Instruction) Data() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := binary.NewBorshEncoder(buf).Encode(obj); err != nil {
		return nil, fmt.Errorf("unable to encode instruction: %w", err)
	}
	return buf.Bytes(), nil
}

func (obj *Instruction) TextEncode(encoder *text.Encoder, option *text.Option) error {
	return encoder.Encode(obj.Impl, option)
}

func (obj *Instruction) UnmarshalWithDecoder(decoder *binary.Decoder) error {
	return InstructionImplDef.UnmarshalBinaryVariant(decoder, &obj.BaseVariant)
}

func (obj *Instruction) MarshalWithEncoder(encoder *binary.Encoder) error {
	err := encoder.WriteBytes(obj.TypeID.Bytes()[:obj.typeIdLen], false)
	if err != nil {
		return fmt.Errorf("unable to write variant type: %w", err)
	}
	return encoder.Encode(obj.Impl)
}
