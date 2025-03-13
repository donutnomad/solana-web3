package binary

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	binaryCompat "github.com/gagliardetto/binary"
	"io"
	"reflect"
	"strings"
)

type TypeID = binaryCompat.TypeID
type Decoder = binaryCompat.Decoder
type Encoder = binaryCompat.Encoder

func NewBorshEncoder(writer io.Writer) *Encoder {
	return binaryCompat.NewBorshEncoder(writer)
}

func NewBorshDecoder(data []byte) *Decoder {
	return binaryCompat.NewBorshDecoder(data)
}

type TypeIDEncoding = binaryCompat.TypeIDEncoding
type VariantType = binaryCompat.VariantType

type VariantDefinition struct {
	typeIDToType   map[TypeID]reflect.Type
	typeIDToName   map[TypeID]string
	typeNameToID   map[string]TypeID
	typeIDEncoding TypeIDEncoding
}

type VariantTypeHash struct {
	Name string
	Hash string
	Type interface{}
}

func NewVariantDefinitionAnchorType(types []VariantTypeHash) (out *VariantDefinition) {
	if len(types) < 0 {
		panic("it's not valid to create a variant definition without any types")
	}
	typeCount := len(types)
	out = &VariantDefinition{
		typeIDEncoding: binaryCompat.AnchorTypeIDEncoding,
		typeIDToType:   make(map[TypeID]reflect.Type, typeCount),
		typeIDToName:   make(map[TypeID]string, typeCount),
		typeNameToID:   make(map[string]TypeID, typeCount),
	}
	for _, typeDef := range types {
		typeID := binaryCompat.TypeIDFromSighash(Sighash([]byte(typeDef.Hash)))
		out.typeIDToType[typeID] = reflect.TypeOf(typeDef.Type)
		out.typeIDToName[typeID] = typeDef.Name
		out.typeNameToID[typeDef.Name] = typeID
	}
	return out
}

func Sighash(data []byte) []byte {
	sum := sha256.Sum256(data)
	return sum[0:8]
}

func (d *VariantDefinition) TypeID(name string) TypeID {
	id, found := d.typeNameToID[name]
	if !found {
		knownNames := make([]string, len(d.typeNameToID))
		i := 0
		for name := range d.typeNameToID {
			knownNames[i] = name
			i++
		}

		panic(fmt.Errorf("trying to use an unknown type name %q, known names are %q", name, strings.Join(knownNames, ", ")))
	}

	return id
}

func (d *VariantDefinition) UnmarshalBinaryVariant(decoder *Decoder, a *BaseVariant) (err error) {
	var def = d
	var typeID TypeID
	switch def.typeIDEncoding {
	case binaryCompat.Uvarint32TypeIDEncoding:
		val, err := decoder.ReadUvarint32()
		if err != nil {
			return fmt.Errorf("uvarint32: unable to read variant type id: %s", err)
		}
		typeID = binaryCompat.TypeIDFromUvarint32(val)
	case binaryCompat.Uint32TypeIDEncoding:
		val, err := decoder.ReadUint32(binary.LittleEndian)
		if err != nil {
			return fmt.Errorf("uint32: unable to read variant type id: %s", err)
		}
		typeID = binaryCompat.TypeIDFromUint32(val, binary.LittleEndian)
	case binaryCompat.Uint8TypeIDEncoding:
		id, err := decoder.ReadUint8()
		if err != nil {
			return fmt.Errorf("uint8: unable to read variant type id: %s", err)
		}
		typeID = binaryCompat.TypeIDFromBytes([]byte{id})
	case binaryCompat.AnchorTypeIDEncoding:
		typeID, err = decoder.ReadTypeID()
		if err != nil {
			return fmt.Errorf("anchor: unable to read variant type id: %s", err)
		}
	case binaryCompat.NoTypeIDEncoding:
		typeID = binaryCompat.NoTypeIDDefaultID
	}

	a.TypeID = typeID

	typeGo := def.typeIDToType[typeID]
	if typeGo == nil {
		return fmt.Errorf("no known type for type %d", typeID)
	}

	if typeGo.Kind() == reflect.Ptr {
		a.Impl = reflect.New(typeGo.Elem()).Interface()
		if err = decoder.Decode(a.Impl); err != nil {
			return fmt.Errorf("unable to decode variant type %d: %s", typeID, err)
		}
	} else {
		// This is not the most optimal way of doing things for "value"
		// types (over "pointer" types) as we always allocate a new pointer
		// element, unmarshal it and then either keep the pointer type or turn
		// it into a value type.
		//
		// However, in non-reflection based code, one would do like this and
		// avoid an `new` memory allocation:
		//
		// ```
		// name := eos.Name("")
		// json.Unmarshal(data, &name)
		// ```
		//
		// This would work without a problem. In reflection code however, I
		// did not find how one can go from `reflect.Zero(typeGo)` (which is
		// the equivalence of doing `name := eos.Name("")`) and take the
		// pointer to it so it can be unmarshalled correctly.
		//
		// A played with various iteration, and nothing got it working. Maybe
		// the next step would be to explore the `unsafe` package and obtain
		// an unsafe pointer and play with it.
		value := reflect.New(typeGo)
		if err = decoder.Decode(value.Interface()); err != nil {
			return fmt.Errorf("unable to decode variant type %d: %s", typeID, err)
		}

		a.Impl = value.Elem().Interface()
	}
	return nil
}

type BaseVariant = binaryCompat.BaseVariant
