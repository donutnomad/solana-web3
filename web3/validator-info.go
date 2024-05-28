package web3

import (
	"bytes"
	"encoding/json"
	"github.com/donutnomad/solana-web3/web3/utils"
	binary "github.com/gagliardetto/binary"
)

var VALIDATOR_INFO_KEY = MustPublicKey(
	"Va1idator1nfo111111111111111111111111111111",
)

type ValidatorInfo struct {
	Key  PublicKey
	Info struct {
		/** validator name */
		Name string `json:"name,omitempty"`
		/** optional, validator website */
		Website *string `json:"website,omitempty"`
		/** optional, extra information the validator chose to share */
		Details *string `json:"details,omitempty"`
		/** optional, used to identify validators on keybase.io */
		KeybaseUsername *string `json:"keybaseUsername,omitempty"`
	}
}

// ValidatorInfoFromConfigData Deserialize ValidatorInfo from the config account data. Exactly two config
// keys are required in the data.
func ValidatorInfoFromConfigData(buffer []byte) (*ValidatorInfo, error) {
	buf := bytes.NewBuffer(buffer)
	configKeyCount, size, err := utils.DecodeLength(buf.Bytes())
	if err != nil {
		return nil, err
	}
	buf.Next(size)
	if configKeyCount != 2 {
		return nil, nil
	}

	type ConfigKey struct {
		publicKey PublicKey
		isSigner  bool
	}
	var configKeys []ConfigKey
	for i := 0; i < 2; i++ {
		var publicKey [PUBLIC_KEY_LENGTH]byte
		if _, err = buf.Read(publicKey[:]); err != nil {
			return nil, err
		}
		b, err := buf.ReadByte()
		if err != nil {
			return nil, err
		}
		configKeys = append(configKeys, ConfigKey{
			publicKey: publicKey,
			isSigner:  b == 1,
		})
	}

	if configKeys[0].publicKey.Equals(VALIDATOR_INFO_KEY) {
		if configKeys[1].isSigner {
			out, err := binary.NewBorshDecoder(buf.Bytes()).ReadRustString()
			if err != nil {
				return nil, err
			}
			var ret ValidatorInfo
			ret.Key = configKeys[1].publicKey
			if err = json.Unmarshal([]byte(out), &ret.Info); err != nil {
				return nil, err
			}
			return &ret, nil
		}
	}
	return nil, nil
}
