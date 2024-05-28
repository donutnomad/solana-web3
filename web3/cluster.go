package web3

import "fmt"

type Cluster string

const (
	Devnet      Cluster = "devnet"
	Testnet     Cluster = "testnet"
	MainnetBeta Cluster = "mainnet-beta"
)

func (c Cluster) Url() string {
	return apiUrl(c, true)
}

func (c Cluster) UrlWithoutTls() string {
	return apiUrl(c, false)
}

var endpoint = map[string]map[string]string{
	"http": {
		"devnet":       "http://api.devnet.solana.com",
		"testnet":      "http://api.testnet.solana.com",
		"mainnet-beta": "http://api.mainnet-beta.solana.com/",
	},
	"https": {
		"devnet":       "https://api.devnet.solana.com",
		"testnet":      "https://api.testnet.solana.com",
		"mainnet-beta": "https://api.mainnet-beta.solana.com/",
	},
}

func apiUrl(cluster Cluster, tls bool) string {
	key := "http"
	if tls {
		key = "https"
	}
	if cluster == "" {
		return endpoint[key]["devnet"]
	}
	url, exists := endpoint[key][string(cluster)]
	if !exists {
		panic(fmt.Sprintf("unknown %s cluster: %s", key, cluster))
	}
	return url
}
