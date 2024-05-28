package web3

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/donutnomad/solana-web3/web3/utils"
	"io"
)

type TransactionVersion int

func (c *TransactionVersion) UnmarshalJSON(data []byte) (err error) {
	var val interface{}
	if err := json.Unmarshal(data, &val); err != nil {
		return err
	}
	switch v := val.(type) {
	case string:
		*c = TransactionVersionLegacy
	case float64:
		*c = TransactionVersion(int(v))
	default:
		return fmt.Errorf("unexpected type %T", v)
	}
	return nil
}

const (
	TransactionVersionLegacy TransactionVersion = -1
	TransactionVersion0      TransactionVersion = 0
)

type VersionedTransaction struct {
	Signatures [][64]byte
	Message    VersionedMessage
}

func NewVersionedTransaction(message VersionedMessage, signatures [][64]byte) (VersionedTransaction, error) {
	var ret = VersionedTransaction{
		Message: message,
	}
	if len(signatures) > 0 {
		if len(signatures) != message.Header().NumRequiredSignatures {
			return ret, errors.New("expected signatures length to be equal to the number of required signatures")
		}
		ret.Signatures = signatures
	} else {
		var defaultSignatures = make([][64]byte, 0)
		for i := 0; i < message.Header().NumRequiredSignatures; i++ {
			defaultSignatures = append(defaultSignatures, [64]byte{})
		}
		ret.Signatures = defaultSignatures
	}
	return ret, nil
}

func (t *VersionedTransaction) Version() TransactionVersion {
	return TransactionVersionLegacy
}

func (t *VersionedTransaction) Sign(signers ...Signer) error {
	messageData := t.Message.Serialize()
	signerPubkeys := t.Message.StaticAccountKeys()[0:t.Message.Header().NumRequiredSignatures]
	for _, signer := range signers {
		signerIndex := utils.FindIndex(signerPubkeys, func(pubkey PublicKey) bool {
			return pubkey.Equals(signer.PublicKey())
		})
		if signerIndex < 0 {
			return fmt.Errorf("cannot sign with non signer key %s", signer.PublicKey())
		}
		ret, err := signer.Sign(messageData)
		if err != nil {
			return err
		}
		t.Signatures[signerIndex] = ret
	}
	return nil
}

func (t *VersionedTransaction) AddSignature(publicKey PublicKey, signature [64]byte) {
	signerPubkeys := t.Message.StaticAccountKeys()[0:t.Message.Header().NumRequiredSignatures]
	signerIndex := utils.FindIndex(signerPubkeys, func(pubkey PublicKey) bool {
		return pubkey.Equals(publicKey)
	})
	if signerIndex < 0 {
		panic(fmt.Sprintf("can not add signature; %s is not required to sign this transaction", publicKey))
	}
	t.Signatures[signerIndex] = signature
}

func (t *VersionedTransaction) Serialize() []byte {
	serializedMessage := t.Message.Serialize()
	var signatureCount []byte
	utils.EncodeLength(&signatureCount, len(t.Signatures))

	output := make([]byte, 0, len(signatureCount)+len(signatureCount)*64+len(serializedMessage))
	output = append(output, signatureCount...)
	for _, sig := range t.Signatures {
		output = append(output, sig[:]...)
	}
	output = append(output, serializedMessage...)
	return output
}

func (t *VersionedTransaction) Deserialize(data []byte) error {
	signaturesLength, size, err := utils.DecodeLength(data)
	if err != nil {
		return err
	}
	data = data[size:]
	if len(data) < 64*signaturesLength {
		return io.EOF
	}
	var signatures [][64]byte
	for i := 0; i < signaturesLength; i++ {
		var sig [64]byte
		copy(sig[:], data[0:SignatureLengthInBytes])
		signatures = append(signatures, sig)
		data = data[SignatureLengthInBytes:]
	}
	var v VersionedMessage
	if err = v.Deserialize(data); err != nil {
		return err
	}
	t.Message = v
	t.Signatures = signatures
	return nil
}
