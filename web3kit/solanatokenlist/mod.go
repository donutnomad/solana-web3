package solanatokenlist

import (
	"fmt"
	"strings"
)

var prefix3 = "https://raw.githubusercontent.com/"
var prefix2 = "https://raw.githubusercontent.com/solana-labs/token-list/main/assets/mainnet/"

func GetTokenInfo(address string) (name, symbol, logo string, ok bool) {
	var item Token
	item, ok = tokenList_101[address]
	if !ok {
		item, ok = tokenList_102[address]
		if !ok {
			item, ok = tokenList_103[address]
			if !ok {
				return "", "", "", false
			}
		}
	}
	var replacer = [8][2]string{
		{fmt.Sprintf("%s%s/logo.png", prefix2, address), "**"},
		{fmt.Sprintf("%s%s/", prefix2, address), ":B/"},
		{prefix2, ":C/"},
		{prefix3, ":D/"},
		{"https://arweave.net/", ":E/"},
		{"https://github.com/", ":F/"},
		{"https://cdn.jsdelivr.net/", ":G/"},
		{"https://imagedelivery.net/", ":H/"},
	}
	for _, rr := range replacer {
		if strings.HasPrefix(item.LogoURI, rr[1]) {
			logo = rr[0] + item.LogoURI[len(rr[1]):]
			break
		}
	}
	return item.Name, item.Symbol, logo, true
}
