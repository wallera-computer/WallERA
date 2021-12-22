package token

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"

	"github.com/wallera-computer/wallera/crypto"
)

const (
	RequestRandomBytes uint = iota
	RequestSign
	RequestPublicKey
	RequestMnemonic
	RequestSupportedSignAlgorithms
)

type Request struct {
	ID uint
}

type RandomBytesRequest struct {
	Request
	Amount uint64
}

type RandomBytesResponse struct {
	Response
	Data []byte
}

type SignRequest struct {
	Request
	Data           []byte
	DerivationPath crypto.DerivationPath
	Algorithm      crypto.Algorithm
}
type signRequestInternal struct {
	Data           string
	DerivationPath crypto.DerivationPath
	Algorithm      crypto.Algorithm
}

func (sri signRequestInternal) Bytes() []byte {
	b, err := base64.StdEncoding.DecodeString(sri.Data)
	if err != nil {
		panic(err)
	}

	return b
}

type SignResponse struct {
	Response
	Data []byte
}

type PublicKeyRequest struct {
	Request
	DerivationPath crypto.DerivationPath
}

type PublicKeyResponse struct {
	Response
	Data []byte
}

type MnemonicRequest struct {
	Request
	DerivationPath crypto.DerivationPath
}

type MnemonicResponse struct {
	Response
	Words []string
}

// SupportedSignAlgorithms doesn't have inputs
// just a response
type SupportedSignAlgorithmsResponse struct {
	Response
	Algorithms []crypto.Algorithm
}

type SupportedSignAlgorithmsRequest struct {
	Request
}

type Response struct {
	ID uint
}

func ReadRequest(data []byte) (Request, error) {
	var req Request
	return req, unmarshal(data, &req)
}

func RequestedOp(data []byte) (uint, error) {
	var req Request
	if err := unmarshal(data, &req); err != nil {
		return 0, err
	}

	return req.ID, nil
}

func PackageRequest(req interface{}) ([]byte, error) {
	return marshal(req)
}

func UnpackResponse(resp []byte, dest interface{}) error {
	return unmarshal(resp, dest)
}

func Dispatch(data []byte, t crypto.Token) ([]byte, error) {
	var resp []byte
	var dispatchErr error

	log.Printf("dispatching %+v", string(data))

	reqID, err := RequestedOp(data)
	if err != nil {
		return nil, fmt.Errorf("cannot read op, %w", err)
	}

	switch reqID {
	case RequestRandomBytes:
		r := RandomBytesRequest{}
		if err := json.Unmarshal(data, &r); err != nil {
			return nil, err
		}

		rb, err := t.RandomBytes(r.Amount)
		if err != nil {
			return nil, err
		}

		rbResp := RandomBytesResponse{
			Response: Response{
				ID: reqID,
			},
			Data: rb,
		}

		resp, dispatchErr = marshal(rbResp)
	case RequestSign:
		r := signRequestInternal{}
		if err := json.Unmarshal(data, &r); err != nil {
			return nil, err
		}

		tt := t.Clone()
		if err := tt.Initialize(r.DerivationPath); err != nil {
			return nil, err
		}

		data, err := tt.Sign(r.Bytes(), r.Algorithm)
		if err != nil {
			return nil, err
		}

		sResp := SignResponse{
			Response: Response{
				ID: reqID,
			},
			Data: data,
		}

		resp, dispatchErr = marshal(sResp)
	case RequestPublicKey:
		r := PublicKeyRequest{}
		if err := json.Unmarshal(data, &r); err != nil {
			return nil, err
		}

		tt := t.Clone()
		if err := tt.Initialize(r.DerivationPath); err != nil {
			return nil, err
		}

		data, err := tt.PublicKey()
		if err != nil {
			return nil, err
		}

		pkResp := PublicKeyResponse{
			Response: Response{
				ID: reqID,
			},
			Data: data,
		}

		resp, dispatchErr = marshal(pkResp)
	case RequestMnemonic:
		r := MnemonicRequest{}
		if err := json.Unmarshal(data, &r); err != nil {
			return nil, err
		}

		tt := t.Clone()
		if err := tt.Initialize(r.DerivationPath); err != nil {
			return nil, err
		}

		data, err := tt.Mnemonic()
		if err != nil {
			return nil, err
		}

		mnResp := MnemonicResponse{
			Response: Response{
				ID: reqID,
			},
			Words: data,
		}

		resp, dispatchErr = marshal(mnResp)
	case RequestSupportedSignAlgorithms:
		data := t.SupportedSignAlgorithms()

		mnResp := SupportedSignAlgorithmsResponse{
			Response: Response{
				ID: reqID,
			},
			Algorithms: data,
		}

		resp, dispatchErr = marshal(mnResp)
	default:
		return nil, fmt.Errorf("cannot handle request")
	}

	return resp, dispatchErr
}

func unmarshal(data []byte, dest interface{}) error {
	return json.NewDecoder(bytes.NewReader(data)).Decode(&dest)
}

func marshal(src interface{}) ([]byte, error) {
	b := bytes.Buffer{}
	err := json.NewEncoder(&b).Encode(src)
	return b.Bytes(), err
}
