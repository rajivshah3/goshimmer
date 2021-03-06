package message

import (
	"net/http"
	"sync"

	"github.com/iotaledger/goshimmer/packages/binary/messagelayer/message"
	"github.com/iotaledger/goshimmer/plugins/messagelayer"
	"github.com/iotaledger/goshimmer/plugins/webapi"
	"github.com/iotaledger/hive.go/logger"
	"github.com/iotaledger/hive.go/node"
	"github.com/labstack/echo"
)

// PluginName is the name of the web API message endpoint plugin.
const PluginName = "WebAPI message Endpoint"

var (
	// plugin is the plugin instance of the web API message endpoint plugin.
	plugin *node.Plugin
	once   sync.Once
	log    *logger.Logger
)

// Plugin gets the plugin instance.
func Plugin() *node.Plugin {
	once.Do(func() {
		plugin = node.NewPlugin(PluginName, node.Enabled, configure)
	})
	return plugin
}

func configure(plugin *node.Plugin) {
	log = logger.NewLogger(PluginName)
	webapi.Server().POST("message/findById", findMessageByID)
	webapi.Server().POST("message/sendPayload", sendPayload)
}

// findMessageByID returns the array of messages for the
// given message ids (MUST be encoded in base58), in the same order as the parameters.
// If a node doesn't have the message for a given ID in its ledger,
// the value at the index of that message ID is empty.
// If an ID is not base58 encoded, an error is returned
func findMessageByID(c echo.Context) error {
	var request Request
	if err := c.Bind(&request); err != nil {
		log.Info(err.Error())
		return c.JSON(http.StatusBadRequest, Response{Error: err.Error()})
	}

	var result []Message
	for _, id := range request.IDs {
		log.Info("Received:", id)

		msgID, err := message.NewId(id)
		if err != nil {
			log.Info(err)
			return c.JSON(http.StatusBadRequest, Response{Error: err.Error()})
		}

		msgObject := messagelayer.Tangle().Message(msgID)
		msgMetadataObject := messagelayer.Tangle().MessageMetadata(msgID)

		if !msgObject.Exists() || !msgMetadataObject.Exists() {
			result = append(result, Message{})
			continue
		}

		msg := msgObject.Unwrap()
		msgMetadata := msgMetadataObject.Unwrap()

		msgResp := Message{
			Metadata: Metadata{
				Solid:              msgMetadata.IsSolid(),
				SolidificationTime: msgMetadata.SolidificationTime().Unix(),
			},
			ID:              msg.Id().String(),
			TrunkID:         msg.TrunkId().String(),
			BranchID:        msg.BranchId().String(),
			IssuerPublicKey: msg.IssuerPublicKey().String(),
			IssuingTime:     msg.IssuingTime().Unix(),
			SequenceNumber:  msg.SequenceNumber(),
			Payload:         msg.Payload().Bytes(),
			Signature:       msg.Signature().String(),
		}
		result = append(result, msgResp)

		msgMetadataObject.Release()
		msgObject.Release()
	}

	return c.JSON(http.StatusOK, Response{Messages: result})
}

// Response is the HTTP response containing the queried messages.
type Response struct {
	Messages []Message `json:"messages,omitempty"`
	Error    string    `json:"error,omitempty"`
}

// Request holds the message ids to query.
type Request struct {
	IDs []string `json:"ids"`
}

// Message contains information about a given message.
type Message struct {
	Metadata        `json:"metadata,omitempty"`
	ID              string `json:"ID,omitempty"`
	TrunkID         string `json:"trunkId,omitempty"`
	BranchID        string `json:"branchId,omitempty"`
	IssuerPublicKey string `json:"issuerPublicKey,omitempty"`
	IssuingTime     int64  `json:"issuingTime,omitempty"`
	SequenceNumber  uint64 `json:"sequenceNumber,omitempty"`
	Payload         []byte `json:"payload,omitempty"`
	Signature       string `json:"signature,omitempty"`
}

// Metadata contains metadata information of a message.
type Metadata struct {
	Solid              bool  `json:"solid,omitempty"`
	SolidificationTime int64 `json:"solidificationTime,omitempty"`
}
