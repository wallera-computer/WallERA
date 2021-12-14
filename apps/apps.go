package apps

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/hsanjuan/go-nfctype4/apdu"
)

// App represents an application, in charge of handling a given AppID and a set of commands.
// An App accepts a apdu.CAPDU packet in input, and returns an error (for log consumption),
// and a response for the USB host, as a byte slice.
// It may happen that a App returns both a non-nil error as well as a response byte slice.
// In that case, the error must be logged but the execution flow must always return the byte
// slice to the host, for protocol reasons.
type App interface {
	Name() string
	ID() byte
	Commands() (commandIDs []byte)
	Handle(command byte, data []byte) (response []byte, code APDUCode, err error)
}

type commandMapping struct {
	appID   byte
	command byte
}

// Handler keeps track of all the supported apps, and their commands.
type Handler struct {
	appMap        map[byte]App
	commandAppMap map[commandMapping]struct{}
}

func NewHandler() *Handler {
	return &Handler{
		appMap:        map[byte]App{},
		commandAppMap: map[commandMapping]struct{}{},
	}
}

func (h Handler) mappingExists(appID byte) bool {
	_, exists := h.appMap[appID]
	return exists
}

func (h Handler) commandAppMappingExists(appID, command byte) bool {
	_, exists := h.commandAppMap[commandMapping{
		appID:   appID,
		command: command,
	}]

	return exists
}

// Register registers apps into h.
// If an app was already registered, an error will be returned.
func (h *Handler) Register(apps ...App) error {
	for _, app := range apps {
		appID := app.ID()
		cmds := app.Commands()

		if h.mappingExists(appID) {
			return fmt.Errorf("mapping for %s already exists", app.Name())
		}

		h.appMap[appID] = app

		for _, cmd := range cmds {
			h.commandAppMap[commandMapping{
				appID:   appID,
				command: cmd,
			}] = struct{}{}
		}
	}

	return nil

}

// unmarshalCAPDU returns a command APDU packet from data.
func UnmarshalCAPDU(data []byte) (apdu.CAPDU, error) {
	packet := apdu.CAPDU{}
	_, err := packet.Unmarshal(data)
	if err != nil {
		return apdu.CAPDU{}, nil
	}

	return packet, nil
}

// Handle routes packet to the appropriate app handler.
// It returns a byte slice containing a response for the USB host, and an error
// which if present, should be logged.
func (h *Handler) Handle(data []byte) ([]byte, error) {
	capdu, err := UnmarshalCAPDU(data)
	if err != nil {
		return nil, err
	}

	appID := capdu.CLA
	command := capdu.INS

	if !h.mappingExists(appID) {
		return packageResponse(nil, APDUCLANotSupported),
			fmt.Errorf("appID %v not supported", appID)
	}

	if !h.commandAppMappingExists(appID, command) {
		return packageResponse(nil, APDUCLANotSupported),
			fmt.Errorf("command ID %v not supported in app %v", command, appID)
	}

	app := h.appMap[appID]

	respData, respCode, err := app.Handle(command, data)

	return packageResponse(respData, respCode), err
}

func packageResponse(data []byte, code APDUCode) []byte {
	buffer := &bytes.Buffer{}

	write := func(dest io.Writer, data interface{}) {
		if err := binary.Write(dest, binary.BigEndian, data); err != nil {
			panic(fmt.Sprintf("cannot package response, %s", err.Error()))
		}
	}

	write(buffer, data)
	write(buffer, code)

	return buffer.Bytes()
}
