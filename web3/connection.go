package web3

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/donutnomad/solana-web3/web3/utils"
	binary "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/ws"
	"github.com/mr-tron/base58"
	"io"
	"log"
	"math/big"
	"net/http"
	"reflect"
	"regexp"
	"strings"
	"time"
)

type Encoding string

var (
	EncodingJsonParsed Encoding = "jsonParsed"
	EncodingBase64     Encoding = "base64"
)

type EncodingData = solana.Data
type AccountInfoD = AccountInfo[EncodingData]

// AccountInfo Information describing an account
type AccountInfo[T any] struct {
	// `true` if this account's data contains a loaded program
	Executable bool `json:"executable,omitempty"`
	// Identifier of the program that owns the account
	Owner PublicKey `json:"owner,omitempty"`
	// Number of lamports assigned to the account
	Lamports uint64 `json:"lamports,omitempty"`
	// Optional data assigned to the account
	Data T `json:"data,omitempty"`
	// Optional rent epoch info for account
	RentEpoch *uint64 `json:"rentEpoch,omitempty"`
	// The data size of the account
	Space uint64 `json:"space,omitempty"`
}

// ParsedAccountData Parsed account data
type ParsedAccountData struct {
	// Name of the program that owns this account
	Program string `json:"program,omitempty"`
	// Parsed account data
	Parsed any `json:"parsed,omitempty"`
	// Space used by account data
	Space uint64 `json:"space,omitempty"`
}

// Context represents extra contextual information for RPC responses
type Context struct {
	ApiVersion string `json:"apiVersion"`
	Slot       uint64 `json:"slot"`
}

// SendOptions represents options for sending transactions
type SendOptions struct {
	SkipPreflight       *bool       `json:"skipPreflight,omitempty"`       // Disable transaction verification step
	PreflightCommitment *Commitment `json:"preflightCommitment,omitempty"` // Preflight commitment level
	MaxRetries          *uint       `json:"maxRetries,omitempty"`          // Maximum number of times for the RPC node to retry sending the transaction to the leader.
	MinContextSlot      *uint64     `json:"minContextSlot,omitempty"`      // The minimum slot that the request can be evaluated at
}

// ConfirmOptions represents options for confirming transactions
type ConfirmOptions struct {
	SkipPreflight       *bool       // Disable transaction verification step
	Commitment          *Commitment // Desired commitment level
	PreflightCommitment *Commitment // Preflight commitment level
	MaxRetries          *uint       // Maximum number of times for the RPC node to retry sending the transaction to the leader.
	MinContextSlot      *uint64     // The minimum slot that the request can be evaluated at
}

// ConfirmedSignaturesForAddress2Options represents options for getConfirmedSignaturesForAddress2
type ConfirmedSignaturesForAddress2Options struct {
	Before TransactionSignature // Start searching backwards from this transaction signature.
	Until  TransactionSignature // Search until this transaction signature is reached, if found before `limit`.
	Limit  *int                 // Maximum transaction signatures to return (between 1 and 1,000, default: 1,000).
}

type RpcResponseAndContext[T any] struct {
	Context Context `json:"context"`
	Value   T       `json:"value"`
}

type SlotRpcResult[T any] struct {
	JsonRPC string            `json:"jsonrpc"`
	Id      int               `json:"id"`
	Result  T                 `json:"result"`
	Error   *RpcResponseError `json:"error"`

	resultIsNil bool
}

type slotRpcResult[T any] struct {
	JsonRPC string            `json:"jsonrpc"`
	Id      int               `json:"id"`
	Result  T                 `json:"result"`
	Error   *RpcResponseError `json:"error"`
}

func (r *SlotRpcResult[T]) UnmarshalJSON(data []byte) error {
	var m = make(map[string]any)
	err := json.Unmarshal(data, &m)
	var resultIsNil = false
	if err == nil {
		if v, ok := m["result"]; ok {
			resultIsNil = v == nil
		}
	}
	var e slotRpcResult[T]
	err = json.Unmarshal(data, &e)
	if err != nil {
		return err
	}

	r.JsonRPC = e.JsonRPC
	r.Id = e.Id
	r.Result = e.Result
	r.Error = e.Error
	r.resultIsNil = resultIsNil

	return nil
}

type RpcResponse[T any] struct {
	JsonRPC string                   `json:"jsonrpc"`
	Id      int                      `json:"id"`
	Result  RpcResponseAndContext[T] `json:"result"`
	Error   *RpcResponseError        `json:"error"`
}

// TransactionSignature represents a transaction signature
type TransactionSignature string

func (s TransactionSignature) IsZero() bool {
	return s == ""
}
func (s TransactionSignature) String() string {
	return string(s)
}

// Commitment
// The level of commitment desired when querying state
//
//	'processed': Query the most recent block which has reached 1 confirmation by the connected node
//	'confirmed': Query the most recent block which has reached 1 confirmation by the cluster
//	'finalized': Query the most recent block which has been finalized by the cluster
type Commitment string

var (
	CommitmentProcessed    Commitment = "processed"
	CommitmentConfirmed    Commitment = "confirmed"
	CommitmentFinalized    Commitment = "finalized"
	CommitmentRecent       Commitment = "recent"       // Deprecated: as of v1.5.5
	CommitmentSingle       Commitment = "single"       // Deprecated: as of v1.5.5
	CommitmentSingleGossip Commitment = "singleGossip" // Deprecated: as of v1.5.5
	CommitmentRoot         Commitment = "root"         // Deprecated: as of v1.5.5
	CommitmentMax          Commitment = "max"          // Deprecated: as of v1.5.5
)

func CommitmentFromString(str string) *Commitment {
	c := Commitment(strings.ToLower(str))
	switch c {
	case CommitmentProcessed:
	case CommitmentConfirmed:
	case CommitmentFinalized:
	case CommitmentRecent:
	case CommitmentSingle:
	case CommitmentSingleGossip:
	case CommitmentRoot:
	case CommitmentMax:
	default:
		return &c
	}
	return nil
}

type RpcParams struct {
	methodName string
	args       []any
}
type RpcBatchRequest = func(requests []RpcParams) []any

type RpcWebSocketClient struct {
}

type Connection struct {
	commitment                       *Commitment
	confirmTransactionInitialTimeout *int
	rpcEndpoint                      string
	rpcWsEndpoint                    string
	rpcClient                        *CustomClient
	wsClient                         *ws.Client
	rpcRequest                       func(ctx context.Context, methodName string, args []any) (string, error)

	disableBlockhashCaching bool

	blockhashInfo struct {
		LatestBlockhash       *BlockhashWithExpiryBlockHeight
		LastFetch             uint64
		SimulatedSignatures   []string
		TransactionSignatures []string
	}
	pollingBlockhash bool
}

func NewConnection(
	endpoint string,
	config *ConnectionConfig,
) (*Connection, error) {
	var conn = Connection{}

	var wsEndpoint = ""
	var httpHeaders map[string]string
	var disableRetryOnRateLimit = false

	if config != nil {
		conn.commitment = config.Commitment
		conn.confirmTransactionInitialTimeout = config.ConfirmTransactionInitialTimeout
		if config.WsEndpoint != nil {
			wsEndpoint = *config.WsEndpoint
		}
		if config.DisableRetryOnRateLimit != nil {
			disableRetryOnRateLimit = *config.DisableRetryOnRateLimit
		}
	}

	u, err := assertEndpointURL(endpoint)
	if err != nil {
		return nil, err
	}
	conn.rpcEndpoint = u
	if len(wsEndpoint) == 0 {
		u, err = utils.MakeWebsocketURL(endpoint)
		if err != nil {
			return nil, err
		}
		conn.rpcWsEndpoint = u
	} else {
		conn.rpcWsEndpoint = wsEndpoint
	}
	conn.rpcClient = NewCustomClient(conn.rpcEndpoint, httpHeaders, disableRetryOnRateLimit)
	conn.rpcRequest = func(ctx context.Context, methodName string, args []any) (string, error) {
		resp, err := conn.rpcClient.SendRequest(ctx, methodName, args)
		if err != nil {
			return "", fmt.Errorf("rpcRequest %s: %w", methodName, err)
		}
		return resp, nil
	}
	conn.wsClient, err = ws.ConnectWithOptions(context.Background(), conn.rpcWsEndpoint, &ws.Options{})
	if err != nil {
		return nil, err
	}
	return &conn, nil
}

func (c *Connection) buildArgsAtLeastConfirmed(args []any, override *Commitment, encoding *Encoding, extra any) ([]any, error) {
	var commitment = c.Commitment()
	if override != nil {
		commitment = override
	}
	if commitment != nil && !(*commitment == CommitmentConfirmed || *commitment == CommitmentFinalized) {
		return nil, fmt.Errorf("using Connection with default commitment: `%s`, but method requires at least `confirmed`", string(*commitment))
	}
	return c.buildArgs(args, commitment, encoding, extra), nil
}

func msg(format string, a ...any) string {
	return fmt.Sprintf(format, a...)
}

func (c *Connection) buildArgs(args []any, override *Commitment, encoding *Encoding, extra any) []any {
	var commitment = c.Commitment()
	if override != nil {
		commitment = override
	}
	if commitment != nil || encoding != nil || extra != nil {
		var options = make(map[string]any)
		if encoding != nil {
			options["encoding"] = *encoding
		}
		if commitment != nil {
			options["commitment"] = *commitment
		}
		if extra != nil {
			for key, value := range utils.StructToMap(extra) {
				if key != "commitment" && key != "encoding" {
					options[key] = value
				}
			}
		}
		return append(args, options)
	}
	return args
}

// Commitment The default commitment used for requests
func (c *Connection) Commitment() *Commitment {
	return c.commitment
}

// RpcEndpoint The RPC endpoint
func (c *Connection) RpcEndpoint() string {
	return c.rpcEndpoint
}

// GetBalanceConfig Configuration object for changing getBalance query behavior
type GetBalanceConfig struct {
	// The level of commitment desired
	Commitment *Commitment
	// The minimum slot that the request can be evaluated at
	MinContextSlot uint64 `json:"minContextSlot,omitempty"`
}

// GetWsClient returns a websocket client for subscription
func (c *Connection) GetWsClient() *ws.Client {
	return c.wsClient
}

// GetBalanceAndContext Fetch the balance for the specified public key, return with context
func (c *Connection) GetBalanceAndContext(publicKey PublicKey, config GetBalanceConfig) (*RpcResponseAndContext[uint64], error) {
	args := c.buildArgs([]any{publicKey.Base58()}, config.Commitment, nil, config)
	return requestContext[uint64](context.Background(), c, "getBalance", args,
		msg("failed to get balance for %s", publicKey),
	)
}

// GetBalance Fetch the balance for the specified public key
func (c *Connection) GetBalance(publicKey PublicKey, config GetBalanceConfig) (uint64, error) {
	res, err := c.GetBalanceAndContext(publicKey, config)
	if err != nil {
		return 0, err
	}
	return res.Value, nil
}

// GetBlockTime Fetch the estimated production time of a block
func (c *Connection) GetBlockTime(slot int64) (uint64, error) {
	return requestNonContextValue[uint64](context.Background(), c, "getBlockTime", []any{slot},
		msg("failed to get block time for slot %d", slot),
	)
}

type TransactionDetail string

const (
	TransactionDetail_Accounts   TransactionDetail = "accounts"
	TransactionDetail_Full       TransactionDetail = "full"
	TransactionDetail_None       TransactionDetail = "none"
	TransactionDetail_Signatures TransactionDetail = "signatures"
)

type GetBlockConfig struct {
	// The level of commitment desired
	Commitment *Commitment
	// The max transaction version to return in responses. If the requested transaction is a higher version, an error will be returned
	MaxSupportedTransactionVersion uint64 `json:"maxSupportedTransactionVersion,omitempty"`
	// Whether to populate the rewards array.
	Rewards bool `json:"rewards,omitempty"`
}

type AnnotatedAccountKey struct {
	Pubkey   string  `json:"pubkey"`
	Signer   bool    `json:"signer"`
	Writable bool    `json:"writable"`
	Source   *string `json:"source,omitempty"` // transaction,lookupTable
}

