package irys

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/donutnomad/solana-web3/web3"
	"github.com/donutnomad/solana-web3/web3kit"
	"io"
	"math/big"
	"net/http"
	"strconv"
	"time"
)

// IrysNode represents the Irys node.
type IrysNode struct {
	apiURL     string
	token      string
	headers    map[string]string
	HttpClient *http.Client
}

type Endpoint string

const (
	NODE1 Endpoint = "https://node1.irys.xyz"
	NODE2 Endpoint = "https://node2.irys.xyz"
	DEV   Endpoint = "https://devnet.irys.xyz"
)

func (e Endpoint) String() string {
	return string(e)
}

func NewIrys(node Endpoint) *IrysNode {
	return &IrysNode{
		apiURL:     node.String(),
		token:      "solana",
		HttpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

type UploadResponse struct {
	Id             string `json:"id"`
	Timestamp      int64  `json:"timestamp"`
	Version        string `json:"version"`
	Public         string `json:"public"`
	Signature      string `json:"signature"`
	DeadlineHeight int    `json:"deadlineHeight"`
	Block          int    `json:"block"`
}

var InsufficientBalance = errors.New("insufficient balance")

func (i *IrysNode) UploadImage(ctx context.Context, connection *web3.Connection, signer web3.Signer, image_ []byte) (*UploadResponse, error) {
	format, ok := sniffImageFormat(image_)
	if !ok {
		return nil, errors.New("not supported image format")
	}
	return i.UploadData(ctx, connection, signer, image_, "image/"+format)
}

func (i *IrysNode) UploadData(ctx context.Context, connection *web3.Connection, signer web3.Signer, data []byte, contentType string) (*UploadResponse, error) {
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	Must(i.FundByBytes(ctx, connection, signer, len(data)))
	return i.Upload(data, signer, map[string]string{
		"Content-Type": contentType,
	})
}

func (i *IrysNode) UploadJson(ctx context.Context, connection *web3.Connection, signer web3.Signer, data any) (_ *UploadResponse, err error) {
	defer Recover(&err)
	marshal := Must1(json.Marshal(data))
	Must(i.FundByBytes(ctx, connection, signer, len(marshal)))
	return i.Upload(marshal, signer, map[string]string{
		"Content-Type": "application/json",
	})
}

func (i *IrysNode) FundByBytes(ctx context.Context, connection *web3.Connection, signer web3.Signer, bs int) error {
	price, _, err := i.checkBalance(bs, signer.PublicKey())
	if err == nil {
		return nil
	}
	if !errors.Is(err, InsufficientBalance) {
		return err
	}
	return i.Fund(ctx, func(to web3.PublicKey) (web3.TransactionSignature, error) {
		return web3kit.Token.Transfer(ctx, connection, signer, signer, to, web3.PublicKey{}, price*5, web3.SystemProgramID, true, web3.ConfirmOptions{
			SkipPreflight: web3.Ref(true),
		})
	})
}

func (i *IrysNode) Upload(data []byte, signer web3.Signer, tags map[string]string) (_ *UploadResponse, err error) {
	defer Recover(&err)
	if len(data) > 50_000_000 {
		Must(errors.New("data too big"))
	}
	var tags_ []Tag
	for key, value := range tags {
		tags_ = append(tags_, Tag{
			Name:  key,
			Value: value,
		})
	}
	var tx = createTransaction(data, tags_)
	Must(tx.Sign(signer))
	var out UploadResponse
	Must(i.postTransaction(Must1(tx.asBytes()), &out))
	return &out, nil
}

// GetPrice calculates the price for [bytes] bytes paid for with [token] for the loaded Irys node.
func (i *IrysNode) GetPrice(bytes int) (uint64, error) {
	body, err := i.get(fmt.Sprintf("%s/price/%s/%d", i.apiURL, i.token, bytes), "Getting storage cost")
	if err != nil {
		return 0, err
	}
	return strconv.ParseUint(string(body), 10, 64)
}

func (i *IrysNode) GetBalance(address string) (uint64, error) {
	var data struct {
		Balance string `json:"balance"`
	}
	err := i.getObj(fmt.Sprintf("%s/account/balance/%s?address=%v", i.apiURL, i.token, address), "Getting balance", &data)
	if err != nil {
		return 0, err
	}
	return strconv.ParseUint(data.Balance, 10, 64)
}

func (i *IrysNode) Fund(ctx context.Context, transfer func(to web3.PublicKey) (web3.TransactionSignature, error)) (err error) {
	defer Recover(&err)
	to := Must1(i.getBundlerAddress(i.token))
	sig := Must1(transfer(*to))
	if !sig.IsZero() {
		if Must1(i.submitTransaction(ctx, sig.String())) == nil {
			err = fmt.Errorf("failed to post funding tx - %s - keep this id!\n", sig.String())
		}
	} else {
		err = errors.New("send transaction failed, signature: " + sig.String())
	}
	return
}

func (i *IrysNode) checkBalance(bs int, address web3.PublicKey) (price uint64, balance uint64, err error) {
	price, err = i.GetPrice(bs)
	if err != nil {
		return
	}
	balance, err = i.GetBalance(address.String())
	if err != nil {
		return
	}
	if balance >= price {
		return
	}
	err = InsufficientBalance
	return
}

// postTransaction uploads the Irys transaction.
func (i *IrysNode) postTransaction(rawTransaction []byte, ret any) error {
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/tx/%s", i.apiURL, i.token), bytes.NewBuffer(rawTransaction))
	if err != nil {
		return err
	}
	for key, value := range i.headers {
		req.Header.Set(key, value)
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	res, err := i.HttpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = res.Body.Close()
	}()

	// Check response status
	switch res.StatusCode {
	case 201:
		return fmt.Errorf(getErrorMessage(res))
	case 402:
		retryAfterHeader := res.Header.Get("Retry-After")
		errorMsg := fmt.Sprintf("%s%s", res.Status, getRetryAfterMessage(retryAfterHeader))
		return fmt.Errorf(errorMsg)
	default:
		if res.StatusCode >= 400 {
			return fmt.Errorf("whilst uploading Irys transaction: %s %s", res.Status, getErrorMessage(res))
		}
	}
	content, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(content, ret)
}

// submitTransaction submits a transaction.
func (i *IrysNode) submitTransaction(ctx context.Context, transactionID string) (*http.Response, error) {
	reqJSON, err := json.Marshal(map[string]string{"tx_id": transactionID})
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/account/balance/%s", i.apiURL, i.token)
	for idx := 0; idx < 5; idx++ {
		select {
		case <-time.After(200 * time.Millisecond):
			continue
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			var res *http.Response
			res, err = i.HttpClient.Post(url, "application/json", bytes.NewBuffer(reqJSON))
			if err != nil {
				continue
			}
			err = checkAndThrow(res, fmt.Sprintf("Posting transaction %s information to the bundler", transactionID), []int{202})
			if err != nil {
				continue
			}
			return res, nil
		}
	}
	return nil, err
}

func (i *IrysNode) getBundlerAddress(token string) (*web3.PublicKey, error) {
	var data struct {
		Version   string            `json:"version"`
		Addresses map[string]string `json:"addresses"`
		Gateway   string            `json:"gateway"`
	}
	err := i.getObj(fmt.Sprintf("%s/info", i.apiURL), "Getting Bundler address", &data)
	if err != nil {
		return nil, err
	}
	if v, ok := data.Addresses[token]; ok {
		out, err := web3.NewPublicKey(v)
		return &out, err
	}
	return nil, fmt.Errorf("specified bundler does not support token %s", token)
}

func (i *IrysNode) get(url string, context string) (_ []byte, err error) {
	defer Recover(&err)
	res := Must1(i.HttpClient.Get(url))
	defer res.Body.Close()
	Must(checkAndThrow(res, context, nil))
	return io.ReadAll(res.Body)
}

func (i *IrysNode) getObj(url string, context string, obj any) (err error) {
	defer Recover(&err)
	return json.Unmarshal(Must1(i.get(url, context)), obj)
}

func parseInt(s string) (*big.Int, error) {
	ret := new(big.Int)
	_, ok := ret.SetString(s, 10)
	if !ok {
		return nil, errors.New("parse int failed")
	}
	return ret, nil
}

// checkAndThrow throws an error if the provided http.Response has a status code != 200.
func checkAndThrow(res *http.Response, context string, exceptions []int) error {
	if res != nil && res.StatusCode != 200 && !contains(exceptions, res.StatusCode) {
		return fmt.Errorf("HTTP Error: %s: %d %s", context, res.StatusCode, getErrorMessage(res))
	}
	return nil
}

// contains checks if a given value exists in a slice.
func contains(slice []int, value int) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

// getErrorMessage retrieves the error message from the http.Response.
func getErrorMessage(res *http.Response) string {
	if res == nil || res.Body == nil {
		return ""
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return ""
	}
	return string(body)
}

// getRetryAfterMessage retrieves the retry-after message.
func getRetryAfterMessage(retryAfterHeader string) string {
	if retryAfterHeader != "" {
		return fmt.Sprintf(" - retry after %s", retryAfterHeader)
	}
	return ""
}

var supportsImageFormat = [][]string{{"jpeg", "\xff\xd8"}, {"png", "\x89PNG\r\n\x1a\n"}}

func sniffImageFormat(bs []byte) (string, bool) {
	var match = func(magic string, b []byte) bool {
		for i, c := range b {
			if magic[i] != c && magic[i] != '?' {
				return false
			}
		}
		return true
	}
	for _, format := range supportsImageFormat {
		magic := format[1]
		if len(bs) >= len(magic) && match(magic, bs[:len(magic)]) {
			return format[0], true
		}
	}
	return "", false
}

func Recover(err *error) {
	if r := recover(); r != nil {
		if e, ok := r.(error); ok {
			*err = e
		} else {
			panic(r)
		}
	}
}
func Must(err error) {
	if err != nil {
		panic(err)
	}
}
func Must1[T any](arg T, err error) T {
	if err != nil {
		panic(err)
	}
	return arg
}
func Must2[T any, T2 any](arg T, arg2 T2, err error) (T, T2) {
	if err != nil {
		panic(err)
	}
	return arg, arg2
}
