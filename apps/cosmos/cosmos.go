package cosmos

import (
	"fmt"
	"log"
)

type command byte

const (
	appName                     = "COSMOS"
	appID               byte    = 85
	claGetVersion       command = 0x00
	claSignSecp256K1    command = 0x02
	claGetAddrSecp256K1 command = 0x04
)

type Cosmos struct {
}

func (c *Cosmos) Name() string {
	return appName
}

func (c *Cosmos) ID() byte {
	return appID
}

func (c *Cosmos) Commands() (commandIDs []byte) {
	ret := []byte{
		byte(claGetVersion),
		byte(claSignSecp256K1),
		byte(claGetAddrSecp256K1),
	}

	return ret
}

func (c *Cosmos) Handle(command byte, data []byte) (response []byte, err error) {
	switch command {
	case byte(claGetVersion):
		return c.handleGetVersion(data)
	case byte(claSignSecp256K1):
		return c.handleSignSecp256K1(data)
	case byte(claGetAddrSecp256K1):
		return c.handleGetAddrSecp256K1(data)
	default:
		// TODO: handle this
		return nil, fmt.Errorf("command not found")
	}
}

func (c *Cosmos) handleGetVersion(data []byte) (response []byte, err error) {
	log.Println("called handleGetVersion")
	return nil, nil
}

func (c *Cosmos) handleSignSecp256K1(data []byte) (response []byte, err error) {
	return nil, nil
}

func (c *Cosmos) handleGetAddrSecp256K1(data []byte) (response []byte, err error) {
	return nil, nil
}
