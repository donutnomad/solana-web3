package web3

import (
	"encoding/json"
	"github.com/donutnomad/solana-web3/web3/utils"
)

const versionPrefixMask = 0x7f

type VersionedMessage struct {
	Raw any
}

func (c *VersionedMessage) check() {
	if _, ok := c.Raw.(MessageV0); ok {
	} else if _, ok := c.Raw.(Message); ok {
	} else {
		panic("invalid message")
	}
}

func (c *VersionedMessage) Header() MessageHeader {
	c.check()
	if v, ok := c.Raw.(MessageV0); ok {
		return v.Header
	}
	return c.Raw.(Message).Header
}

func (c *VersionedMessage) RecentBlockhash() Blockhash {
	c.check()
	if v, ok := c.Raw.(MessageV0); ok {
		return v.RecentBlockhash
	}
	return c.Raw.(Message).RecentBlockhash
}

func (c *VersionedMessage) CompiledInstructions() []CompiledInstruction {
	c.check()
	if v, ok := c.Raw.(MessageV0); ok {
		return v.CompiledInstructions
	}
	message := c.Raw.(Message)
	return message.CompiledInstructions()
}

func (c *VersionedMessage) TransactionMessage(args *DecompileArgs) (*TransactionMessage, error) {
	c.check()
	return NewTransactionMessageFrom(*c, args)
}

func (c *VersionedMessage) StaticAccountKeys() []PublicKey {
	c.check()
	if v, ok := c.Raw.(MessageV0); ok {
		return v.StaticAccountKeys
	}

	message := c.Raw.(Message)
	return message.StaticAccountKeys()
}

func (c *VersionedMessage) GetAccountKeys(args GetAccountKeysArgs) (*MessageAccountKeys, error) {
	c.check()
	if v, ok := c.Raw.(MessageV0); ok {
		return v.GetAccountKeys(args)
	}
	message := c.Raw.(Message)
	keys := message.GetAccountKeys()
	return &keys, nil
}

func (c *VersionedMessage) Version() TransactionVersion {
	if v, ok := c.Raw.(MessageV0); ok {
		return v.Version()
	}
	return TransactionVersionLegacy
}

func (c *VersionedMessage) Serialize() []byte {
	c.check()
	if v, ok := c.Raw.(MessageV0); ok {
		return v.Serialize()
	}
	message := c.Raw.(Message)
	return message.Serialize()
}

func (c *VersionedMessage) Deserialize(data []byte) error {
	prefix := data[0]
	maskedPrefix := prefix & versionPrefixMask
	if maskedPrefix == prefix {
		var m Message
		err := m.Deserialize(data)
		if err != nil {
			return err
		}
		c.Raw = m
		return nil
	} else {
		var m MessageV0
		err := m.Deserialize(data)
		if err != nil {
			return err
		}
		c.Raw = m
		return nil
	}
}

func (c *VersionedMessage) MarshalJSON() ([]byte, error) {
	c.check()
	if v, ok := c.Raw.(MessageV0); ok {
		data, err := json.Marshal(&v)
		if err != nil {
			return nil, err
		}
		return utils.AppendToFirst(data, MessageVersion0Prefix), nil
	}
	message := c.Raw.(Message)
	return json.Marshal(&message)
}

func (c *VersionedMessage) UnmarshalJSON(data []byte) error {
	prefix := data[0]
	maskedPrefix := prefix & versionPrefixMask
	if maskedPrefix == prefix {
		var v Message
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		v.inflate()
		c.Raw = v
	} else {
		var v MessageV0
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		c.Raw = v
	}
	return nil
}
