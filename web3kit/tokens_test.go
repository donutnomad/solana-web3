package web3kit

import (
	"context"
	"github.com/donutnomad/solana-web3/web3"
	"testing"
)

func TestGetTokenName(t *testing.T) {
	var ctx = context.Background()
	var connection = Must1(web3.NewConnection(web3.Devnet.Url(), nil))
	var tokens = [][]any{
		{"G6nZYEvhwFxxnp1KZr1v9igXtipuB5zL6oDGNMRZqF3q", true}, // static
		{"So11111111111111111111111111111111111111112", true},  // static
		{"DGGETjRbXeNyq2bpA7FLmWwqjFLtS8p5aYjzUwtAHtZd", true}, // static

		{"FH3i2zWEZRKQVkdqKknkfXzYgrGSTcc48VnwoJduf2o1", true}, // metaplex(devnet)
		{"Pg5zD1tVAEJQEwWSGRP5ajbCRuHvfGSC42niZHpYaJ7", true},  // metaplex(devnet)
		{"Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB", true}, // USDT static
		{web3.Keypair.Generate().PublicKey().Base58(), false},
	}
	for _, entry := range tokens {
		token := entry[0].(string)
		required := entry[1].(bool)
		var mint = web3.MustPublicKey(token)
		name, symbol, logo, err := Token.GetTokenName(ctx, connection, mint, nil)
		if err != nil {
			panic(err)
		}
		if required && (len(name) == 0 || len(symbol) == 0) {
			panic("failed:" + token)
		}
		t.Log(token, name, symbol, logo, err)
	}
}
