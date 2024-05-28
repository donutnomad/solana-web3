package web3

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"filippo.io/edwards25519"
	binary "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/mr-tron/base58"
	"github.com/pkg/errors"
	"math"
)

// MAX_SEED_LENGTH Maximum length of derived pubkey seed
const MAX_SEED_LENGTH = 32

// PUBLIC_KEY_LENGTH Size of public key in bytes
const PUBLIC_KEY_LENGTH = 32

type Signature = solana.Signature

func SignatureFromBytes(input []byte) Signature {
	return solana.SignatureFromBytes(input)
}

func SignatureFromBase58(input string) (Signature, error) {
	return solana.SignatureFromBase58(input)
}

type PublicKey [PUBLIC_KEY_LENGTH]byte

// DefaultPubKey Default public key value. The base58-encoded string representation is all ones (as seen below) The underlying BN number is 32 bytes that are all zeros
var DefaultPubKey = MustPublicKey("11111111111111111111111111111111")

func NewPublicKeyFromBs(bs []byte) (out PublicKey) {
	if len(bs) == 0 {
		return
	}
	copy(out[:], bs[0:min(PUBLIC_KEY_LENGTH, len(bs))])
	return
}

func NewPublicKey(input string) (out PublicKey, _ error) {
	decode, err := base58.Decode(input)
	if err != nil {
		return PublicKey{}, err
	}
	if len(decode) != PUBLIC_KEY_LENGTH {
		return PublicKey{}, errors.New("invalid public key input")
	}
	copy(out[:], decode)
	return
}

func MustPublicKey(input string) PublicKey {
	out, err := NewPublicKey(input)
	if err != nil {
		panic(err)
	}
	return out
}

// CreateWithSeed Derive a public key from another key, a seed, and a program ID.
// The program ID will also serve as the owner of the public key, giving
// it permission to write data to the account.
func CreateWithSeed(fromPublicKey PublicKey, seed string, programId PublicKey) PublicKey {
	var buffer []byte
	buffer = append(buffer, fromPublicKey.Bytes()...)
	buffer = append(buffer, []byte(seed)...)
	buffer = append(buffer, programId.Bytes()...)
	publicKeyBytes := sha256.Sum256(buffer)
	return publicKeyBytes
}

// CreateProgramAddress Derive a program address from seeds and a program ID.
func CreateProgramAddress(seeds [][]byte, programId PublicKey) (PublicKey, error) {
	var buffer []byte
	for _, seed := range seeds {
		if len(seed) > MAX_SEED_LENGTH {
			return DefaultPubKey, errors.New("max seed length exceeded")
		}
		buffer = append(buffer, seed...)
	}
	buffer = append(buffer, programId.Bytes()...)
	buffer = append(buffer, []byte("ProgramDerivedAddress")...)
	publicKeyBytes := sha256.Sum256(buffer)
	if IsOnCurve(publicKeyBytes) {
		return DefaultPubKey, errors.New("invalid seeds, address must fall off the curve")
	}
	return publicKeyBytes, nil
}

// FindProgramAddress Find a valid program address
//
// Valid program addresses must fall off the ed25519 curve.  This function
// iterates a nonce until it finds one that when combined with the seeds
// results in a valid program address.
func FindProgramAddress(seeds [][]byte, programId PublicKey) (address PublicKey, _ uint8, err error) {
	for nonce := uint8(math.MaxUint8); nonce != 0; nonce-- {
		address, err = CreateProgramAddress(append(seeds, []byte{nonce}), programId)
		if err == nil {
			return address, nonce, nil
		}
	}
	err = errors.New("unable to find a viable program address nonce")
	return
}

// IsOnCurve Check that a pubkey is on the ed25519 curve.
func IsOnCurve(pubkey PublicKey) bool {
	_, err := new(edwards25519.Point).SetBytes(pubkey[:])
	return err == nil
}

func UniquePublicKey() PublicKey {
	var bs = make([]byte, 32)
	_, err := rand.Read(bs)
	if err != nil {
		panic(err)
	}
	return NewPublicKeyFromBs(bs)
}

func (p PublicKey) D() solana.PublicKey {
	return solana.PublicKey(p)
}

func (p *PublicKey) D2() *PublicKey {
	if p != nil {
		return p
	}
	return nil
}

func (p *PublicKey) RefBs() *[32]byte {
	if p == nil {
		return nil
	}
	var tmp [32]byte = *p
	return &tmp
}

func (p PublicKey) Equals(publicKey PublicKey) bool {
	return p == publicKey
}

func (p PublicKey) MarshalWithEncoder(encoder *binary.Encoder) error {
	return encoder.WriteBytes(p.Bytes(), false)
}

// Base58 Return the base-58 representation of the public key
func (p PublicKey) Base58() string {
	return base58.Encode(p[:])
}

// Bytes Return the byte array representation of the public key in big endian
func (p PublicKey) Bytes() []byte {
	return p[:]
}

// String Return the base-58 representation of the public key
func (p PublicKey) String() string {
	return p.Base58()
}

func (p PublicKey) IsZero() bool {
	return p == DefaultPubKey
}

func (p PublicKey) Verify(message []byte, signature Signature) bool {
	return ed25519.Verify(p[:], message, signature[:])
}

func (p PublicKey) MarshalJSON() ([]byte, error) {
	var s = p.String()
	return json.Marshal(s)
}

func (p *PublicKey) UnmarshalJSON(data []byte) error {
	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}
	key, err := NewPublicKey(s)
	if err != nil {
		return err
	}
	for idx, item := range key.Bytes() {
		p[idx] = item
	}
	return nil
}
