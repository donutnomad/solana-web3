package test

import (
	"encoding/base64"
	"encoding/json"
	"github.com/donutnomad/solana-web3/web3"
	"github.com/joho/godotenv"
	"github.com/mr-tron/base58"
	"os"
	"path"
	"runtime"
	"strings"
)

var _testLogo = "/9j/4AAQSkZJRgABAQEBLAEsAAD/4QmEaHR0cDovL25zLmFkb2JlLmNvbS94YXAvMS4wLwA8P3hwYWNrZXQgYmVnaW49Iu+7vyIgaWQ9Ilc1TTBNcENlaGlIenJlU3pOVGN6a2M5ZCI/PiA8eDp4bXBtZXRhIHhtbG5zOng9ImFkb2JlOm5zOm1ldGEvIiB4OnhtcHRrPSJYTVAgQ29yZSA2LjAuMCI+IDxyZGY6UkRGIHhtbG5zOnJkZj0iaHR0cDovL3d3dy53My5vcmcvMTk5OS8wMi8yMi1yZGYtc3ludGF4LW5zIyI+IDxyZGY6RGVzY3JpcHRpb24gcmRmOmFib3V0PSIiIHhtbG5zOmRjPSJodHRwOi8vcHVybC5vcmcvZGMvZWxlbWVudHMvMS4xLyI+IDxkYzpzdWJqZWN0PiA8cmRmOlNlcS8+IDwvZGM6c3ViamVjdD4gPC9yZGY6RGVzY3JpcHRpb24+IDwvcmRmOlJERj4gPC94OnhtcG1ldGE+ICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgPD94cGFja2V0IGVuZD0idyI/Pv/bAEMABAMDBAMDBAQDBAUEBAUGCgcGBgYGDQkKCAoPDRAQDw0PDhETGBQREhcSDg8VHBUXGRkbGxsQFB0fHRofGBobGv/bAEMBBAUFBgUGDAcHDBoRDxEaGhoaGhoaGhoaGhoaGhoaGhoaGhoaGhoaGhoaGhoaGhoaGhoaGhoaGhoaGhoaGhoaGv/AABEIAGQAZAMBEQACEQEDEQH/xAAcAAACAgMBAQAAAAAAAAAAAAABAgYIAAUHCQT/xAA/EAABAgQDBQQGCAQHAAAAAAABAgMABAURBiExBxJBYXEIEyJRIzZ1gbGzFBUyQlJicpEW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"

func init() {
	_, filename, _, _ := runtime.Caller(0)
	root := path.Dir(path.Dir(filename))
	_ = godotenv.Load(path.Join(root, ".env"))
}

func TestingLogo() []byte {
	return Must1(base64.StdEncoding.DecodeString(_testLogo))
}

func processPath(input string) string {
	if strings.HasPrefix(input, "~") {
		dir := Must1(os.UserHomeDir())
		return path.Join(dir, input[1:])
	} else {
		return input
	}
}

func GetYourPrivateKey() web3.Signer {
	privateKey := os.Getenv("TEST_SIGNER")
	if len(privateKey) == 0 {
		panic("Please specify the environment variable: TEST_SIGNER=~/.config/solana/id.json")
	}
	var bs []byte
	if strings.HasSuffix(privateKey, ".json") {
		file := Must1(os.ReadFile(processPath(privateKey)))
		Must(json.Unmarshal(file, &bs))
	} else {
		bs = Must1(base58.Decode(privateKey))
	}
	return web3.NewSigner(bs)
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
func Must3[T any, T2 any, T3 any](arg T, arg2 T2, arg3 T3, err error) (T, T2, T3) {
	if err != nil {
		panic(err)
	}
	return arg, arg2, arg3
}
