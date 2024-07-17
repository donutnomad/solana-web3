package web3

import (
	"bytes"
	"sync"
)

var requestBytesPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, 256))
	},
}