type RewardsResult struct {
	Pubkey      string  `json:"pubkey"`
	Lamports    int64   `json:"lamports"`
	PostBalance *int64  `json:"postBalance,omitempty"`
	RewardType  *string `json:"rewardType,omitempty"`
	Commission  *int64  `json:"commission,omitempty"`
}

type BlockResponseCommonTx[T any] struct {
	Transaction T                         `json:"transaction,omitempty"`
	Meta        *ConfirmedTransactionMeta `json:"meta,omitempty"`
	Version     *TransactionVersion       `json:"version,omitempty"`
}

type BlockResponse[T any] struct {
	Blockhash         Blockhash                  `json:"blockhash,omitempty"`
	PreviousBlockhash Blockhash                  `json:"previousBlockhash,omitempty"`
	ParentSlot        uint64                     `json:"parentSlot,omitempty"`
	Transactions      []BlockResponseCommonTx[T] `json:"transactions,omitempty"`
	Rewards           []RewardsResult            `json:"rewards,omitempty"`
	BlockTime         int64                      `json:"blockTime,omitempty"`
	BlockHeight       uint64                     `json:"blockHeight,omitempty"`
}

type VersionedTransactionRet struct {
	Message    VersionedMessage `json:"message,omitempty"`
	Signatures []string         `json:"signatures,omitempty"`
}

type AccountsModeTransactionRet struct {
	AccountKeys []AnnotatedAccountKey `json:"accountKeys,omitempty"`
	Signatures  []string              `json:"signatures,omitempty"`
}

// GetBlockWithAccounts Fetch a processed block from the cluster
//
//	transactionDetails: // Level of transaction detail to return, either "full", "accounts", "signatures", or "none". If
//	// parameter not provided, the default detail level is "full". If "accounts" are requested,
//	// transaction details only include signatures and an annotated list of accounts in each
//	// transaction. Transaction metadata is limited to only: fee, err, pre_balances, post_balances,
//	// pre_token_balances, and post_token_balances.
func (c *Connection) GetBlockWithAccounts(slot uint64, config GetBlockConfig) (*BlockResponse[AccountsModeTransactionRet], error) {
	return getBlock[BlockResponse[AccountsModeTransactionRet]](context.Background(), c, slot, config, TransactionDetail_Accounts)
}

// GetBlockWithNone Fetch a processed block from the cluster
// transactions of the response is nil
func (c *Connection) GetBlockWithNone(slot uint64, config GetBlockConfig) (*BlockResponse[struct{}], error) {
	return getBlock[BlockResponse[struct{}]](context.Background(), c, slot, config, TransactionDetail_None)
}

// GetBlock
// detail: default: full
func (c *Connection) GetBlock(slot uint64, config GetBlockConfig) (*BlockResponse[VersionedTransactionRet], error) {
	return c.GetBlockCtx(context.Background(), slot, config)
}

// GetBlockCtx
// detail: default: full
func (c *Connection) GetBlockCtx(ctx context.Context, slot uint64, config GetBlockConfig) (*BlockResponse[VersionedTransactionRet], error) {
	type MessageResponse struct {
		Header              MessageHeader               `json:"header"`
		AccountKeys         []PublicKey                 `json:"accountKeys,omitempty"`
		RecentBlockhash     Blockhash                   `json:"recentBlockhash,omitempty"`
		Instructions        []CompiledInstruction       `json:"instructions,omitempty"`
		AddressTableLookups []MessageAddressTableLookup `json:"addressTableLookups,omitempty"`
	}
	type Tx struct {
		Message    MessageResponse `json:"message,omitempty"`
		Signatures []string        `json:"signatures,omitempty"`
	}
	ret, err := getBlock[BlockResponse[Tx]](ctx, c, slot, config, TransactionDetail_Full)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return nil, nil
	}

	versionMessageFromResponse := func(version *TransactionVersion, response MessageResponse) VersionedMessage {
		if version != nil && *version == TransactionVersion0 {
			return VersionedMessage{
				Raw: MessageV0{
					Header:               response.Header,
					StaticAccountKeys:    response.AccountKeys,
					RecentBlockhash:      response.RecentBlockhash,
					CompiledInstructions: response.Instructions,
					AddressTableLookups:  response.AddressTableLookups,
				},
			}
		} else {
			m := Message{
				Header:          response.Header,
				AccountKeys:     response.AccountKeys,
				RecentBlockhash: response.RecentBlockhash,
				Instructions:    response.Instructions,
			}
			m.inflate()
			return VersionedMessage{
				Raw: m,
			}
		}
	}

	var txs []BlockResponseCommonTx[VersionedTransactionRet]
	for _, item := range ret.Transactions {
		txs = append(txs, BlockResponseCommonTx[VersionedTransactionRet]{
			Transaction: VersionedTransactionRet{
				Message:    versionMessageFromResponse(item.Version, item.Transaction.Message),
				Signatures: item.Transaction.Signatures,
			},
			Meta:    item.Meta,
			Version: item.Version,
		})
	}
	return &BlockResponse[VersionedTransactionRet]{
		Blockhash:         ret.Blockhash,
		PreviousBlockhash: ret.PreviousBlockhash,
		ParentSlot:        ret.ParentSlot,
		Transactions:      txs,
		Rewards:           ret.Rewards,
		BlockTime:         ret.BlockTime,
		BlockHeight:       ret.BlockHeight,
	}, nil
}

func getBlock[T any](ctx context.Context, c *Connection, slot uint64, config GetBlockConfig, detail TransactionDetail) (*T, error) {
	opt := map[string]any{
		"transactionDetails":             detail,
		"maxSupportedTransactionVersion": config.MaxSupportedTransactionVersion,
		"rewards":                        config.Rewards,
	}
	// Default: finalized
	//  * processed is not supported.
	var commitment = CommitmentFinalized
	if config.Commitment != nil {
		if *config.Commitment != CommitmentProcessed {
			commitment = *config.Commitment
		}
	}
	args, err := c.buildArgsAtLeastConfirmed([]any{slot}, &commitment, nil, opt)
	if err != nil {
		return nil, err
	}
	return requestNonContext[T](ctx, c, "getBlock", args, "failed to get block")
}

// GetMinimumLedgerSlot Fetch the lowest slot that the node has information about in its ledger.
// This value may increase over time if the node is configured to purge older ledger data
func (c *Connection) GetMinimumLedgerSlot() (uint64, error) {
	return requestContextValue[uint64](context.Background(), c, "minimumLedgerSlot", nil,
		"failed to get minimum ledger slot",
	)
}

// GetFirstAvailableBlock Fetch the slot of the lowest confirmed block that has not been purged from the ledger
func (c *Connection) GetFirstAvailableBlock() (uint64, error) {
	return requestNonContextValue[uint64](context.Background(), c, "getFirstAvailableBlock", nil,
		"failed to get first available block",
	)
}

// GetSupplyConfig Configuration object for changing `getSupply` request behavior
type GetSupplyConfig struct {
	// The level of commitment desired
	Commitment *Commitment
	// Exclude non circulating accounts list from response
	ExcludeNonCirculatingAccountsList bool `json:"excludeNonCirculatingAccountsList,omitempty"`
}

// Supply represents the supply information
type Supply struct {
	// Total supply in lamports
	Total uint64 `json:"total"`
	// Circulating supply in lamports
	Circulating uint64 `json:"circulating"`
	// Non-circulating supply in lamports
	NonCirculating uint64 `json:"nonCirculating"`
	// List of non-circulating account addresses
	NonCirculatingAccounts []PublicKey `json:"nonCirculatingAccounts"`
}

// GetSupply Fetch information about the current supply
func (c *Connection) GetSupply(config GetSupplyConfig) (*RpcResponseAndContext[Supply], error) {
	args := c.buildArgs(nil, config.Commitment, nil, config)
	return requestContext[Supply](context.Background(), c, "getSupply", args, "failed to get supply")
}

// TokenAmount represents a token amount object in different formats for various use cases.
type TokenAmount struct {
	// Raw amount of tokens as string ignoring decimals
	Amount string `json:"amount"`
	// Number of decimals configured for the token's mint
	Decimals int `json:"decimals"`
	// Token amount as float, accounts for decimals
	UIAmount *float64 `json:"uiAmount,omitempty"`
	// Token amount as string, accounts for decimals
	UIAmountString *string `json:"uiAmountString,omitempty"`
}

// GetTokenSupply Fetch the current supply of a token mint
func (c *Connection) GetTokenSupply(tokenMintAddress PublicKey, commitment *Commitment) (*RpcResponseAndContext[TokenAmount], error) {
	args := c.buildArgs([]any{tokenMintAddress.Base58()}, commitment, nil, nil)
	return requestContext[TokenAmount](context.Background(), c, "getTokenSupply", args, "failed to get token supply")
}

// GetTokenAccountBalance Fetch the current balance of a token account
func (c *Connection) GetTokenAccountBalance(tokenAddress PublicKey, commitment *Commitment) (*RpcResponseAndContext[TokenAmount], error) {
	args := c.buildArgs([]any{tokenAddress.Base58()}, commitment, nil, nil)
	return requestContext[TokenAmount](context.Background(), c, "getTokenAccountBalance", args, "failed to get token account balance")
}

type GetTokenAccountsByDelegateConfig struct {
	// Optional commitment level
	Commitment *Commitment
	// The minimum slot that the request can be evaluated at
	MinContextSlot uint64 `json:"minContextSlot,omitempty"`
}

type GetTokenAccountsByDelegateResponse struct {
	Pubkey  PublicKey `json:"pubkey,omitempty"`
	Account struct {
		Executable bool            `json:"executable,omitempty"`
		Owner      PublicKey       `json:"owner,omitempty"`
		Lamports   uint64          `json:"lamports,omitempty"`
		Data       json.RawMessage `json:"data,omitempty"`
		RentEpoch  uint64          `json:"rentEpoch,omitempty"`
	} `json:"account"`
}

// GetTokenAccountsByDelegate Returns all SPL Token accounts by approved Delegate.
func (c *Connection) GetTokenAccountsByDelegate(ownerAddress PublicKey, filter TokenAccountsFilter, config GetTokenAccountsByDelegateConfig) (*RpcResponseAndContext[GetTokenAccountsByDelegateResponse], error) {
	var _args = []any{ownerAddress.Base58()}
	if filter.mint != nil {
		_args = append(_args, _M{"mint": filter.mint.Base58()})
	} else {
		_args = append(_args, _M{"programId": filter.programId.Base58()})
	}
	args := c.buildArgs(_args, config.Commitment, &EncodingBase64, config)
	return requestContext[GetTokenAccountsByDelegateResponse](context.Background(), c, "getTokenAccountsByDelegate", args,
		msg("failed to get token accounts owned by account %s", ownerAddress),
	)
}

type GetTokenAccountsByOwnerConfig struct {
	// Optional commitment level
	Commitment *Commitment
	// The minimum slot that the request can be evaluated at
	MinContextSlot uint64 `json:"minContextSlot,omitempty"`
}

type GetProgramAccountsResponse struct {
	Account AccountInfoD `json:"account"`
	Pubkey  PublicKey    `json:"pubkey"`
}

type TokenAccountsFilter struct {
	mint      *PublicKey
	programId PublicKey
}

func NewTokenAccountsFilter(mint PublicKey) TokenAccountsFilter {
	return TokenAccountsFilter{
		mint: &mint,
	}
}

func NewTokenAccountsFilterProgram(programId PublicKey) TokenAccountsFilter {
	return TokenAccountsFilter{
		programId: programId,
	}
}

type _M = map[string]any

// GetTokenAccountsByOwner Fetch all the token accounts owned by the specified account
func (c *Connection) GetTokenAccountsByOwner(
	ownerAddress PublicKey,
	filter TokenAccountsFilter,
	config GetTokenAccountsByOwnerConfig,
) (*RpcResponseAndContext[[]GetProgramAccountsResponse], error) {
	_args := []any{ownerAddress.Base58()}
	if filter.mint != nil {
		_args = append(_args, _M{"mint": filter.mint.Base58()})
	} else {
		_args = append(_args, _M{"programId": filter.programId.Base58()})
	}
	args := c.buildArgs(_args, config.Commitment, &EncodingBase64, config)
	return requestContext[[]GetProgramAccountsResponse](context.Background(), c, "getTokenAccountsByOwner", args, msg("failed to get token accounts owned by account %s", ownerAddress))
}

