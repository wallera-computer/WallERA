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
	"github.com/wallera-computer/wallera/tee/trusted_applet_go/info"
	"github.com/wallera-computer/wallera/tee/trusted_applet_go/token"
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
		DerivationPath: crypto.DerivationPath{
			Purpose:      44,
			CoinType:     118,
			Account:      0,
			Change:       0,
			AddressIndex: 0,
		},
	}

	resp := token.MnemonicResponse{}

	if err := doRequest(token.RequestMnemonic, req, &resp); err != nil {
		return err
	}

	log.Println("generated mnemonic")
	log.Println(strings.Join(resp.Words, ", "))

	return nil
}

func pubkey() error {
	req := token.PublicKeyRequest{
		DerivationPath: crypto.DerivationPath{
			Purpose:      44,
			CoinType:     118,
			Account:      0,
			Change:       0,
			AddressIndex: 0,
		},
	}

	resp := token.PublicKeyResponse{}

	if err := doRequest(token.RequestPublicKey, req, &resp); err != nil {
		return err
	}

	log.Println("generated pubkey")
	log.Println(resp.Data)

	return nil
}

func randomBytes() error {
	req := token.RandomBytesRequest{
		Amount: 42,
	}

	resp := token.RandomBytesResponse{}

	if err := doRequest(token.RequestRandomBytes, req, &resp); err != nil {
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

	if err := doRequest(token.RequestSign, req, &resp); err != nil {
		return err
	}

	log.Println("generated signature")
	log.Println(len(resp.Data), resp.Data)

	return nil
}

func supportedAlgorithms() error {
	resp := token.SupportedSignAlgorithmsResponse{}

	if err := doRequest(token.RequestSupportedSignAlgorithms, nil, &resp); err != nil {
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

func doRequest(reqType uint, input interface{}, output interface{}) error {
	reqBytes, err := token.PackageRequest(input, reqType)
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

	data := res.PayloadBytes()

	if err := token.UnpackResponse(data, &output); err != nil {
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
			return
		}
	}

	if ferr != nil {
		log.Println(ferr)
		return
	}

	log.Println("finished running")

	time.Sleep(1 * time.Second)
}
