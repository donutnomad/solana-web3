package web3kit

import "errors"

var InvalidAccountSizeErr = errors.New("InvalidAccountSizeErr")
var TokenAccountNotFoundErr = errors.New("TokenAccountNotFoundErr")
var TokenInvalidAccountOwnerErr = errors.New("TokenInvalidAccountOwnerErr")
var TokenInvalidMintErr = errors.New("TokenInvalidMintErr")