type ParsedAccount struct {
	Account AccountInfo[ParsedAccountData] `json:"account"`
	Pubkey  PublicKey                      `json:"pubkey"`
}

// GetParsedTokenAccountsByOwner Fetch parsed token accounts owned by the specified account
func (c *Connection) GetParsedTokenAccountsByOwner(
	ownerAddress PublicKey,
	filter TokenAccountsFilter,
	commitment *Commitment,
) (*RpcResponseAndContext[[]ParsedAccount], error) {
	_args := []any{ownerAddress.Base58()}
	if filter.mint != nil {
		_args = append(_args, map[string]any{"mint": filter.mint.Base58()})
	} else {
		_args = append(_args, map[string]any{"programId": filter.programId.Base58()})
	}
	args := c.buildArgs(_args, commitment, &EncodingJsonParsed, nil)
	return requestContext[[]ParsedAccount](context.Background(), c, "getTokenAccountsByOwner", args, msg("failed to get token accounts owned by account %s", ownerAddress))
}

// GetLargestAccountsConfig Configuration object for changing `getLargestAccounts` query behavior
type GetLargestAccountsConfig struct {
	// The level of commitment desired
	Commitment *Commitment
	// Filter largest accounts by whether they are part of the circulating supply
	Filter *LargestAccountsFilter `json:"filter"`
}

type LargestAccountsFilter string

var (
	LargestAccountsFilterCirculating    LargestAccountsFilter = "circulating"
	LargestAccountsFilterNonCirculating LargestAccountsFilter = "nonCirculating"
)

// AccountBalancePair Pair of an account address and its balance
type AccountBalancePair struct {
	Address  PublicKey `json:"address"`
	Lamports uint64    `json:"lamports"`
}

// GetLargestAccounts Fetch the 20 largest accounts with their current balances
func (c *Connection) GetLargestAccounts(config GetLargestAccountsConfig) (*RpcResponseAndContext[[]AccountBalancePair], error) {
	args := c.buildArgs(nil, config.Commitment, nil, config)
	return requestContext[[]AccountBalancePair](context.Background(), c, "getLargestAccounts", args, "failed to get largest accounts")
}

// TokenAccountBalancePair Token address and balance.
type TokenAccountBalancePair struct {
	Address PublicKey `json:"address"`
	TokenAmount
}

// GetTokenLargestAccounts Fetch the 20 largest token accounts with their current balances
// for a given mint.
func (c *Connection) GetTokenLargestAccounts(mintAddress PublicKey, commitment *Commitment) (*RpcResponseAndContext[[]TokenAccountBalancePair], error) {
	args := c.buildArgs([]any{mintAddress.Base58()}, commitment, nil, nil)
	return requestContext[[]TokenAccountBalancePair](context.Background(), c, "getTokenLargestAccounts", args, "failed to get token largest accounts")
}

// GetAccountInfoConfig Configuration object for changing `getAccountInfo` query behavior
type GetAccountInfoConfig struct {
	//  The level of commitment desired
	Commitment *Commitment
	//  The minimum slot that the request can be evaluated at
	MinContextSlot uint64 `json:"minContextSlot,omitempty"`
	//  Optional data slice to limit the returned account data
	DataSlice *DataSlice `json:"dataSlice,omitempty"`
}

// GetAccountInfoAndContext Fetch all the account info for the specified public key, return with context
func (c *Connection) GetAccountInfoAndContext(publicKey PublicKey, config GetAccountInfoConfig) (*RpcResponseAndContext[*AccountInfoD], error) {
	args := c.buildArgs([]any{publicKey.Base58()}, config.Commitment, &EncodingBase64, config)
	return requestContext[*AccountInfoD](context.Background(), c, "getAccountInfo", args, msg("failed to get info about account %s", publicKey))
}

// GetParsedAccountInfo Fetch parsed account info for the specified public key
func (c *Connection) GetParsedAccountInfo(publicKey PublicKey, config GetAccountInfoConfig) (*RpcResponseAndContext[*AccountInfo[ParsedAccountData]], error) {
	args := c.buildArgs([]any{publicKey.Base58()}, config.Commitment, &EncodingJsonParsed, config)
	return requestContext[*AccountInfo[ParsedAccountData]](context.Background(), c, "getAccountInfo", args, msg("failed to get info about account %s", publicKey))
}

// GetAccountInfo Fetch all the account info for the specified public key
func (c *Connection) GetAccountInfo(publicKey PublicKey, config GetAccountInfoConfig) (*AccountInfoD, error) {
	resp, err := c.GetAccountInfoAndContext(publicKey, config)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("failed to get info about account %s", publicKey), err)
	}
	return resp.Value, nil
}

// GetMultipleAccountsConfig Configuration object for getMultipleAccounts
type GetMultipleAccountsConfig struct {
	//  Optional commitment level
	Commitment *Commitment
	//  The minimum slot that the request can be evaluated at
	MinContextSlot uint64 `json:"minContextSlot,omitempty"`
	//  Optional data slice to limit the returned account data
	DataSlice *DataSlice `json:"dataSlice,omitempty"`
}

// GetMultipleParsedAccounts Fetch all the account info for multiple accounts specified by an array of public keys, return with context
func (c *Connection) GetMultipleParsedAccounts(publicKeys []PublicKey, config GetMultipleAccountsConfig) (*RpcResponseAndContext[[]*AccountInfo[ParsedAccountDataOrBytes]], error) {
	keys := utils.Map(publicKeys, func(t PublicKey) string {
		return t.Base58()
	})
	args := c.buildArgs([]any{keys}, config.Commitment, &EncodingJsonParsed, config)
	return requestContext[[]*AccountInfo[ParsedAccountDataOrBytes]](context.Background(), c, "getMultipleAccounts", args, msg("failed to get info for accounts %v", keys))
}

// GetMultipleAccountsInfoAndContext Fetch all the account info for multiple accounts specified by an array of public keys, return with context
func (c *Connection) GetMultipleAccountsInfoAndContext(publicKeys []PublicKey, config GetMultipleAccountsConfig) (*RpcResponseAndContext[[]*AccountInfoD], error) {
	keys := utils.Map(publicKeys, func(t PublicKey) string {
		return t.Base58()
	})
	args := c.buildArgs([]any{keys}, config.Commitment, &EncodingBase64, config)
	return requestContext[[]*AccountInfoD](context.Background(), c, "getMultipleAccounts", args, msg("failed to get info for accounts %v", keys))
}

// GetMultipleAccountsInfo Fetch all the account info for multiple accounts specified by an array of public keys
// publicKeys: up to a maximum of 100
func (c *Connection) GetMultipleAccountsInfo(publicKeys []PublicKey, config GetMultipleAccountsConfig) ([]*AccountInfoD, error) {
	res, err := c.GetMultipleAccountsInfoAndContext(publicKeys, config)
	if err != nil {
		return nil, err
	}
	return res.Value, nil
}

// GetStakeActivationConfig Configuration object for `getStakeActivation`
type GetStakeActivationConfig struct {
	//  Optional commitment level
	Commitment *Commitment
	//  Epoch for which to calculate activation details. If parameter not provided, defaults to current epoch
	Epoch uint64 `json:"epoch,omitempty"`
	//  The minimum slot that the request can be evaluated at
	MinContextSlot uint64 `json:"minContextSlot,omitempty"`
}

// StakeActivationData Stake Activation data
type StakeActivationData struct {
	// the stake account's activation state
	// 'active' | 'inactive' | 'activating' | 'deactivating'
	State string `json:"state"`
	// stake active during the epoch
	Active uint64 `json:"active"`
	// stake inactive during the epoch
	Inactive uint64 `json:"inactive"`
}

// getStakeActivation Returns epoch activation information for a stake account that has been delegated
func (c *Connection) getStakeActivation(publicKey PublicKey, config GetStakeActivationConfig) (*StakeActivationData, error) {
	args := c.buildArgs([]any{publicKey.Base58()}, config.Commitment, nil, config)
	ret, err := requestContext[StakeActivationData](context.Background(), c, "getStakeActivation", args, msg("failed to get Stake Activation %s", publicKey))
	if err != nil {
		return nil, err
	}
	return &ret.Value, nil
}

type DataSlice struct {
	Offset *uint64 `json:"offset,omitempty"`
	Length *uint64 `json:"length,omitempty"`
}

// GetProgramAccountsConfig Configuration object for getProgramAccounts requests
type GetProgramAccountsConfig struct {
	// Optional commitment level
	Commitment *Commitment
	// Optional encoding for account data (default base64)
	// To use "jsonParsed" encoding, please refer to `getParsedProgramAccounts` in connection.ts
	Encoding Encoding `json:"encoding,omitempty"`
	// Optional data slice to limit the returned account data
	DataSlice *DataSlice `json:"dataSlice,omitempty"`
	// Optional array of filters to apply to accounts
	Filters []GetProgramAccountsFilter `json:"filters,omitempty"`
	// The minimum slot that the request can be evaluated at
	MinContextSlot uint64 `json:"minContextSlot,omitempty"`
	// wrap the result in an RpcResponse JSON object
	WithContext bool `json:"withContext,omitempty"`
}

type RPCFilterMemcmp struct {
	Offset uint64        `json:"offset"`
	Bytes  solana.Base58 `json:"bytes"`
}

type GetProgramAccountsFilter struct {
	Memcmp   *RPCFilterMemcmp `json:"memcmp,omitempty"`
	DataSize *uint64          `json:"dataSize,omitempty"`
}

type MemcmpFilter struct {
	Memcmp *struct {
		// offset into program account data to start comparison
		Offset uint64
		// data to match, as base-58 encoded string and limited to less than 129 bytes
		Bytes string
	}
}

type DataSizeFilter struct {
	DataSize *uint64
}

// GetProgramAccounts Fetch all the accounts owned by the specified program id
func (c *Connection) GetProgramAccounts(programId PublicKey, config GetProgramAccountsConfig) ([]GetProgramAccountsResponse, error) {
	config.WithContext = true
	res, err := c.GetProgramAccountsAndContext(programId, config)
	if err != nil {
		return nil, err
	}
	return res.Value, nil
}

// GetProgramAccountsAndContext Fetch all the accounts owned by the specified program id
func (c *Connection) GetProgramAccountsAndContext(programId PublicKey, config GetProgramAccountsConfig) (*RpcResponseAndContext[[]GetProgramAccountsResponse], error) {
	if config.Encoding == "" {
		config.Encoding = EncodingBase64
	}
	args := c.buildArgs([]any{programId.Base58()}, config.Commitment, &config.Encoding, config)
	return requestContext[[]GetProgramAccountsResponse](context.Background(), c, "getProgramAccounts", args, msg("failed to get accounts owned by program  %s", programId))
}

// GetParsedProgramAccountsConfig is the configuration object for getParsedProgramAccounts.
type GetParsedProgramAccountsConfig struct {
	// Optional commitment level
	Commitment *Commitment
	// Optional array of filters to apply to accounts
	Filters []GetProgramAccountsFilter `json:"filters,omitempty"`
	// The minimum slot that the request can be evaluated at
	MinContextSlot uint64 `json:"minContextSlot,omitempty"`
}

// GetParsedProgramAccounts Fetch and parse all the accounts owned by the specified program id
func (c *Connection) GetParsedProgramAccounts(programId PublicKey, config GetParsedProgramAccountsConfig) ([]struct {
	Pubkey  PublicKey                             `json:"pubkey"`
	Account AccountInfo[ParsedAccountDataOrBytes] `json:"account"`
}, error) {
	args := c.buildArgs([]any{programId.Base58()}, config.Commitment, &EncodingJsonParsed, config)
	return requestNonContextValue[[]struct {
		Pubkey  PublicKey                             `json:"pubkey"`
		Account AccountInfo[ParsedAccountDataOrBytes] `json:"account"`
	}](context.Background(), c, "getProgramAccounts", args, msg("failed to get accounts owned by program %s", programId))
}

