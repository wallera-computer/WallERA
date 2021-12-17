package token

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/mitchellh/mapstructure"
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
	ID   uint
	Data interface{}
}

type RandomBytesRequest struct {
	Amount uint64
}

type RandomBytesResponse struct {
	Data []byte
}

type SignRequest struct {
	Data           []byte
	DerivationPath crypto.DerivationPath
	Algorithm      crypto.Algorithm
}

type SignResponse struct {
	Data []byte
}

type PublicKeyRequest struct {
	DerivationPath crypto.DerivationPath
}

type PublicKeyResponse struct {
	Data []byte
}

type MnemonicRequest struct {
	DerivationPath crypto.DerivationPath
}

type MnemonicResponse struct {
	Words []string
}

// SupportedSignAlgorithms doesn't have inputs
// just a response
type SupportedSignAlgorithmsResponse struct {
	Algorithms []crypto.Algorithm
}

type Response struct {
	ID   uint
	Data interface{}
}

func ReadRequest(data []byte) (Request, error) {
	var req Request
	return req, unmarshal(data, &req)
}

func MarshalResponse(r Response) ([]byte, error) {
	return marshal(r)
}

func Marshal(i interface{}) ([]byte, error) {
	return marshal(i)
}

func Unmarshal(b []byte, i interface{}) error {
	return unmarshal(b, &i)
}

func PackageRequest(req interface{}, reqCode uint) ([]byte, error) {
	rr := Request{
		ID:   reqCode,
		Data: req,
	}

	return Marshal(rr)
}

func UnpackResponse(resp []byte, dest interface{}) error {
	r := Response{}
	if err := unmarshal(resp, &r); err != nil {
		return err
	}

	b, err := base64.StdEncoding.DecodeString(r.Data.(string))
	if err != nil {
		return err
	}

	return json.Unmarshal(b, dest)
}

func Dispatch(req Request, t crypto.Token) (Response, error) {
	var resp Response
	var dispatchErr error

	switch req.ID {
	case RequestRandomBytes:
		r := RandomBytesRequest{}
		err := mapstructure.Decode(req.Data, &r)
		if err != nil {
			return Response{}, fmt.Errorf("cannot unmarshal into structure, %w", err)
		}

		rb, err := t.RandomBytes(r.Amount)
		if err != nil {
			return Response{}, err
		}

		resp = Response{
			ID: req.ID,
		}

		rbResp := RandomBytesResponse{
			Data: rb,
		}

		resp.Data, dispatchErr = marshal(rbResp)
	case RequestSign:
		r := SignRequest{}
		err := mapstructure.Decode(req.Data, &r)
		if err != nil {
			return Response{}, fmt.Errorf("cannot unmarshal into structure, %w", err)
		}

		tt := t.Clone()
		if err := tt.Initialize(r.DerivationPath); err != nil {
			return Response{}, err
		}

		data, err := tt.Sign(r.Data, r.Algorithm)
		if err != nil {
			return Response{}, err
		}

		resp = Response{
			ID: req.ID,
		}

		sResp := SignResponse{
			Data: data,
		}

		resp.Data, dispatchErr = marshal(sResp)
	case RequestPublicKey:
		r := PublicKeyRequest{}
		err := mapstructure.Decode(req.Data, &r)
		if err != nil {
			return Response{}, fmt.Errorf("cannot unmarshal into structure, %w", err)
		}

		tt := t.Clone()
		if err := tt.Initialize(r.DerivationPath); err != nil {
			return Response{}, err
		}

		data, err := tt.PublicKey()
		if err != nil {
			return Response{}, err
		}

		resp = Response{
			ID: req.ID,
		}

		pkResp := PublicKeyResponse{
			Data: data,
		}

		resp.Data, dispatchErr = marshal(pkResp)
	case RequestMnemonic:
		r := MnemonicRequest{}
		err := mapstructure.Decode(req.Data, &r)
		if err != nil {
			return Response{}, fmt.Errorf("cannot unmarshal into structure, %w", err)
		}

		tt := t.Clone()
		if err := tt.Initialize(r.DerivationPath); err != nil {
			return Response{}, err
		}

		data, err := tt.Mnemonic()
		if err != nil {
			return Response{}, err
		}

		resp = Response{
			ID: req.ID,
		}

		mnResp := MnemonicResponse{
			Words: data,
		}

		resp.Data, dispatchErr = marshal(mnResp)
	case RequestSupportedSignAlgorithms:
		data := t.SupportedSignAlgorithms()

		resp = Response{
			ID: req.ID,
		}

		mnResp := SupportedSignAlgorithmsResponse{
			Algorithms: data,
		}

		resp.Data, dispatchErr = marshal(mnResp)
	default:
		return Response{}, fmt.Errorf("cannot handle request")
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
