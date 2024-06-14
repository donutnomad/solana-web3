package irys

import (
	"context"
	"github.com/davecgh/go-spew/spew"
	"github.com/donutnomad/solana-web3/test"
	"github.com/donutnomad/solana-web3/web3"
	"github.com/donutnomad/solana-web3/web3kit"
	"testing"
)

func TestGetPrice(t *testing.T) {
	var irysNode = NewIrys(DEV)

	var price = Must1(irysNode.GetPrice(1024))
	t.Logf("Price(SOL): %d", price)
}

func TestGetBalance(t *testing.T) {
	var irysNode = NewIrys(DEV)

	var balance = Must1(irysNode.GetBalance("G7gLJ333oxdVJWXHShSvaMsEkp3MxyzCx2nDxq55h663"))
	t.Logf("Balance(SOL): %d", balance)
}

func TestFund(t *testing.T) {
	var irysNode = NewIrys(DEV)
	var connection = Must1(web3.NewConnection(web3.Devnet.Url(), nil))
	var signer = test.GetYourPrivateKey()
	var ctx = context.Background()
	var amount uint64 = 100000000 // 0.1SOL

	var beforeBalance = Must1(irysNode.GetBalance(signer.PublicKey().Base58()))
	err := irysNode.Fund(ctx, func(to web3.PublicKey) (web3.TransactionSignature, error) {
		return web3kit.Token.Transfer(ctx, connection, signer, signer, to, web3.PublicKey{}, amount, web3.SystemProgramID, true, web3.ConfirmOptions{
			SkipPreflight: web3.Ref(true),
		})
	})
	if err != nil {
		panic(err)
	}
	var balance = Must1(irysNode.GetBalance(signer.PublicKey().Base58()))
	t.Logf("beforeBalance: %d, afterBalance: %d", beforeBalance, balance)
}

func TestUpload(t *testing.T) {
	var irysNode = NewIrys(DEV)
	var connection = Must1(web3.NewConnection(web3.Devnet.Url(), nil))
	var ctx = context.Background()
	var signer = test.GetYourPrivateKey()

	metadata := Must1(irysNode.UploadJson(ctx, connection, signer, map[string]string{
		"name":   "CCCC",
		"symbol": "ZZZZZZ",
	}))
	t.Logf("Result: %s", spew.Sdump(metadata))
	t.Logf("Url: https://arweave.net/%s", metadata.Id)
}

func TestImage(t *testing.T) {
	var irysNode = NewIrys(DEV)
	var connection = Must1(web3.NewConnection(web3.Devnet.Url(), nil))
	var ctx = context.Background()
	var signer = test.GetYourPrivateKey()
	var image = test.TestingLogo()

	metadata := Must1(irysNode.UploadImage(ctx, connection, signer, image))
	t.Logf("Result: %s", spew.Sdump(metadata))
	t.Logf("Url: https://arweave.net/%s", metadata.Id)
}