type BlockhashWithExpiryBlockHeight struct {
	Blockhash            Blockhash `json:"blockhash,omitempty"`
	LastValidBlockHeight uint64    `json:"lastValidBlockHeight,omitempty"`
}

// BlockheightBasedTransactionConfirmationStrategy A strategy for confirming transactions that uses the last valid
// block height for a given blockhash to check for transaction expiration.
type BlockheightBasedTransactionConfirmationStrategy struct {
	Signature TransactionSignature
	BlockhashWithExpiryBlockHeight
}

// DurableNonceTransactionConfirmationStrategy A strategy for confirming durable nonce transactions.
type DurableNonceTransactionConfirmationStrategy struct {
	Signature TransactionSignature
	// The lowest slot at which to fetch the nonce value from the
	// nonce account. This should be no lower than the slot at
	// which the last-known value of the nonce was fetched.
	MinContextSlot uint64
	// The account where the current value of the nonce is stored.
	NonceAccountPubkey PublicKey
	// The nonce value that was used to sign the transaction
	// for which confirmation is being sought.
	NonceValue DurableNonce
}

type SignatureResult struct {
	Err any `json:"err"`
}

func (s SignatureResult) HasErr() bool {
	if s.Err == nil {
		return true
	}
	return reflect.ValueOf(s.Err).IsNil()
}

type SimulatedTransactionAccountInfo struct {
	// True if this account's data contains a loaded program
	Executable bool `json:"executable"`
	// Identifier of the program that owns the account
	Owner string `json:"owner"`
	// Number of lamports assigned to the account
	Lamports uint64 `json:"lamports"`
	// Optional data assigned to the account
	Data []string `json:"data"`
	// Optional rent epoch info for account
	RentEpoch *uint64 `json:"rentEpoch,omitempty"`
}

type TransactionReturnData struct {
	ProgramId string      `json:"programId"`
	Data      solana.Data `json:"data"`
}

func (r *TransactionReturnData) Bytes() []byte {
	return r.Data.Content
}

type SimulatedTransactionResponse struct {
	Err           TransactionError                  `json:"err,omitempty"`
	Logs          []string                          `json:"logs,omitempty"`
	Accounts      []SimulatedTransactionAccountInfo `json:"accounts,omitempty"`
	UnitsConsumed uint64                            `json:"unitsConsumed,omitempty"`
	ReturnData    *TransactionReturnData            `json:"returnData,omitempty"`
}

type SimulateTransactionConfig struct {
	// Optional parameter used to enable signature verification before simulation
	SigVerify *bool `json:"sigVerify,omitempty"`
	// Optional parameter used to replace the simulated transaction's recent blockhash with the latest blockhash
	ReplaceRecentBlockhash *bool `json:"replaceRecentBlockhash,omitempty"`
	// Optional parameter used to set the commitment level when selecting the latest block
	Commitment *Commitment
	// Optional parameter used to specify a list of account addresses to return post simulation state for
	Accounts []PublicKey
	// Optional parameter used to specify the minimum block slot that can be used for simulation
	MinContextSlot *int64 `json:"minContextSlot,omitempty"`
}

func (c *Connection) SimulateTransaction(tx Transaction, config SimulateTransactionConfig, signers []Signer) (SimulatedTransactionResponse, error) {
	var transaction = Transaction{
		feePayer:     &*tx.feePayer,
		instructions: tx.instructions,
		NonceInfo:    tx.NonceInfo,
		signatures:   tx.signatures,
	}

	if transaction.NonceInfo != nil && len(signers) > 0 {
		err := transaction.Sign(signers...)
		if err != nil {
			return SimulatedTransactionResponse{}, err
		}
	} else {
		disableCache := c.disableBlockhashCaching
		for {
			latestBlockhash, err := c._blockhashWithExpiryBlockHeight(disableCache)
			if err != nil {
				return SimulatedTransactionResponse{}, err
			}
			transaction.LastValidBlockHeight = &latestBlockhash.LastValidBlockHeight
			transaction.RecentBlockhash = &latestBlockhash.Blockhash

			if len(signers) == 0 {
				break
			}

			if err := transaction.Sign(signers...); err != nil {
				return SimulatedTransactionResponse{}, err
			}
			if transaction.Signature().IsZero() {
				return SimulatedTransactionResponse{}, errors.New("!signature") // should never happen
			}

			var s = transaction.Signature()
			signature := base64.StdEncoding.EncodeToString(s[:])
			if !utils.Contain(c.blockhashInfo.SimulatedSignatures, signature) {
				// The signature of this transaction has not been seen before with the
				// current recentBlockhash, all done. Let's break
				c.blockhashInfo.SimulatedSignatures = append(c.blockhashInfo.SimulatedSignatures, signature)
				break
			} else {
				// This transaction would be treated as duplicate (its derived signature
				// matched to one of already recorded signatures).
				// So, we must fetch a new blockhash for a different signature by disabling
				// our cache not to wait for the cache expiration (BLOCKHASH_CACHE_TIMEOUT_MS).
				disableCache = true
			}
		}
	}

	wireTransaction, err := transaction.Serialize()
	if err != nil {
		return SimulatedTransactionResponse{}, err
	}
	encodedTransaction := base64.StdEncoding.EncodeToString(wireTransaction)
	if len(signers) > 0 {
		config.SigVerify = Ref(true)
	}
	extra := utils.StructToMap(config)
	if len(config.Accounts) > 0 {
		addresses := utils.Map(config.Accounts, func(t PublicKey) string {
			return t.String()
		})
		extra["accounts"] = map[string]any{
			"encoding":  "base64",
			"addresses": addresses,
		}
	}
	args := c.buildArgs([]any{encodedTransaction}, config.Commitment, &EncodingBase64, extra)
	return requestContextValue[SimulatedTransactionResponse](context.Background(), c, "simulateTransaction", args, "failed to simulate transaction")
}

func (c *Connection) SimulateTransactionV0(transaction VersionedTransaction) (SimulatedTransactionResponse, error) {
	encodedTransaction := base64.StdEncoding.EncodeToString(transaction.Serialize())
	args := c.buildArgs([]any{encodedTransaction}, nil, &EncodingBase64, nil)
	return requestContextValue[SimulatedTransactionResponse](context.Background(), c, "simulateTransaction", args, "failed to simulate transaction")
}

func (c *Connection) SendAndConfirmTransaction(
	ctx context.Context, tx Transaction, signers []Signer, options ConfirmOptions,
) (TransactionSignature, error) {
	transaction := &tx
	signature, err := c.SendTransaction(transaction, signers, SendOptions{
		SkipPreflight:       options.SkipPreflight,
		PreflightCommitment: options.PreflightCommitment,
		MaxRetries:          options.MaxRetries,
		MinContextSlot:      options.MinContextSlot,
	})
	if err != nil {
		return "", err
	}

	var status SignatureResult
	if transaction.RecentBlockhash != nil && transaction.LastValidBlockHeight != nil {
		ret, err := c.ConfirmTransaction(ctx, BlockheightBasedTransactionConfirmationStrategy{
			Signature: signature,
			BlockhashWithExpiryBlockHeight: BlockhashWithExpiryBlockHeight{
				Blockhash:            *transaction.RecentBlockhash,
				LastValidBlockHeight: *transaction.LastValidBlockHeight,
			},
		}, options.Commitment)
		if err != nil {
			return "", nil
		}
		status = ret.Value
	} else if transaction.MinNonceContextSlot != nil && transaction.NonceInfo != nil {
		nonceAccountPubkey := transaction.NonceInfo.NonceInstruction.Keys[0].Pubkey
		ret, err := c.ConfirmTransaction(ctx, DurableNonceTransactionConfirmationStrategy{
			Signature:          signature,
			MinContextSlot:     *transaction.MinNonceContextSlot,
			NonceAccountPubkey: nonceAccountPubkey,
			NonceValue:         transaction.NonceInfo.Nonce,
		}, options.Commitment)
		if err != nil {
			return "", nil
		}
		status = ret.Value
	} else {
		ret, err := c.ConfirmTransaction(ctx, signature, options.Commitment)
		if err != nil {
			return "", nil
		}
		status = ret.Value
	}

	if !status.HasErr() {
		var content = ""
		marshal, err := json.Marshal(status)
		if err == nil {
			content = string(marshal)
		}
		return "", fmt.Errorf("transaction %s failed %s", signature, content)
	}

	return signature, nil
}

func (c *Connection) ConfirmTransaction(
	ctx context.Context,
	strategy any,
	commitment *Commitment,
) (*RpcResponseAndContext[SignatureResult], error) {
	if commitment == nil {
		commitment = c.Commitment()
	}

	var checkSignature = func(rawSignature TransactionSignature) error {
		decodedSignature, err := base58.Decode(string(rawSignature))
		if err != nil {
			return fmt.Errorf("signature must be base58 encoded: %s", string(rawSignature))
		}
		if len(decodedSignature) != 64 {
			return errors.New("signature has invalid length")
		}
		return nil
	}

	if v, ok := strategy.(string); ok {
		if err := checkSignature(TransactionSignature(v)); err != nil {
			return nil, err
		}
		return c.confirmTransactionUsingLegacyTimeoutStrategy(ctx, commitment, TransactionSignature(v))
	}
	if v, ok := strategy.(BlockheightBasedTransactionConfirmationStrategy); ok {
		if err := checkSignature(v.Signature); err != nil {
			return nil, err
		}
		return c.confirmTransactionUsingBlockHeightExceedanceStrategy(ctx, commitment, v)
	}
	if v, ok := strategy.(DurableNonceTransactionConfirmationStrategy); ok {
		if err := checkSignature(v.Signature); err != nil {
			return nil, err
		}
		return c.confirmTransactionUsingDurableNonceStrategy(ctx, commitment, v)
	}
	return nil, errors.New("invalid strategy type")
}

func getTimeout(initialTimeout *int, commitment *Commitment) int {
	var timeoutMs int
	if initialTimeout != nil {
		timeoutMs = *initialTimeout
	} else {
		timeoutMs = 60 * 1000
	}
	if commitment != nil {
		switch *commitment {
		case CommitmentProcessed:
			fallthrough
		case CommitmentRecent:
			fallthrough
		case CommitmentSingle:
			fallthrough
		case CommitmentConfirmed:
			fallthrough
		case CommitmentSingleGossip:
			if initialTimeout != nil {
				timeoutMs = *initialTimeout
			} else {
				timeoutMs = 30 * 1000
			}
		default:
		}
	}
	return timeoutMs
}

func (c *Connection) confirmation(ctx context.Context, wsClient *ws.Client, signature TransactionSignature, commitment_ *Commitment) (*RpcResponseAndContext[SignatureResult], error) {
	var commitment = Commitment("")
	if commitment_ != nil {
		commitment = *commitment_
	}
	response, err := c.GetSignatureStatus(signature, SignatureStatusConfig{})
	if err == nil {
		value := response.Value
		if commitment == CommitmentConfirmed || commitment == CommitmentSingle || commitment == CommitmentSingleGossip {
			if value.ConfirmationStatus == TransactionConfirmationStatusProcessed {
				// Wait Websocket
			} else {
				return &RpcResponseAndContext[SignatureResult]{
					Context: response.Context,
					Value: SignatureResult{
						Err: value.Err,
					},
				}, nil
			}
		}
		if commitment == CommitmentFinalized || commitment == CommitmentMax || commitment == CommitmentRoot {
			if value.ConfirmationStatus == TransactionConfirmationStatusProcessed || value.ConfirmationStatus == TransactionConfirmationStatusConfirmed {
				// Wait Websocket
			} else {
				return &RpcResponseAndContext[SignatureResult]{
					Context: response.Context,
					Value: SignatureResult{
						Err: value.Err,
					},
				}, nil
			}
		}
	}
	sub, err := wsClient.SignatureSubscribe(
		solana.MustSignatureFromBase58(string(signature)),
		rpc.CommitmentType(commitment),
	)
	if err != nil {
		return nil, err
	}
	defer sub.Unsubscribe()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case resp, ok := <-sub.Response():
			if !ok {
				return nil, fmt.Errorf("subscription closed")
			}
			return &RpcResponseAndContext[SignatureResult]{
				Context: response.Context,
				Value: SignatureResult{
					resp.Value.Err,
				},
			}, nil
		case err := <-sub.Err():
			return nil, err
		}
	}
}

