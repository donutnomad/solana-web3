package irys

import (
	"crypto/rand"
	"crypto/sha512"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/donutnomad/solana-web3/web3"
	"github.com/linkedin/goavro/v2"
)

const schemaStr = `{
    "type": "array",
    "items": {
        "type": "record",
        "name": "Tag",
        "fields": [
            { "name": "name", "type": "string" },
            { "name": "value", "type": "string" }
        ]
    }
}`

type BundlrTx struct {
	signature []byte
	owner     []byte
	target    []byte
	anchor    []byte
	tags      []Tag
	data      []byte
}

type Tag struct {
	Name  string
	Value string
}

func createTransaction(buffer []byte, tags []Tag) BundlrTx {
	return BundlrTx{
		signature: nil,
		owner:     nil,
		target:    nil,
		anchor:    randBs(32),
		tags:      tags,
		data:      buffer,
	}
}

func (tx *BundlrTx) Sign(signer web3.Signer) error {
	tx.owner = signer.PublicKey().Bytes()
	message, err := tx.getMessage()
	if err != nil {
		return err
	}
	sig, err := signer.Sign(message[:])
	if err != nil {
		return err
	}
	tx.signature = sig[:]
	return nil
}

func (tx *BundlrTx) asBytes() (_ []byte, err error) {
	if len(tx.signature) == 0 {
		return nil, errors.New("not sign")
	}
	data := tx.data
	var encodedTags = make([]byte, 0)
	if len(tx.tags) > 0 {
		encodedTags, err = encodeTags(tx.tags)
		if err != nil {
			return nil, err
		}
	}
	sigLength := 64
	pubLength := 32
	length := 2 + sigLength + pubLength + 34 + 16 + len(encodedTags) + len(data)

	var b = make([]byte, 0, length)

	b = binary.LittleEndian.AppendUint16(b, 2 /*sig type*/)
	b = append(b, tx.signature...)
	b = append(b, tx.owner...)
	b = append(b, toLenOption(tx.target))
	b = append(b, tx.target...)
	b = append(b, toLenOption(tx.anchor))
	b = append(b, tx.anchor...)
	b = binary.LittleEndian.AppendUint64(b, uint64(len(tx.tags)))
	b = binary.LittleEndian.AppendUint64(b, uint64(len(encodedTags)))
	b = append(b, encodedTags...)
	b = append(b, data...)
	if len(b) != length {
		panic("invalid length")
	}
	return b, nil
}

func (tx *BundlrTx) getMessage() (_ [48]byte, err error) {
	var encodedTags = make([]byte, 0)
	if len(tx.tags) > 0 {
		encodedTags, err = encodeTags(tx.tags)
		if err != nil {
			return
		}
	}
	var chunks = [][]byte{
		[]byte("dataitem"), // dataitem as buffer
		[]byte("1"),        // one as buffer
		[]byte("2"),        // sig type
		tx.owner,
		tx.target,
		tx.anchor,
		encodedTags,
		tx.data,
	}
	var acc = sha512.Sum384([]byte(fmt.Sprintf("list%d", len(chunks))))
	for _, chunk := range chunks {
		b := sha384(
			sha512.Sum384([]byte(fmt.Sprintf("blob%d", len(chunk)))),
			sha512.Sum384(chunk),
		)
		acc = sha384(acc, b)
	}
	return acc, nil
}

func sha384(arg1 [48]byte, arg2 [48]byte) [48]byte {
	return sha512.Sum384(merge48Byte(arg1, arg2))
}

func merge48Byte(input [48]byte, input2 [48]byte) []byte {
	var ret = make([]byte, 0, 48*2)
	ret = append(ret, input[:]...)
	ret = append(ret, input2[:]...)
	return ret
}

func randBs(length int) []byte {
	var bs = make([]byte, length)
	_, err := rand.Read(bs[:])
	if err != nil {
		panic(err)
	}
	return bs
}

func encodeTags(input []Tag) ([]byte, error) {
	var tags []map[string]any
	for _, tag := range input {
		tags = append(tags, map[string]any{
			"name":  tag.Name,
			"value": tag.Value,
		})
	}
	codec, err := goavro.NewCodec(schemaStr)
	if err != nil {
		return nil, err
	}
	return codec.BinaryFromNative(nil, tags)
}

func toLenOption(i []byte) uint8 {
	if len(i) == 0 {
		return 0
	}
	return 1
}
