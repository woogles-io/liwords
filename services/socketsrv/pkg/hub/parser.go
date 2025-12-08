package sockets

import (
	"context"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

	"github.com/woogles-io/liwords/pkg/entity"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

const (
	ipcTimeout = 10 * time.Second
)

func extendTopic(c *Client, topic string) string {
	// The publish topic should encode the user ID and the login status.
	// This is so we don't have to wastefully unmarshal and remarshal here,
	// and also because NATS plays nicely with hierarchical subject names.
	first := ""
	second := ""
	if !c.authenticated {
		first = "anon"
	} else {
		first = "auth"
	}
	second = c.userID

	return topic + "." + first + "." + second + "." + c.connID
}

func (h *Hub) parseAndExecuteMessage(ctx context.Context, msg []byte, c *Client) error {
	// All socket messages are encoded entity.Events.
	// Determine header size based on protocol version
	headerSize := 2 // v1: 2-byte length prefix
	if c.protocolVersion == 2 {
		headerSize = 3 // v2: 3-byte length prefix
	}

	msgType := pb.MessageType(msg[headerSize])
	payload := msg[headerSize+1:]

	// Handle v2-specific messages locally (don't forward to NATS)
	switch msgType {
	case pb.MessageType_HANDSHAKE:
		return h.handleHandshake(ctx, payload, c)
	case pb.MessageType_SUBSCRIBE_REQUEST:
		return h.handleSubscribe(ctx, payload, c)
	case pb.MessageType_UNSUBSCRIBE_REQUEST:
		return h.handleUnsubscribe(ctx, payload, c)
	default:
		// Forward to NATS as before
		topicName := "ipc.pb." + strconv.Itoa(int(msgType))
		fullTopic := extendTopic(c, topicName)
		log.Debug().Str("fullTopic", fullTopic).Msg("nats-publish")
		return h.pubsub.natsconn.Publish(fullTopic, payload)
	}
}

// handleHandshake processes the v2 handshake message
func (h *Hub) handleHandshake(ctx context.Context, payload []byte, c *Client) error {
	handshake := &pb.Handshake{}
	if err := proto.Unmarshal(payload, handshake); err != nil {
		log.Err(err).Msg("unmarshal-handshake")
		return err
	}

	log.Debug().Str("connID", c.connID).Int32("version", int32(handshake.Version)).Msg("received-handshake")

	// Send acknowledgment
	ack := &pb.HandshakeAck{
		Version: pb.ProtocolVersion_PROTOCOL_V2,
		Success: true,
	}

	evt := entity.WrapEvent(ack, pb.MessageType_HANDSHAKE_ACK)
	bts, err := evt.SerializeV2()
	if err != nil {
		return err
	}
	c.send <- bts
	return nil
}

// handleSubscribe processes subscription requests (v2)
func (h *Hub) handleSubscribe(ctx context.Context, payload []byte, c *Client) error {
	req := &pb.SubscribeRequest{}
	if err := proto.Unmarshal(payload, req); err != nil {
		log.Err(err).Msg("unmarshal-subscribe-request")
		return err
	}

	log.Debug().Str("connID", c.connID).Str("path", req.Path).Msg("handling-subscribe")

	// Use existing registerRealm IPC - backend already handles path parsing
	rrr := &pb.RegisterRealmRequest{
		Path:   req.Path,
		UserId: c.userID,
	}
	data, err := proto.Marshal(rrr)
	if err != nil {
		return err
	}

	var realms []string
	if req.Path == "/" {
		// Lobby - no need to request a realm
		realms = []string{string(LobbyRealm), "chat-" + string(LobbyRealm)}
	} else {
		resp, err := h.pubsub.natsconn.Request("ipc.request.registerRealm", data, ipcTimeout)
		if err != nil {
			log.Err(err).Msg("subscribe-register-realm-error")
			// Send error response
			response := &pb.SubscribeResponse{
				Success:      false,
				ErrorMessage: "Failed to register realm: " + err.Error(),
			}
			evt := entity.WrapEvent(response, pb.MessageType_SUBSCRIBE_RESPONSE)
			bts, _ := evt.SerializeV2()
			c.send <- bts
			return err
		}

		rrResp := &pb.RegisterRealmResponse{}
		if err = proto.Unmarshal(resp.Data, rrResp); err != nil {
			return err
		}
		realms = rrResp.Realms
	}

	// Add client to the new realms
	if len(realms) > 0 {
		h.JoinRealms(c, realms)
		// Send initial realm info
		h.sendRealmInitInfo(c)
	}

	// Send success response to client
	response := &pb.SubscribeResponse{
		Success:  len(realms) > 0,
		Channels: realms,
	}
	if len(realms) == 0 {
		response.ErrorMessage = "No realms available for path"
	}
	evt := entity.WrapEvent(response, pb.MessageType_SUBSCRIBE_RESPONSE)
	bts, err := evt.SerializeV2()
	if err != nil {
		return err
	}
	c.send <- bts

	log.Debug().Str("connID", c.connID).Str("path", req.Path).Interface("realms", realms).Msg("subscribe-complete")
	return nil
}

// handleUnsubscribe processes unsubscription requests (v2)
func (h *Hub) handleUnsubscribe(ctx context.Context, payload []byte, c *Client) error {
	req := &pb.UnsubscribeRequest{}
	if err := proto.Unmarshal(payload, req); err != nil {
		log.Err(err).Msg("unmarshal-unsubscribe-request")
		return err
	}

	log.Debug().Str("connID", c.connID).Str("path", req.Path).Msg("handling-unsubscribe")

	// Notify backend of leaving (for presence updates, etc.)
	h.pubsub.natsconn.Publish(extendTopic(c, "ipc.pb.leaveTab"), []byte{})

	if req.Path == "" {
		// Unsubscribe from all realms
		h.LeaveAllRealms(c)
	} else {
		// Unsubscribe from all current realms (for now)
		// In the future, we could track which realms correspond to which paths
		h.LeaveAllRealms(c)
	}

	log.Debug().Str("connID", c.connID).Msg("unsubscribe-complete")
	return nil
}