type tmpConfirmResponse struct {
	response *RpcResponseAndContext[SignatureResult]
	err      error
}

func (c *Connection) confirmTransactionUsingLegacyTimeoutStrategy(ctx context.Context, commitment *Commitment, signature TransactionSignature) (*RpcResponseAndContext[SignatureResult], error) {
	var ch = make(chan tmpConfirmResponse)
	go func() {
		confirmation, err := c.confirmation(ctx, c.wsClient, signature, commitment)
		ch <- tmpConfirmResponse{
			response: confirmation,
			err:      err,
		}
	}()
	timeoutMs := getTimeout(c.confirmTransactionInitialTimeout, commitment)
	for {
		select {
		case v := <-ch:
			return v.response, v.err
		case <-time.After(time.Duration(timeoutMs) * time.Millisecond):
			return nil, NewTransactionExpiredTimeoutError(string(signature), timeoutMs/1000)
		}
	}
}

func (c *Connection) confirmTransactionUsingBlockHeightExceedanceStrategy(
	ctx context.Context,
	commitment *Commitment,
	strategy BlockheightBasedTransactionConfirmationStrategy,
) (*RpcResponseAndContext[SignatureResult], error) {
	var expiry = make(chan byte)
	var ch = make(chan tmpConfirmResponse)
	go func() {
		confirmation, err := c.confirmation(ctx, c.wsClient, strategy.Signature, commitment)
		ch <- tmpConfirmResponse{
			response: confirmation,
			err:      err,
		}
	}()
	go func() {
		checkBlockHeight := func() *uint64 {
			ret, err := c.GetBlockHeight(GetBlockHeightConfig{
				Commitment: commitment,
			})
			if err != nil {
				return nil
			}
			return &ret
		}
		currentBlockHeight := checkBlockHeight()
		for currentBlockHeight == nil || *currentBlockHeight <= strategy.LastValidBlockHeight {
			time.Sleep(1 * time.Second)
			currentBlockHeight = checkBlockHeight()
		}
		expiry <- 0
	}()
	for {
		select {
		case v := <-ch:
			return v.response, v.err
		case <-expiry:
			return nil, TransactionExpiredBlockheightExceededError{
				Signature: string(strategy.Signature),
			}
		}
	}
}

func (c *Connection) confirmTransactionUsingDurableNonceStrategy(ctx context.Context, commitment *Commitment, strategy DurableNonceTransactionConfirmationStrategy) (*RpcResponseAndContext[SignatureResult], error) {
	var expiry = make(chan *uint64)
	var ch = make(chan tmpConfirmResponse)
	go func() {
		confirmation, err := c.confirmation(ctx, c.wsClient, strategy.Signature, commitment)
		ch <- tmpConfirmResponse{
			response: confirmation,
			err:      err,
		}
	}()
	go func() {
		var currentNonceValue = strategy.NonceValue
		var lastCheckedSlot *uint64 = nil
		getCurrentNonceValue := func() string {
			resp, err := c.GetNonceAndContext(strategy.NonceAccountPubkey, GetNonceAndContextConfig{
				Commitment:     commitment,
				MinContextSlot: strategy.MinContextSlot,
			})
			if err != nil {
				return currentNonceValue
			}
			lastCheckedSlot = &resp.Context.Slot
			return resp.Value.Nonce.String()
		}
		currentNonceValue = getCurrentNonceValue()
		for {
			if strategy.NonceValue != currentNonceValue {
				expiry <- lastCheckedSlot
				break
			}
			time.Sleep(2 * time.Second)
			currentNonceValue = getCurrentNonceValue()
		}
	}()
	for {
		select {
		case v := <-ch:
			return v.response, v.err
		case lastCheckedSlot := <-expiry:
			var signatureStatus *RpcResponseAndContext[SignatureStatus]
			for {
				status, err := c.GetSignatureStatus(strategy.Signature, SignatureStatusConfig{})
				if err != nil {
					return nil, err
				}
				if status == nil {
					break
				}
				slot := strategy.MinContextSlot
				if lastCheckedSlot != nil {
					slot = *lastCheckedSlot
				}
				if status.Context.Slot < slot {
					time.Sleep(400 * time.Millisecond)
					continue
				}
				signatureStatus = status
			}
			tErr := TransactionExpiredNonceInvalidError{
				Signature: string(strategy.Signature),
			}
			if signatureStatus != nil {
				commitmentForStatus := CommitmentFinalized
				if commitment != nil {
					commitmentForStatus = *commitment
				}
				confirmationStatus := signatureStatus.Value.ConfirmationStatus
				if commitmentForStatus == CommitmentProcessed || commitmentForStatus == CommitmentRecent {
					if confirmationStatus != TransactionConfirmationStatusProcessed && confirmationStatus != TransactionConfirmationStatusConfirmed && confirmationStatus != TransactionConfirmationStatusFinalized {
						return nil, tErr
					}
				} else if commitmentForStatus == CommitmentConfirmed || commitmentForStatus == CommitmentSingleGossip || commitmentForStatus == CommitmentSingle {
					if confirmationStatus != TransactionConfirmationStatusConfirmed && confirmationStatus != TransactionConfirmationStatusFinalized {
						return nil, tErr
					}
				} else if commitmentForStatus == CommitmentFinalized || commitmentForStatus == CommitmentMax || commitmentForStatus == CommitmentRoot {
					if confirmationStatus != TransactionConfirmationStatusFinalized {
						return nil, tErr
					}
				}
				return &RpcResponseAndContext[SignatureResult]{
					Context: signatureStatus.Context,
					Value: SignatureResult{
						Err: signatureStatus.Value.Err,
					},
				}, nil
			} else {
				return nil, tErr
			}
		}
	}
}

// GetBlockHeightConfig is the configuration object for changing `getBlockHeight` query behavior.
type GetBlockHeightConfig = BaseConfig

func (c *Connection) GetBlockHeight(config GetBlockHeightConfig) (uint64, error) {
	args := c.buildArgs(nil, config.Commitment, nil, config)
	return requestNonContextValue[uint64](context.Background(), c, "getBlockHeight", args, "failed to get block height information")
}

// SignatureStatusConfig Configuration object for changing query behavior
type SignatureStatusConfig struct {
	// enable searching status history, not needed for recent transactions
	SearchTransactionHistory bool `json:"searchTransactionHistory,omitempty"`
}

type TransactionError = any

type TransactionConfirmationStatus string

const (
	TransactionConfirmationStatusProcessed TransactionConfirmationStatus = "processed"
	TransactionConfirmationStatusConfirmed TransactionConfirmationStatus = "confirmed"
	TransactionConfirmationStatusFinalized TransactionConfirmationStatus = "finalized"
)

// SignatureStatus represents the signature status.
type SignatureStatus struct {
	// When the transaction was processed
	Slot uint64 `json:"slot"`
	// The number of blocks that have been confirmed and voted on in the fork containing `Slot`
	Confirmations uint64 `json:"confirmations"`
	// Transaction error, if any
	Err *TransactionError `json:"err"`
	// Cluster confirmation status, if data available.
	// Possible responses: `processed`, `confirmed`, `finalized`
	ConfirmationStatus TransactionConfirmationStatus `json:"confirmationStatus,omitempty"`
}

func (c *Connection) GetSignatureStatus(signature TransactionSignature, config SignatureStatusConfig) (*RpcResponseAndContext[SignatureStatus], error) {
	statuses, err := c.GetSignatureStatuses([]TransactionSignature{signature}, config)
	if err != nil {
		return nil, err
	}
	if len(statuses.Value) != 1 {
		return nil, errors.New("assert statuses.Value length == 1")
	}
	return &RpcResponseAndContext[SignatureStatus]{
		Context: statuses.Context,
		Value:   statuses.Value[0],
	}, nil
}

func (c *Connection) GetSignatureStatuses(signatures []TransactionSignature, config SignatureStatusConfig) (*RpcResponseAndContext[[]SignatureStatus], error) {
	var args = []any{signatures}
	if config.SearchTransactionHistory {
		args = append(args, utils.StructToMap(config))
	}
	return requestContext[[]SignatureStatus](context.Background(), c, "getSignatureStatuses", args, "failed to get signature status")
}

// ContactInfo represents information describing a cluster node.
type ContactInfo struct {
	// Identity public key of the node
	Pubkey string `json:"pubkey"`
	// Gossip network address for the node
	Gossip string `json:"gossip,omitempty"`
	// TPU network address for the node (null if not available)
	TPU string `json:"tpu,omitempty"`
	// JSON RPC network address for the node (null if not available)
	RPC string `json:"rpc,omitempty"`
	// Software version of the node (null if not available)
	Version string `json:"version,omitempty"`
}

// GetClusterNodes Return the list of nodes that are currently participating in the cluster
func (c *Connection) GetClusterNodes() ([]ContactInfo, error) {
	return requestNonContextValue[[]ContactInfo](context.Background(), c, "getClusterNodes", nil, "failed to get cluster nodes")
}

// VoteAccountStatus A collection of cluster vote accounts
type VoteAccountStatus struct {
	// Active vote accounts
	Current []VoteAccountInfo `json:"current,omitempty"`
	// Inactive vote accounts
	Delinquent []VoteAccountInfo `json:"delinquent,omitempty"`
}

// VoteAccountInfo represents information describing a vote account.
type VoteAccountInfo struct {
	// Public key of the vote account
	VotePubkey string `json:"votePubkey"`
	// Identity public key of the node voting with this account
	NodePubkey string `json:"nodePubkey"`
	// The stake, in lamports, delegated to this vote account and activated
	ActivatedStake int `json:"activatedStake"`
	// Whether the vote account is staked for this epoch
	EpochVoteAccount bool `json:"epochVoteAccount"`
	// Recent epoch voting credit history for this voter
	EpochCredits [][3]int `json:"epochCredits"`
	// A percentage (0-100) of rewards payout owed to the voter
	Commission int `json:"commission"`
	// Most recent slot voted on by this vote account
	LastVote int `json:"lastVote"`
}

// GetVoteAccounts Return the list of nodes that are currently participating in the cluster
func (c *Connection) GetVoteAccounts(commitment *Commitment) (VoteAccountStatus, error) {
	args := c.buildArgs(nil, commitment, nil, nil)
	return requestNonContextValue[VoteAccountStatus](context.Background(), c, "getVoteAccounts", args, "failed to get vote accounts")
}

type GetSlotConfig = BaseConfig

// GetSlot Fetch the current slot that the node is processing
func (c *Connection) GetSlot(config GetSlotConfig) (uint64, error) {
	args := c.buildArgs(nil, config.Commitment, nil, config)
	return requestNonContextValue[uint64](context.Background(), c, "getSlot", args, "failed to get slot")
}

type GetSlotLeaderConfig = BaseConfig

// GetSlotLeader Fetch the current slot leader of the cluster
func (c *Connection) GetSlotLeader(config GetSlotLeaderConfig) (PublicKey, error) {
	args := c.buildArgs(nil, config.Commitment, nil, config)
	return requestNonContextValue[PublicKey](context.Background(), c, "getSlotLeader", args, "failed to get slot leader")
}

// GetSlotLeaders Fetch `limit` number of slot leaders starting from `startSlot`
// @param startSlot fetch slot leaders starting from this slot
// @param limit number of slot leaders to return
func (c *Connection) GetSlotLeaders(startSlot uint64, limit uint64) ([]PublicKey, error) {
	return requestNonContextValue[[]PublicKey](context.Background(), c, "getSlotLeaders", []any{startSlot, limit}, "failed to get slot leaders")
}

type BaseConfig struct {
	Commitment *Commitment
	// The minimum slot that the request can be evaluated at
	MinContextSlot uint64 `json:"minContextSlot,omitempty"`
}

type GetTransactionCountConfig = BaseConfig

// GetTransactionCount Fetch the current transaction count of the cluster
func (c *Connection) GetTransactionCount(config GetTransactionCountConfig) (uint64, error) {
	args := c.buildArgs(nil, config.Commitment, nil, config)
	return requestNonContextValue[uint64](context.Background(), c, "getTransactionCount", args, "failed to get transaction count")
}

