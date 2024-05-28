package web3

import (
	"errors"
	binary "github.com/gagliardetto/binary"
	"math"
)

const LOOKUP_TABLE_META_SIZE = 56

type AddressLookupTableAccount struct {
	Key   PublicKey
	State AddressLookupTableState
}

func (a AddressLookupTableAccount) IsActive() bool {
	return a.State.DeactivationSlot == math.MaxUint64
}

type AddressLookupTableState struct {
	TypeIndex              uint32
	DeactivationSlot       uint64
	LastExtendedSlot       uint64
	LastExtendedStartIndex uint8
	Authority              *PublicKey
	Addresses              []PublicKey
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func (obj *AddressLookupTableState) MarshalWithEncoder(encoder *binary.Encoder) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
			} else {
				panic(r)
			}
		}
	}()
	must(encoder.Encode(obj.TypeIndex))
	must(encoder.Encode(obj.DeactivationSlot))
	must(encoder.Encode(obj.LastExtendedSlot))
	must(encoder.Encode(obj.LastExtendedStartIndex))

	if obj.Authority == nil {
		must(encoder.WriteUint8(0))
		var b [32]byte
		must(encoder.WriteBytes(b[:], false))
	} else {
		must(encoder.WriteUint8(1))
		must(encoder.Encode(obj.Authority))
	}
	// padding
	diff := max(LOOKUP_TABLE_META_SIZE-encoder.Written(), 0)
	for i := 0; i < diff; i++ {
		must(encoder.WriteByte(0))
	}
	// address
	for _, address := range obj.Addresses {
		must(encoder.Encode(&address))
	}
	return nil
}

func (obj *AddressLookupTableState) UnmarshalWithDecoder(decoder *binary.Decoder) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
			} else {
				panic(r)
			}
		}
	}()
	must(decoder.Decode(&obj.TypeIndex))
	must(decoder.Decode(&obj.DeactivationSlot))
	must(decoder.Decode(&obj.LastExtendedSlot))
	must(decoder.Decode(&obj.LastExtendedStartIndex))
	if must2(decoder.ReadUint8()) == 0 {
		must(decoder.SkipBytes(32))
		obj.Authority = nil
	} else {
		must(decoder.Decode(&obj.Authority))
	}
	// padding
	must(decoder.SkipBytes(2))
	serializedAddressesLen := decoder.Len() - LOOKUP_TABLE_META_SIZE
	if serializedAddressesLen < 0 || serializedAddressesLen%32 != 0 {
		return errors.New("lookup table is invalid")
	}
	numSerializedAddresses := serializedAddressesLen / 32
	for i := 0; i < numSerializedAddresses; i++ {
		var address PublicKey
		must(decoder.Decode(&address))
		obj.Addresses = append(obj.Addresses, address)
	}
	return nil
}
