// Copyright (c) F-Secure Corporation
// https://foundry.f-secure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
	_ "unsafe"

	"github.com/f-secure-foundry/tamago/soc/imx6"
	_ "github.com/f-secure-foundry/tamago/soc/imx6/imx6ul"

	"github.com/wallera-computer/wallera/crypto"
	"github.com/wallera-computer/wallera/tee/mem"
	"github.com/wallera-computer/wallera/tee/cryptography_applet/info"
	"github.com/wallera-computer/wallera/tee/cryptography_applet/token"
	"github.com/wallera-computer/wallera/tee/trusted_os/tz/client"
	tztypes "github.com/wallera-computer/wallera/tee/trusted_os/tz/types"
)

//go:linkname ramStart runtime.ramStart
var ramStart uint32 = mem.NonSecureStart

//go:linkname ramSize runtime.ramSize
var ramSize uint32 = mem.NonSecureSize

//go:linkname hwinit runtime.hwinit
func hwinit() {
	imx6.Init()
	imx6.UART2.Init()
}

//go:linkname printk runtime.printk
func printk(c byte) {
	if imx6.Native {
		// monitor call to request logs on Secure World SSH console
		printSecure(c)
	} else {
		imx6.UART2.Tx(c)
	}
}

func init() {
	log.SetFlags(log.Ltime)
	log.SetOutput(os.Stdout)

	if !imx6.Native {
		return
	}

	if err := imx6.SetARMFreq(900); err != nil {
		panic(fmt.Sprintf("WARNING: error setting ARM frequency: %v", err))
	}
}

func mnemonic() error {
	req := token.MnemonicRequest{
		Request: token.Request{
			ID: token.RequestMnemonic,
		},
		DerivationPath: crypto.DerivationPath{
			Purpose:      44,
			CoinType:     118,
			Account:      0,
			Change:       0,
			AddressIndex: 0,
		},
	}

	resp := token.MnemonicResponse{}

	if err := doRequest(req, &resp); err != nil {
		return err
	}

	log.Println("generated mnemonic")
	log.Println(strings.Join(resp.Words, ", "))

	return nil
}

func pubkey() error {
	req := token.PublicKeyRequest{
		Request: token.Request{
			ID: token.RequestPublicKey,
		},
		DerivationPath: crypto.DerivationPath{
			Purpose:      44,
			CoinType:     118,
			Account:      0,
			Change:       0,
			AddressIndex: 0,
		},
	}

	resp := token.PublicKeyResponse{}

	if err := doRequest(req, &resp); err != nil {
		return err
	}

	log.Println("generated pubkey")
	log.Println(resp.Data)

	return nil
}

func randomBytes() error {
	req := token.RandomBytesRequest{
		Request: token.Request{
			ID: token.RequestRandomBytes,
		},
		Amount: 42,
	}

	resp := token.RandomBytesResponse{}

	if err := doRequest(req, &resp); err != nil {
		return err
	}

	log.Println("generated random bytes")
	log.Println(len(resp.Data), resp.Data)

	return nil
}

func sign() error {
	signData := bytes.Repeat([]byte{42}, 42)
	hs := sha256.Sum256(signData)

	req := token.SignRequest{
		Request: token.Request{
			ID: token.RequestSign,
		},
		Data: hs[:],
		DerivationPath: crypto.DerivationPath{
			Purpose:      44,
			CoinType:     118,
			Account:      0,
			Change:       0,
			AddressIndex: 0,
		},
		Algorithm: crypto.AlgoSecp256K1,
	}

	resp := token.SignResponse{}

	if err := doRequest(req, &resp); err != nil {
		return err
	}

	log.Println("generated signature")
	log.Println(len(resp.Data), resp.Data)

	return nil
}

func supportedAlgorithms() error {
	req := token.SupportedSignAlgorithmsRequest{
		Request: token.Request{
			ID: token.RequestSupportedSignAlgorithms,
		},
	}

	resp := token.SupportedSignAlgorithmsResponse{}

	if err := doRequest(req, &resp); err != nil {
		return err
	}

	log.Println("supported algorithms")
	sa := []string{}
	for _, a := range resp.Algorithms {
		sa = append(sa, a.String())
	}

	log.Println(strings.Join(sa, ", "))

	return nil
}

func doRequest(input interface{}, output interface{}) error {
	reqBytes, err := token.PackageRequest(input)
	if err != nil {
		return err
	}

	m := tztypes.Mail{
		AppID:   info.AppletID,
		Payload: reqBytes,
	}

	err = client.NonsecureRPC{}.SendMail(m)
	if err != nil {
		return err
	}

	log.Println("mail sent")

	res, err := client.NonsecureRPC{}.RetrieveResult(m.AppID)
	if err != nil {
		return err
	}

	if err := token.UnpackResponse(res.Payload, output); err != nil {
		return err
	}

	return nil
}

func main() {
	log.Println("normal world os!")
	defer exit()

	funcList := []func() error{
		mnemonic,
		pubkey,
		sign,
		randomBytes,
		supportedAlgorithms,
	}

	var ferr error
	for i, f := range funcList {
		log.Println("calling function", i)
		if ferr = f(); ferr != nil {
			break
		}
	}

	if ferr != nil {
		log.Println(ferr)
		return
	}

	log.Println("finished running")

	time.Sleep(1 * time.Second)
}
