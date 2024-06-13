package web3

import "github.com/donutnomad/solana-web3/web3/utils"

type ComplexSigner struct {
	PublicKey PublicKey
	signers   []Signer
}

func NewComplexSigner(sig Signer) ComplexSigner {
	cs := ComplexSigner{
		PublicKey: sig.PublicKey(),
		signers:   []Signer{sig},
	}
	return cs
}

func NewComplexSignerMulti(account PublicKey, signers []Signer) ComplexSigner {
	cs := ComplexSigner{
		PublicKey: account,
		signers:   signers,
	}
	return cs
}

func (c ComplexSigner) IsMultiSig() bool {
	return len(c.signers) > 1
}

func (c ComplexSigner) Addresses() []PublicKey {
	if !c.IsMultiSig() {
		return nil
	}
	return utils.Map(c.Signers(), func(t Signer) PublicKey {
		return t.PublicKey()
	})
}

func (c ComplexSigner) Signers() []Signer {
	return c.signers
}

func (c ComplexSigner) GetSigner(key PublicKey) Signer {
	ret, ok := utils.Find(c.Signers(), func(sig Signer) bool {
		return sig.PublicKey().Equals(key)
	})
	if ok {
		return ret
	}
	return nil
}