// GetTotalSupply Fetch the current total currency supply of the cluster in lamports
// Deprecated: since v1.2.8. Please use {@link getSupply} instead.
func (c *Connection) GetTotalSupply(commitment *Commitment) (uint64, error) {
	ret, err := c.GetSupply(GetSupplyConfig{
		commitment,
		true,
	})
	if err != nil {
		return 0, err
	}
	return ret.Value.Total, nil
}

// InflationGovernor Network Inflation
// (see https://docs.solana.com/implemented-proposals/ed_overview)
type InflationGovernor struct {
	Foundation     float64 `json:"foundation,omitempty"`
	FoundationTerm float64 `json:"foundationTerm,omitempty"`
	Initial        float64 `json:"initial,omitempty"`
	Taper          float64 `json:"taper,omitempty"`
	Terminal       float64 `json:"terminal,omitempty"`
}

// GetInflationGovernor Fetch the cluster InflationGovernor parameters
func (c *Connection) GetInflationGovernor(commitment *Commitment) (*InflationGovernor, error) {
	args := c.buildArgs(nil, commitment, nil, nil)
	return requestNonContext[InflationGovernor](context.Background(), c, "getInflationGovernor", args, "failed to get inflation")
}

// GetInflationRewardConfig is the configuration object for changing `getInflationReward` query behavior.
type GetInflationRewardConfig struct {
	// The level of commitment desired
	Commitment *Commitment
	// An epoch for which the reward occurs. If omitted, the previous epoch will be used
	Epoch uint64 `json:"epoch,omitempty"`
	// The minimum slot that the request can be evaluated at
	MinContextSlot uint64 `json:"minContextSlot,omitempty"`
}

// InflationReward represents the inflation reward for an epoch.
type InflationReward struct {
	// Epoch for which the reward occurs
	Epoch uint64 `json:"epoch"`
	// The slot in which the rewards are effective
	EffectiveSlot uint64 `json:"effectiveSlot"`
	// Reward amount in lamports
	Amount uint64 `json:"amount"`
	// Post balance of the account in lamports
	PostBalance uint64 `json:"postBalance"`
	// Vote account commission when the reward was credited
	Commission *uint64 `json:"commission,omitempty"`
}

// GetInflationReward Fetch the inflation reward for a list of addresses for an epoch
func (c *Connection) GetInflationReward(addresses []PublicKey, config GetInflationRewardConfig) (InflationReward, error) {
	args := c.buildArgs([]any{utils.Map(addresses, func(t PublicKey) string {
		return t.Base58()
	})}, config.Commitment, nil, config)
	return requestNonContextValue[InflationReward](context.Background(), c, "getInflationReward", args, "failed to get inflation reward")
}

// InflationRate represents the inflation rate for an epoch.
type InflationRate struct {
	// Total inflation
	Total float64 `json:"total"`
	// Inflation allocated to validators
	Validator float64 `json:"validator"`
	// Inflation allocated to the foundation
	Foundation float64 `json:"foundation"`
	// Epoch for which these values are valid
	Epoch uint64 `json:"epoch"`
}

// GetInflationRate Fetch the specific inflation values for the current epoch
func (c *Connection) GetInflationRate() (InflationRate, error) {
	return requestNonContextValue[InflationRate](context.Background(), c, "getInflationRate", nil, "failed to get inflation rate")
}

type GetEpochInfoConfig = BaseConfig

// EpochInfo represents information about the current epoch.
type EpochInfo struct {
	// Epoch number
	Epoch uint64 `json:"epoch"`
	// Slot index within the epoch
	SlotIndex uint64 `json:"slotIndex"`
	// Number of slots in the epoch
	SlotsInEpoch uint64 `json:"slotsInEpoch"`
	// Absolute slot number
	AbsoluteSlot uint64 `json:"absoluteSlot"`
	// Block height (optional)
	BlockHeight *uint64 `json:"blockHeight,omitempty"`
	// Transaction count (optional)
	TransactionCount *uint64 `json:"transactionCount,omitempty"`
}

// GetEpochInfo Fetch the Epoch Info parameters
func (c *Connection) GetEpochInfo(config GetEpochInfoConfig) (EpochInfo, error) {
	args := c.buildArgs(nil, config.Commitment, nil, config)
	return requestNonContextValue[EpochInfo](context.Background(), c, "getEpochInfo", args, "failed to get epoch info")
}

// GetEpochSchedule Fetch the Epoch Schedule parameters
func (c *Connection) GetEpochSchedule() (EpochSchedule, error) {
	return requestNonContextValue[EpochSchedule](context.Background(), c, "getEpochSchedule", nil, "failed to get epoch schedule")
}

// LeaderSchedule represents the leader schedule.
type LeaderSchedule map[string][]int

// GetLeaderSchedule Fetch the leader schedule for the current epoch
func (c *Connection) GetLeaderSchedule() (LeaderSchedule, error) {
	return requestNonContextValue[LeaderSchedule](context.Background(), c, "getLeaderSchedule", nil, "failed to get leader schedule")
}

// GetMinimumBalanceForRentExemption Fetch the minimum balance needed to exempt an account
// of `dataLength` size from rent
func (c *Connection) GetMinimumBalanceForRentExemption(dataLength int, commitment *Commitment) (uint64, error) {
	args := c.buildArgs([]any{dataLength}, commitment, nil, nil)
	return requestNonContextValue[uint64](context.Background(), c, "getMinimumBalanceForRentExemption", args, "Unable to fetch minimum balance for rent exemption")
}

// PerfSample represents a performance sample.
type PerfSample struct {
	// Slot number of the sample
	Slot uint64 `json:"slot"`
	// Number of transactions in a sample window
	NumTransactions uint64 `json:"numTransactions"`
	// Number of slots in a sample window
	NumSlots uint64 `json:"numSlots"`
	// Sample window in seconds
	SamplePeriodSecs uint64 `json:"samplePeriodSecs"`
}

// GetRecentPerformanceSamples Fetch recent performance samples
func (c *Connection) GetRecentPerformanceSamples(limit int) ([]PerfSample, error) {
	var args []any
	if limit > 0 {
		args = append(args, limit)
	}
	return requestNonContextValue[[]PerfSample](context.Background(), c, "getRecentPerformanceSamples", args, "failed to get recent performance samples")
}

// GetFeeForMessage Fetch the fee for a message from the cluster, return with context
func (c *Connection) GetFeeForMessage(message VersionedMessage, commitment *Commitment) (*RpcResponseAndContext[*uint64], error) {
	wireMessage := base64.StdEncoding.EncodeToString(message.Serialize())
	args := c.buildArgs([]any{wireMessage}, commitment, nil, nil)
	return requestContext[*uint64](context.Background(), c, "getFeeForMessage", args, "failed to get fee for message")
}

// GetRecentPrioritizationFeesConfig is the configuration object for changing `getRecentPrioritizationFees` query behavior.
type GetRecentPrioritizationFeesConfig struct {
	// If this parameter is provided, the response will reflect a fee to land a transaction locking
	// all of the provided accounts as writable.
	LockedWritableAccounts []PublicKey `json:"lockedWritableAccounts,omitempty"`
}

// RecentPrioritizationFees represents recent prioritization fees.
type RecentPrioritizationFees struct {
	// Slot in which the fee was observed
	Slot uint64 `json:"slot"`
	// The per-compute-unit fee paid by at least one successfully landed transaction,
	// specified in increments of 0.000001 lamports
	PrioritizationFee float64 `json:"prioritizationFee"`
}

// GetRecentPrioritizationFees Fetch a list of prioritization fees from recent blocks.
func (c *Connection) GetRecentPrioritizationFees(config GetRecentPrioritizationFeesConfig) ([]RecentPrioritizationFees, error) {
	accounts := utils.Map(config.LockedWritableAccounts, func(t PublicKey) string {
		return t.Base58()
	})
	var args []any
	if len(accounts) > 0 {
		args = append(args, accounts)
	}
	return requestNonContextValue[[]RecentPrioritizationFees](context.Background(), c, "getRecentPrioritizationFees", args, "failed to get recent prioritization fees")
}

type GetLatestBlockhashConfig = BaseConfig

// GetLatestBlockhash Fetch the latest blockhash from the cluster
func (c *Connection) GetLatestBlockhash(config GetLatestBlockhashConfig) (BlockhashWithExpiryBlockHeight, error) {
	res, err := c.GetLatestBlockhashAndContext(config)
	if err != nil {
		return BlockhashWithExpiryBlockHeight{}, err
	}
	return res.Value, nil
}

// GetLatestBlockhashAndContext Fetch the latest blockhash from the cluster
func (c *Connection) GetLatestBlockhashAndContext(config GetLatestBlockhashConfig) (*RpcResponseAndContext[BlockhashWithExpiryBlockHeight], error) {
	args := c.buildArgs(nil, config.Commitment, nil, config)
	return requestContext[BlockhashWithExpiryBlockHeight](context.Background(), c, "getLatestBlockhash", args, "failed to get latest blockhash")
}

type IsBlockhashValidConfig = BaseConfig

// IsBlockhashValid Returns whether a blockhash is still valid or not
func (c *Connection) IsBlockhashValid(blockhash Blockhash, config IsBlockhashValidConfig) (*RpcResponseAndContext[bool], error) {
	args := c.buildArgs([]any{blockhash}, config.Commitment, nil, config)
	return requestContext[bool](context.Background(), c, "isBlockhashValid", args, msg("failed to determine if the blockhash `%s` is valid", blockhash))
}

// Version represents version info for a node.
type Version struct {
	// Version of solana-core
	SolanaCore string `json:"solana-core"`
	// Feature set (optional)
	FeatureSet *int `json:"feature-set,omitempty"`
}

// GetVersion Fetch the node version
func (c *Connection) GetVersion() (Version, error) {
	return requestNonContextValue[Version](context.Background(), c, "getVersion", nil, "failed to get version")
}

// GetGenesisHash Fetch the genesis hash
func (c *Connection) GetGenesisHash() (Blockhash, error) {
	return requestNonContextValue[Blockhash](context.Background(), c, "getGenesisHash", nil, "failed to get genesis hash")
}

// GetBlockProductionConfig is the configuration object for changing `getBlockProduction` query behavior.
type GetBlockProductionConfig struct {
	// Optional commitment level
	Commitment *Commitment
	// Slot range to return block production for. If parameter not provided, defaults to current epoch.
	Range *struct {
		// First slot to return block production information for (inclusive)
		FirstSlot int `json:"firstSlot"`
		// Last slot to return block production information for (inclusive). If parameter not provided, defaults to the highest slot
		LastSlot *int `json:"lastSlot,omitempty"`
	} `json:"range,omitempty"`
	// Only return results for this validator identity (base-58 encoded)
	Identity string `json:"identity,omitempty"`
}

// BlockProduction represents recent block production information.
type BlockProduction struct {
	// A dictionary of validator identities, as base-58 encoded strings.
	// Value is a two-element array containing the number of leader slots and the number of blocks produced.
	ByIdentity map[string][2]int `json:"byIdentity"`
	// Block production slot range
	Range struct {
		// First slot to return block production information for (inclusive)
		FirstSlot int `json:"firstSlot"`
		// Last slot to return block production information for (inclusive). If parameter not provided, defaults to the highest slot
		LastSlot *int `json:"lastSlot,omitempty"`
	} `json:"range"`
}

// GetBlockProduction Returns recent block production information from the current or previous epoch
func (c *Connection) GetBlockProduction(config GetBlockProductionConfig) (*RpcResponseAndContext[BlockProduction], error) {
	args := c.buildArgs(nil, config.Commitment, &EncodingBase64, config)
	return requestContext[BlockProduction](context.Background(), c, "getBlockProduction", args, "failed to get block production information")
}

// RequestAirdrop Request an allocation of lamports to the specified address
func (c *Connection) RequestAirdrop(to PublicKey, lamports uint64) (TransactionSignature, error) {
	return requestNonContextValue[TransactionSignature](context.Background(), c, "requestAirdrop", []any{to.Base58(), lamports}, msg("airdrop to %s failed", to))
}

func Ref[T any](input T) *T {
	return &input
}

