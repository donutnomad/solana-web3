package irys

import (
	"context"
	"github.com/donutnomad/solana-web3/web3"
	"strings"
)

func UploadLogoAndMetadata(
	ctx context.Context,
	connection *web3.Connection,
	node *IrysNode,
	signer web3.Signer,
	gateway string,
	name string,
	symbol string,
	description string,
	additional map[string]string,
	logo []byte,
) (string, error) {
	var image string
	if len(logo) > 0 {
		response, err := node.UploadImage(ctx, connection, signer, logo)
		if err != nil {
			return "", err
		}
		image = joinPath(gateway, response.Id)
	}
	var metadata = make(map[string]string)
	{
		for key, value := range additional {
			metadata[key] = value
		}
		metadata["name"] = name
		metadata["symbol"] = symbol
		metadata["description"] = description
		metadata["image"] = image
	}
	response, err := node.UploadJson(ctx, connection, signer, metadata)
	if err != nil {
		return "", err
	}
	return joinPath(gateway, response.Id), nil
}

func joinPath(host, path string) string {
	if strings.HasSuffix(host, "/") {
		return host + path
	}
	return host + "/" + path
}
