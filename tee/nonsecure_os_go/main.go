// Copyright (c) F-Secure Corporation
// https://foundry.f-secure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

package main

import (
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

func mnemonic() {

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

	reqBytes, err := token.PackageRequest(req, token.RequestMnemonic)
	if err != nil {
		log.Fatal(err)
	}

	m := tztypes.Mail{
		AppID:   info.AppletID,
		Payload: reqBytes,
	}

	log.Println("mail", m)

	err = client.NonsecureRPC{}.SendMail(m)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("calling retrieveresult from nonsecure world")
	res, err := client.NonsecureRPC{}.RetrieveResult(m.AppID)
	if err != nil {
		log.Fatal(err)
	}

	data := res.PayloadBytes()

	if err := token.UnpackResponse(data, &resp); err != nil {
		log.Fatal(err)
	}

	log.Println("generated mnemonic")
	log.Println(strings.Join(resp.Words, ", "))
}

func main() {
	log.Println("normal world os!")
	defer exit()

	for {
		mnemonic()
		time.Sleep(1 * time.Second)
	}
}