// GetBlocks Fetch confirmed blocks between two slots [startSlot, endSlot]
// commitment: "confirmed" or "finalized"
// returns an array of slots which contain a block
func (c *Connection) GetBlocks(startSlot uint64, endSlot *uint64, commitment *Commitment) ([]uint64, error) {
	var _args []any
	if endSlot != nil {
		_args = []any{startSlot, *endSlot}
	} else {
		_args = []any{startSlot}
	}
	args, err := c.buildArgsAtLeastConfirmed(_args, commitment, nil, nil)
	if err != nil {
		return nil, err
	}
	return requestNonContextValue[[]uint64](context.Background(), c, "getBlocks", args, "failed to get blocks")
}

// BlockSignatures represents a block on the ledger with signatures only.
type BlockSignatures struct {
	// Blockhash of this block
	Blockhash Blockhash `json:"blockhash"`
	// Blockhash of this block's parent
	PreviousBlockhash Blockhash `json:"previousBlockhash"`
	// Slot index of this block's parent
	ParentSlot uint64 `json:"parentSlot"`
	// Vector of signatures
	Signatures []string `json:"signatures"`
	// The unix timestamp of when the block was processed (nullable)
	BlockTime int64 `json:"blockTime,omitempty"`
}

// GetBlockSignatures Fetch a list of signatures from the cluster for a block, excluding rewards
// commitment: "confirmed" or "finalized"
func (c *Connection) GetBlockSignatures(slot uint64, commitment *Commitment) (*BlockSignatures, error) {
	args, err := c.buildArgsAtLeastConfirmed([]any{slot}, commitment, nil, map[string]any{
		"transactionDetails": TransactionDetail_Signatures,
		"rewards":            false,
	})
	if err != nil {
		return nil, err
	}
	return requestNonContext[BlockSignatures](context.Background(), c, "getBlock", args, "failed to get block")
}

// GetConfirmedBlockSignatures Fetch a list of signatures from the cluster for a confirmed block, excluding rewards
// Deprecated: since Solana v1.8.0. Please use {@link getBlockSignatures} instead.
func (c *Connection) GetConfirmedBlockSignatures(slot uint64, commitment *Commitment) (*BlockSignatures, error) {
	args, err := c.buildArgsAtLeastConfirmed([]any{slot}, commitment, nil, map[string]any{
		"transactionDetails": "signatures",
		"rewards":            false,
	})
	if err != nil {
		return nil, err
	}
	return requestNonContext[BlockSignatures](context.Background(), c, "getConfirmedBlock", args, msg("confirmed block %d not found", slot))
}

// SignaturesForAddressOptions is the options for getSignaturesForAddress.
type SignaturesForAddressOptions struct {
	// Start searching backwards from this transaction signature.
	// If not provided, the search starts from the highest max confirmed block.
	Before TransactionSignature `json:"before,omitempty"`
	// Search until this transaction signature is reached, if found before `limit`.
	Until TransactionSignature `json:"until,omitempty"`
	// Maximum transaction signatures to return (between 1 and 1,000, default: 1,000).
	Limit int `json:"limit,omitempty"`
	// The minimum slot that the request can be evaluated at.
	MinContextSlot uint64 `json:"minContextSlot,omitempty"`
}

// ConfirmedSignatureInfo represents a confirmed signature with its status.
type ConfirmedSignatureInfo struct {
	// The transaction signature
	Signature string `json:"signature"`
	// When the transaction was processed
	Slot int `json:"slot"`
	// Error, if any
	Err TransactionError `json:"err"`
	// Memo associated with the transaction, if any
	Memo string `json:"memo,omitempty"`
	// The Unix timestamp of when the transaction was processed (nullable)
	BlockTime *uint64 `json:"blockTime,omitempty"`
	// Cluster confirmation status, if available. Possible values: `processed`, `confirmed`, `finalized`
	ConfirmationStatus TransactionConfirmationStatus `json:"confirmationStatus,omitempty"`
}

// GetSignaturesForAddress Returns confirmed signatures for transactions involving an
// address backwards in time from the provided signature or most recent confirmed block
// commitment: "confirmed" or "finalized"
func (c *Connection) GetSignaturesForAddress(address PublicKey, options SignaturesForAddressOptions, commitment *Commitment) (*ConfirmedSignatureInfo, error) {
	if options.Limit == 0 {
		options.Limit = 1000
	}
	args, err := c.buildArgsAtLeastConfirmed(
		[]any{address.Base58()},
		commitment,
		nil,
		options,
	)
	if err != nil {
		return nil, err
	}
	return requestNonContext[ConfirmedSignatureInfo](context.Background(), c, "getSignaturesForAddress", args, "failed to get signatures for address")
}

// SendTransaction Sign and send a transaction
func (c *Connection) SendTransaction(
	transaction any,
	signers []Signer,
	options SendOptions,
) (TransactionSignature, error) {
	if v, ok := transaction.(*VersionedTransaction); ok {
		if len(signers) > 0 {
			return "", errors.New("invalid arguments")
		}
		return c.SendRawTransaction(v.Serialize(), options)
	} else if v, ok := transaction.(*Transaction); ok {
		if v.NonceInfo != nil {
			if len(signers) > 0 {
				if err := v.Sign(signers...); err != nil {
					return "", err
				}
			} else {
				if v.Signature().IsZero() {
					return "", errors.New("!signature") // should never happen
				}
			}
		} else {
			disableCache := c.disableBlockhashCaching
			for {
				if len(signers) == 0 {
					if v.Signature().IsZero() {
						return "", errors.New("!signature") // should never happen
					}
					break
				}
				latestBlockhash, err := c._blockhashWithExpiryBlockHeight(disableCache)
				if err != nil {
					return "", err
				}
				v.LastValidBlockHeight = &latestBlockhash.LastValidBlockHeight
				v.RecentBlockhash = &latestBlockhash.Blockhash
				if err := v.Sign(signers...); err != nil {
					return "", err
				}
				if v.Signature().IsZero() {
					return "", errors.New("!signature") // should never happen
				}

				var s = v.Signature()
				signature := base64.StdEncoding.EncodeToString(s[:])
				if !utils.Contain(c.blockhashInfo.TransactionSignatures, signature) {
					// The signature of this transaction has not been seen before with the
					// current recentBlockhash, all done. Let's break
					c.blockhashInfo.TransactionSignatures = append(c.blockhashInfo.TransactionSignatures, signature)
					break
				} else {
					// This transaction would be treated as duplicate (its derived signature
					// matched to one of already recorded signatures).
					// So, we must fetch a new blockhash for a different signature by disabling
					// our cache not to wait for the cache expiration (BLOCKHASH_CACHE_TIMEOUT_MS).
					disableCache = true
				}
			}
		}
		wireTransaction, err := v.Serialize()
		if err != nil {
			return "", err
		}
		return c.SendRawTransaction(wireTransaction, options)
	} else {
		return "", errors.New("invalid transaction")
	}
}

type GetNonceConfig = BaseConfig
type GetNonceAndContextConfig = BaseConfig

// GetNonce Fetch the contents of a Nonce account from the cluster
func (c *Connection) GetNonce(nonceAccount PublicKey, config GetNonceConfig) (*system.NonceAccount, error) {
	ctx, err := c.GetNonceAndContext(nonceAccount, config)
	if err != nil {
		return nil, err
	}
	return ctx.Value, nil
}

// GetNonceAndContext Fetch the contents of a Nonce account from the cluster, return with context
func (c *Connection) GetNonceAndContext(
	nonceAccount PublicKey,
	config GetNonceAndContextConfig,
) (*RpcResponseAndContext[*NonceAccount], error) {
	resp, err := c.GetAccountInfoAndContext(nonceAccount, GetAccountInfoConfig{
		Commitment:     config.Commitment,
		MinContextSlot: config.MinContextSlot,
	})
	if err != nil {
		return nil, err
	}
	var value *NonceAccount
	if resp.Value != nil {
		value, err = NonceAccountFromAccountData(resp.Value.Data.Content)
		if err != nil {
			return nil, err
		}
	}
	return &RpcResponseAndContext[*NonceAccount]{
		Context: resp.Context,
		Value:   value,
	}, nil
}

// Attempt to use a recent blockhash for up to 30 seconds
const blockhashCacheTimeoutMs = 30 * 1000

func (c *Connection) _blockhashWithExpiryBlockHeight(disableCache bool) (*BlockhashWithExpiryBlockHeight, error) {
	if !disableCache {
		// Wait for polling to finish
		for c.pollingBlockhash {
			time.Sleep(100 * time.Millisecond)
		}
		timeSinceFetch := uint64(time.Now().UnixMilli()) - c.blockhashInfo.LastFetch
		expired := timeSinceFetch >= blockhashCacheTimeoutMs
		if c.blockhashInfo.LatestBlockhash != nil && !expired {
			return c.blockhashInfo.LatestBlockhash, nil
		}
	}

	return c._pollNewBlockhash()
}

func (c *Connection) _pollNewBlockhash() (*BlockhashWithExpiryBlockHeight, error) {
	c.pollingBlockhash = true
	defer func() {
		c.pollingBlockhash = false
	}()
	startTime := time.Now().UnixMilli()
	cachedLatestBlockhash := c.blockhashInfo.LatestBlockhash
	var cachedBlockhash *Blockhash
	if cachedLatestBlockhash != nil {
		cachedBlockhash = &cachedLatestBlockhash.Blockhash
	} else {
		cachedBlockhash = nil
	}
	for i := 0; i < 50; i++ {
		latestBlockhash, err := c.GetLatestBlockhash(GetLatestBlockhashConfig{
			Commitment: &CommitmentFinalized,
		})
		if err != nil {
			return nil, err
		}
		if cachedBlockhash == nil || (cachedBlockhash != nil && *cachedBlockhash != latestBlockhash.Blockhash) {
			c.blockhashInfo = struct {
				LatestBlockhash       *BlockhashWithExpiryBlockHeight
				LastFetch             uint64
				SimulatedSignatures   []string
				TransactionSignatures []string
			}{LatestBlockhash: &latestBlockhash, LastFetch: uint64(time.Now().UnixMilli()), SimulatedSignatures: nil, TransactionSignatures: nil}
			return &latestBlockhash, nil
		}

		// Sleep for approximately half a slot
		time.Sleep((MS_PER_SLOT / 2) * time.Millisecond)
	}
	return nil, fmt.Errorf("unable to obtain a new blockhash after %dms", time.Now().UnixMilli()-startTime)
}

// SendEncodedTransaction Send a transaction that has already been signed, serialized into the
// wire format, and encoded as a base64 string
func (c *Connection) SendEncodedTransaction(encodedTransaction string, options SendOptions) (TransactionSignature, error) {
	if options.PreflightCommitment == nil {
		options.PreflightCommitment = c.Commitment()
	}
	v := utils.StructToMap(options)
	v["encoding"] = "base64"
	args := []any{encodedTransaction, v}

	res, err := requestNonContextValue[TransactionSignature](context.Background(), c, "sendTransaction", args, "")
	if err != nil {
		var v SolanaJSONRPCError
		if errors.As(err, &v) {
			var d struct {
				Logs []string `json:"logs"`
			}
			_ = json.Unmarshal(v.Err.Data, &d)
			return "", NewSendTransactionError(err.Error(), d.Logs, v.Err.Code)
		}
		return "", err
	}
	return res, nil
}

// SendRawTransaction Send a transaction that has already been signed and serialized into the wire format
func (c *Connection) SendRawTransaction(rawTransaction []byte, options SendOptions) (TransactionSignature, error) {
	return c.SendEncodedTransaction(base64.StdEncoding.EncodeToString(rawTransaction), options)
}

func (c *Connection) GetAddressLookupTable(accountKey PublicKey, config GetAccountInfoConfig) (*RpcResponseAndContext[*AddressLookupTableAccount], error) {
	resp, err := c.GetAccountInfoAndContext(accountKey, config)
	if err != nil {
		return nil, err
	}
	var value *AddressLookupTableAccount
	if resp.Value != nil {
		var decoder = binary.NewDecoderWithEncoding(resp.Value.Data.Content, binary.EncodingBin)
		var state AddressLookupTableState
		if err = state.UnmarshalWithDecoder(decoder); err != nil {
			return nil, fmt.Errorf("[GetAddressLookupTable] UnmarshalWithDecoder: %w", err)
		}
		value = &AddressLookupTableAccount{
			Key:   accountKey,
			State: state,
		}
	}
	return &RpcResponseAndContext[*AddressLookupTableAccount]{
		Context: resp.Context,
		Value:   value,
	}, nil
}

