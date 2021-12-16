package types

import (
	"encoding/base64"
	"fmt"
)

type Mail struct {
	AppID   uint
	Payload interface{}
}

func (m Mail) PayloadBytes() []byte {
	b, ok := m.Payload.(string)
	if !ok {
		panic(fmt.Errorf("payload is not bytes"))
	}

	data, err := base64.StdEncoding.DecodeString(b)
	if err != nil {
		panic(fmt.Errorf("cannot unmarshal base64, %w", err))
	}

	return data
}

func (m *Mail) CopyFrom(o Mail) {
	m.AppID = o.AppID
	m.Payload = o.Payload
}
