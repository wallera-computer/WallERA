package types

type Mail struct {
	AppID   uint
	Payload []byte
}

func (m *Mail) CopyFrom(o Mail) {
	m.AppID = o.AppID
	m.Payload = o.Payload
}