// GetVersionedTransactionConfig Configuration object for changing `getTransaction` query behavior
type GetVersionedTransactionConfig struct {
	// The level of finality desired
	Commitment *Commitment
	// The max transaction version to return in responses. If the requested transaction is a higher version, an error will be returned
	MaxSupportedTransactionVersion int `json:"maxSupportedTransactionVersion"`
}

// VersionedTransactionResponse A processed transaction from the RPC API
type VersionedTransactionResponse struct {
	// the slot during which the transaction was processed
	Slot uint64 `json:"slot,omitempty"`
	// the transaction
	Transaction VersionedTransactionRet `json:"transaction"`
	// Metadata produced from the transaction
	Meta *ConfirmedTransactionMeta `json:"meta,omitempty"`
	// The unix timestamp of when the transaction was processed
	BlockTime *uint64 `json:"blockTime,omitempty"`
	// The transaction version
	Version *TransactionVersion `json:"version,omitempty"`
}

func (v *VersionedTransactionResponse) UnmarshalJSON(data []byte) error {
	type Tmp = struct {
		// the slot during which the transaction was processed
		Slot uint64 `json:"slot,omitempty"`
		// the transaction
		Transaction struct {
			Message    json.RawMessage `json:"message,omitempty"`
			Signatures []string        `json:"signatures,omitempty"`
		} `json:"transaction"`
		// Metadata produced from the transaction
		Meta *ConfirmedTransactionMeta `json:"meta,omitempty"`
		// The unix timestamp of when the transaction was processed
		BlockTime *uint64 `json:"blockTime,omitempty"`
		// The transaction version
		Version *TransactionVersion `json:"version,omitempty"`
	}
	var tmp Tmp
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	var version0 = false
	if tmp.Version != nil {
		if *tmp.Version == TransactionVersion0 {
			version0 = true
		}
	}
	v2 := VersionedTransactionResponse{
		Slot: tmp.Slot,
		Transaction: VersionedTransactionRet{
			Signatures: tmp.Transaction.Signatures,
		},
		Meta:      tmp.Meta,
		BlockTime: tmp.BlockTime,
		Version:   tmp.Version,
	}
	if !version0 {
		var message Message
		if err := json.Unmarshal(tmp.Transaction.Message, &message); err != nil {
			return err
		}
		v2.Transaction.Message = VersionedMessage{message}
	} else {
		var message MessageV0
		if err := json.Unmarshal(tmp.Transaction.Message, &message); err != nil {
			return err
		}
		v2.Transaction.Message = VersionedMessage{message}
	}
	*v = v2
	return nil
}

type CompiledInnerInstruction struct {
	Index        uint64                `json:"index,omitempty"`
	Instructions []CompiledInstruction `json:"instructions"`
}

type TokenBalance struct {
	AccountIndex  uint64       `json:"accountIndex,omitempty"`
	Mint          PublicKey    `json:"mint,omitempty"`
	Owner         PublicKey    `json:"owner,omitempty"`
	UiTokenAmount *TokenAmount `json:"uiTokenAmount,omitempty"`
}

// ConfirmedTransactionMeta Metadata for a confirmed transaction on the ledger
type ConfirmedTransactionMeta struct {
	// The fee charged for processing the transaction
	Fee uint64 `json:"fee,omitempty"`
	// An array of cross program invoked parsed instructions
	InnerInstructions []CompiledInnerInstruction `json:"innerInstructions,omitempty"`
	// The balances of the transaction accounts before processing
	PreBalances []uint64 `json:"preBalances,omitempty"`
	// The balances of the transaction accounts after processing
	PostBalances []int64 `json:"postBalances,omitempty"`
	// An array of program log messages emitted during a transaction
	LogMessages []string `json:"logMessages,omitempty"`
	// The token balances of the transaction accounts before processing
	PreTokenBalances []TokenBalance `json:"preTokenBalances,omitempty"`
	// The token balances of the transaction accounts after processing
	PostTokenBalances []TokenBalance `json:"postTokenBalances,omitempty"`
	// The error result of transaction processing
	Err TransactionError `json:"err,omitempty"`
	// The collection of addresses loaded using address lookup tables
	LoadedAddresses *LoadedAddresses `json:"loadedAddresses,omitempty"`
	// The compute units consumed after processing the transaction
	ComputeUnitsConsumed *uint64 `json:"computeUnitsConsumed,omitempty"`
}

// GetTransaction Fetch a confirmed or finalized transaction from the cluster.
func (c *Connection) GetTransaction(signature string, config GetVersionedTransactionConfig) (*VersionedTransactionResponse, error) {
	args, err := c.buildArgsAtLeastConfirmed([]any{signature}, config.Commitment, nil, config)
	if err != nil {
		return nil, err
	}
	value, err := requestNonContext[VersionedTransactionResponse](context.Background(), c, "getTransaction", args, "failed to get transaction")
	if err != nil {
		return nil, err
	}
	return value, nil
}

type GetStakeMinimumDelegationConfig struct {
	Commitment *Commitment
}

// GetStakeMinimumDelegation get the stake minimum delegation
func (c *Connection) GetStakeMinimumDelegation(config GetStakeMinimumDelegationConfig) (*RpcResponseAndContext[uint64], error) {
	args := c.buildArgs(nil, config.Commitment, &EncodingBase64, config)
	return requestContext[uint64](context.Background(), c, "getStakeMinimumDelegation", args, "failed to get stake minimum delegation")
}

func requestContextValue[T any](ctx context.Context, connection *Connection, method string, args []any, customErrMessage string) (de T, err error) {
	res, err := requestContext[T](ctx, connection, method, args, customErrMessage)
	if err != nil {
		return
	}
	return res.Value, nil
}

func requestContext[T any](ctx context.Context, connection *Connection, methodName string, args []any, customErrMessage string) (*RpcResponseAndContext[T], error) {
	unsafeRes, err := connection.rpcRequest(ctx, methodName, args)
	if err != nil {
		return nil, err
	}
	return createContext[T](unsafeRes, customErrMessage)
}

func requestNonContextValue[T any](ctx context.Context, connection *Connection, method string, args []any, customErrMessage string) (de T, err error) {
	res, err := requestNonContext[T](ctx, connection, method, args, customErrMessage)
	if err != nil {
		return
	}
	if res == nil {
		return de, nil
	}
	return *res, nil
}

func requestNonContext[T any](ctx context.Context, connection *Connection, method string, args []any, customErrMessage string) (de *T, err error) {
	unsafeRes, err := connection.rpcRequest(ctx, method, args)
	if err != nil {
		return
	}
	return createNonContext[T](unsafeRes, customErrMessage)
}

func createContext[T any](input string, customErrMessage string) (*RpcResponseAndContext[T], error) {
	res, err := create[RpcResponse[T]](input, customErrMessage)
	if err != nil {
		return nil, err
	}
	return &res.Result, nil
}

func createNonContext[T any](input string, customErrMessage string) (*T, error) {
	res, err := create[SlotRpcResult[T]](input, customErrMessage)
	if err != nil {
		return nil, err
	}
	if res.resultIsNil {
		return nil, nil
	}
	return &res.Result, nil
}

func create[T any](input string, customErrMessage string) (response T, err error) {
	err = json.Unmarshal([]byte(input), &response)
	if err != nil {
		return
	}
	errorField, found := reflect.TypeOf(response).FieldByName("Error")
	if found && errorField.Type == reflect.TypeOf((*RpcResponseError)(nil)) {
		errorFieldValue := reflect.ValueOf(response).FieldByName("Error").Interface()
		respErr := errorFieldValue.(*RpcResponseError)
		if respErr != nil {
			return response, SolanaJSONRPCError{
				*respErr,
				customErrMessage,
			}
		}
	}
	return response, nil
}

type CustomClient struct {
	*http.Client
	url                     string
	disableRetryOnRateLimit bool
	headers                 map[string]string
}

func NewCustomClient(endpoint string, httpHeaders map[string]string, disableRetryOnRateLimit bool) *CustomClient {
	return &CustomClient{
		Client:                  &http.Client{},
		url:                     endpoint,
		headers:                 httpHeaders,
		disableRetryOnRateLimit: disableRetryOnRateLimit,
	}
}

var CommonHTTPHeaders = map[string]string{
	"solana-client": "js/UNKNOWN",
}

type RPCRequest struct {
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
	ID      int    `json:"id"`
	JSONRPC string `json:"jsonrpc"`
}

func (client *CustomClient) SendRequest(ctx context.Context, method string, args []any) (string, error) {
	request := &RPCRequest{
		Method:  method,
		JSONRPC: "2.0",
	}
	if args != nil {
		request.Params = args
	}
	body, err := json.Marshal(request)
	if err != nil {
		return "", err
	}
	const maxRetries = 5
	waitTime := 500
	var res *http.Response
	for retries := 0; retries < maxRetries; retries++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, client.url, bytes.NewBuffer(body))
		if err != nil {
			return "", err
		}
		req.Header.Add("Content-Type", "application/json")
		for key, value := range CommonHTTPHeaders {
			req.Header.Add(key, value)
		}
		for key, value := range client.headers {
			req.Header.Add(key, value)
		}
		res, err = client.Do(req)
		if err != nil {
			return "", err
		}
		if res.StatusCode != http.StatusTooManyRequests {
			break
		}
		if client.disableRetryOnRateLimit == true {
			break
		}
		log.Printf("Server responded with %d %s. Retrying after %dms delay...\n", res.StatusCode, res.Status, waitTime)
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(time.Duration(waitTime) * time.Millisecond):
			waitTime *= 2
		}
	}
	defer func() {
		_ = res.Body.Close()
	}()
	text, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	if res.StatusCode == http.StatusOK {
		return string(text), nil
	} else {
		return "", fmt.Errorf("%d %s: %s", res.StatusCode, res.Status, string(text))
	}
}

// assertEndpointURL checks if the given URL starts with 'http:' or 'https:'
func assertEndpointURL(putativeURL string) (string, error) {
	match, err := regexp.MatchString(`^https?://`, putativeURL)
	if err != nil {
		return "", err
	}
	if !match {
		// URL does not start with 'http:' or 'https:'
		return "", errors.New("endpoint URL must start with `http:` or `https:`")
	}
	return putativeURL, nil
}

// ConnectionConfig represents configuration for instantiating a Connection
type ConnectionConfig struct {
	Commitment                       *Commitment       // Optional commitment level
	WsEndpoint                       *string           // Optional endpoint URL to the fullnode JSON RPC PubSub WebSocket Endpoint
	HttpHeaders                      map[string]string // Optional HTTP headers object
	DisableRetryOnRateLimit          *bool             // Optional Disable retrying calls when server responds with HTTP 429 (Too Many Requests)
	ConfirmTransactionInitialTimeout *int              // Time to allow for the server to initially process a transaction (in milliseconds)
}

type BigFloat big.Float

func (bf *BigFloat) Raw() *big.Float {
	if bf == nil {
		return nil
	}
	return (*big.Float)(bf)
}

func (bf *BigFloat) UnmarshalJSON(data []byte) error {
	z := new(big.Float)
	str := string(data)
	t, ok := z.SetString(str)
	if !ok {
		return fmt.Errorf("can not parse BigFloat with string `%s`", str)
	}
	*bf = BigFloat(*t)
	return nil
}

type ParsedAccountDataOrBytes struct {
	Data     ParsedAccountData
	Bytes    []byte
	Encoding string
}

func (t *ParsedAccountDataOrBytes) UnmarshalJSON(data []byte) (err error) {
	var array solana.Data
	if err = json.Unmarshal(data, &array); err == nil {
		t.Bytes = array.Content
		t.Encoding = string(array.Encoding)
		return nil
	} else {
		return json.Unmarshal(data, &t.Data)
	}
}
