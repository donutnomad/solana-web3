package web3

import (
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"github.com/gagliardetto/solana-go"
	"github.com/mr-tron/base58"
	"os"
)

type Signer interface {
	Sign(data []byte) ([64]byte, error)
	PublicKey() PublicKey
}

var Keypair keypair

type keypair int

// FromSeed Generate a keypair from a 32 byte seed.
func (k keypair) FromSeed(seed []byte) SignerImpl {
	return NewSigner(ed25519.NewKeyFromSeed(seed))
}

// Generate a new random keypair
func (k keypair) Generate() SignerImpl {
	seed := make([]byte, 32)
	_, err := rand.Read(seed[:])
	if err != nil {
		panic(err)
	}
	return NewSigner(ed25519.NewKeyFromSeed(seed))
}

func (k keypair) FromBase58(input string) SignerImpl {
	return must2(k.TryFromBase58(input))
}

func (k keypair) TryFromBase58(input string) (SignerImpl, error) {
	res, err := base58.Decode(input)
	if err != nil {
		return SignerImpl{}, err
	}
	return NewSigner(res), nil
}

func (k keypair) FromFile(path string) SignerImpl {
	return must2(k.TryFromFile(path))
}

func (k keypair) TryFromFile(path string) (SignerImpl, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return SignerImpl{}, fmt.Errorf("read keygen file: %w", err)
	}
	var values []byte
	err = json.Unmarshal(content, &values)
	if err != nil {
		return SignerImpl{}, fmt.Errorf("decode keygen file: %w", err)
	}
	return NewSigner(values), nil
}

type SignerImpl struct {
	publicKey PublicKey
	secretKey []byte
}

func NewSigner(secretKey []byte) SignerImpl {
	return SignerImpl{
		publicKey: publicKey(secretKey),
		secretKey: secretKey,
	}
}

func (s SignerImpl) Sign(data []byte) (out [64]byte, err error) {
	signData, err := ed25519.PrivateKey(s.secretKey).Sign(rand.Reader, data, crypto.Hash(0))
	if err != nil {
		return
	}
	copy(out[:], signData)
	return
}

func (s SignerImpl) PublicKey() PublicKey {
	return s.publicKey
}

func (s SignerImpl) PrivateKey() solana.Data {
	return solana.Data{
		Content:  s.secretKey,
		Encoding: solana.EncodingBase58,
	}
}

func must2[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func publicKey(secretKey []byte) (out PublicKey) {
	pub := ed25519.PrivateKey(secretKey).Public().(ed25519.PublicKey)
	copy(out[:], pub)
	return
}
