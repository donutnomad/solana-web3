package web3

import (
	"encoding/json"
	"fmt"
	"strings"
)

type SendTransactionError struct {
	Logs    []string
	Message string
}

func NewSendTransactionError(message string, logs []string) SendTransactionError {
	return SendTransactionError{
		logs,
		message,
	}
}

func (e SendTransactionError) Error() string {
	return e.Message + "\n" + strings.Join(e.Logs, "\n")
}

type RpcResponseError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type SolanaJSONRPCError struct {
	Err           RpcResponseError
	CustomMessage string
}

func (e SolanaJSONRPCError) Code() int {
	return e.Err.Code
}

func (e SolanaJSONRPCError) Error() string {
	return fmt.Sprintf("%s: %s", e.CustomMessage, e.Err.Message)
}

type SolanaJSONRPCErrorCode int

const (
	JSON_RPC_SERVER_ERROR_BLOCK_CLEANED_UP SolanaJSONRPCErrorCode = -(iota + 32001)
	JSON_RPC_SERVER_ERROR_SEND_TRANSACTION_PREFLIGHT_FAILURE
	JSON_RPC_SERVER_ERROR_TRANSACTION_SIGNATURE_VERIFICATION_FAILURE
	JSON_RPC_SERVER_ERROR_BLOCK_NOT_AVAILABLE
	JSON_RPC_SERVER_ERROR_NODE_UNHEALTHY
	JSON_RPC_SERVER_ERROR_TRANSACTION_PRECOMPILE_VERIFICATION_FAILURE
	JSON_RPC_SERVER_ERROR_SLOT_SKIPPED
	JSON_RPC_SERVER_ERROR_NO_SNAPSHOT
	JSON_RPC_SERVER_ERROR_LONG_TERM_STORAGE_SLOT_SKIPPED
	JSON_RPC_SERVER_ERROR_KEY_EXCLUDED_FROM_SECONDARY_INDEX
	JSON_RPC_SERVER_ERROR_TRANSACTION_HISTORY_NOT_AVAILABLE
	JSON_RPC_SCAN_ERROR
	JSON_RPC_SERVER_ERROR_TRANSACTION_SIGNATURE_LEN_MISMATCH
	JSON_RPC_SERVER_ERROR_BLOCK_STATUS_NOT_AVAILABLE_YET
	JSON_RPC_SERVER_ERROR_UNSUPPORTED_TRANSACTION_VERSION
	JSON_RPC_SERVER_ERROR_MIN_CONTEXT_SLOT_NOT_REACHED
)

type TransactionExpiredBlockheightExceededError struct {
	Signature string
}

func (e TransactionExpiredBlockheightExceededError) Error() string {
	return fmt.Sprintf("Signature %s has expired: block height exceeded.", e.Signature)
}

type TransactionExpiredNonceInvalidError struct {
	Signature string
}

func (e TransactionExpiredNonceInvalidError) Error() string {
	return fmt.Sprintf("Signature %s has expired: the nonce is no longer valid.", e.Signature)
}

type TransactionExpiredTimeoutError struct {
	Signature string
	desc      string
}

func NewTransactionExpiredTimeoutError(signature string, timeoutSeconds int) TransactionExpiredTimeoutError {
	return TransactionExpiredTimeoutError{
		Signature: signature,
		desc:      fmt.Sprintf("Transaction was not confirmed in %2.d seconds. It is unknown if it succeeded or failed. Check signature `%s` using the Solana Explorer or CLI tools", timeoutSeconds, signature),
	}
}

func (e TransactionExpiredTimeoutError) Error() string {
	return e.desc
}
