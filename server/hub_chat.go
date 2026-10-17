package server

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/nanatheblue/backtrack/protocol"
	"github.com/nanatheblue/backtrack/repository/message"
)

// messageMeta tracks the sender and channel of a message sent in the current session.
type messageMeta struct {
	userID uint32
	room   string
}

func (h *Hub) sendRoomHistory(u *User, room string) {
	if h.msgRepo == nil {
		return
	}

	// Snapshot active session message owners so we only attach live userIDs to messages
	// sent during the current session (preventing recycled IDs from claiming old messages).
	activeOwners := make(map[string]uint32, len(h.messageOwners))
	for msgID, meta := range h.messageOwners {
		if meta.room == room {
			activeOwners[msgID] = meta.userID
		}
	}

	h.dbWg.Add(1)
	go func() {
		defer h.dbWg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), dbOperationTimeout)
		defer cancel()

		recent, err := h.msgRepo.GetRecent(ctx, room, DefaultHistoryLimit)
		if err != nil {
			slog.Error("failed to fetch room history", "room", room, "username", u.Username, "error", err)
			return
		}
		for _, m := range recent {
			uid := uint32(0)
			if ownerID, ok := activeOwners[m.ID]; ok && ownerID == m.UserID {
				uid = m.UserID
			}
			r := m.Room
			if r == "" {
				r = room
			}
			u.send(protocol.MsgChatRecv, protocol.ChatRecvPayload(m.ID, uid, m.UserName, r, m.Body))
		}
	}()
}

func (h *Hub) handleChat(e chatEvent) {
	room := strings.ToUpper(strings.TrimSpace(e.room))
	if room == "" {
		room = protocol.DefaultRoom
	}
	if err := protocol.ValidateRoomName(room); err != nil {
		e.from.send(protocol.MsgServerError, protocol.ErrorPayload(err.Error()))
		return
	}

	members, exists := h.rooms[room]
	if !exists {
		e.from.send(protocol.MsgServerError, protocol.ErrorPayload(fmt.Sprintf("channel %q does not exist", room)))
		return
	}
	if _, inRoom := members[e.from.ID]; !inRoom {
		e.from.send(protocol.MsgServerError, protocol.ErrorPayload(fmt.Sprintf("cannot send message: you have not joined channel %q", room)))
		return
	}

	msgID := uuid.NewString()

	// Track in-memory message ownership for the active connection session
	h.messageOwners[msgID] = messageMeta{userID: e.from.ID, room: room}
	h.messageOrder = append(h.messageOrder, msgID)
	if len(h.messageOrder) > MaxTrackedMessages {
		oldest := h.messageOrder[0]
		h.messageOrder[0] = ""
		h.messageOrder = h.messageOrder[1:]
		delete(h.messageOwners, oldest)
	}

	// Persist message asynchronously
	if h.msgRepo != nil {
		h.dbWg.Add(1)
		go func(m *message.Message) {
			defer h.dbWg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), dbOperationTimeout)
			defer cancel()
			if err := h.msgRepo.Save(ctx, m); err != nil {
				slog.Error("failed to persist message", "room", m.Room, "username", m.UserName, "error", err)
			}
		}(&message.Message{
			ID:       msgID,
			Room:     room,
			UserID:   e.from.ID,
			UserName: e.from.Username,
			Body:     e.body,
		})
	}

	chatMSG := protocol.ChatRecvPayload(msgID, e.from.ID, e.from.Username, room, e.body)
	h.broadcastToRoom(room, protocol.MsgChatRecv, chatMSG, 0)
}

func (h *Hub) handleChatEdit(e chatEditEvent) {
	room := strings.ToUpper(strings.TrimSpace(e.room))
	if room == "" {
		room = protocol.DefaultRoom
	}
	if err := protocol.ValidateRoomName(room); err != nil {
		e.from.send(protocol.MsgServerError, protocol.ErrorPayload(err.Error()))
		return
	}

	members, exists := h.rooms[room]
	if !exists {
		e.from.send(protocol.MsgServerError, protocol.ErrorPayload(fmt.Sprintf("channel %q does not exist", room)))
		return
	}
	if _, inRoom := members[e.from.ID]; !inRoom {
		e.from.send(protocol.MsgServerError, protocol.ErrorPayload(fmt.Sprintf("cannot edit message: you have not joined channel %q", room)))
		return
	}

	meta, found := h.messageOwners[e.msgID]
	if !found {
		e.from.send(protocol.MsgServerError, protocol.ErrorPayload("message not found or not eligible for editing"))
		return
	}
	if meta.userID != e.from.ID {
		e.from.send(protocol.MsgServerError, protocol.ErrorPayload("permission denied: you can only edit messages sent during your current session"))
		return
	}
	if meta.room != room {
		e.from.send(protocol.MsgServerError, protocol.ErrorPayload("message does not belong to specified channel"))
		return
	}

	// Persist edit asynchronously
	if h.msgRepo != nil {
		h.dbWg.Add(1)
		go func(msgID, r, body string, uid uint32) {
			defer h.dbWg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), dbOperationTimeout)
			defer cancel()
			if err := h.msgRepo.Edit(ctx, &message.Message{ID: msgID, Room: r, Body: body}, uid); err != nil {
				slog.Error("failed to persist message edit", "msg_id", msgID, "error", err)
			}
		}(e.msgID, room, e.body, e.from.ID)
	}

	payload := protocol.ChatEditedPayload(e.msgID, e.from.ID, room, e.body)
	h.broadcastToRoom(room, protocol.MsgChatEdited, payload, 0)
}

func (h *Hub) handleChatDelete(e chatDeleteEvent) {
	room := strings.ToUpper(strings.TrimSpace(e.room))
	if room == "" {
		room = protocol.DefaultRoom
	}
	if err := protocol.ValidateRoomName(room); err != nil {
		e.from.send(protocol.MsgServerError, protocol.ErrorPayload(err.Error()))
		return
	}

	members, exists := h.rooms[room]
	if !exists {
		e.from.send(protocol.MsgServerError, protocol.ErrorPayload(fmt.Sprintf("channel %q does not exist", room)))
		return
	}
	if _, inRoom := members[e.from.ID]; !inRoom {
		e.from.send(protocol.MsgServerError, protocol.ErrorPayload(fmt.Sprintf("cannot delete message: you have not joined channel %q", room)))
		return
	}

	meta, found := h.messageOwners[e.msgID]
	if !found {
		e.from.send(protocol.MsgServerError, protocol.ErrorPayload("message not found or not eligible for deletion"))
		return
	}
	if meta.userID != e.from.ID {
		e.from.send(protocol.MsgServerError, protocol.ErrorPayload("permission denied: you can only delete messages sent during your current session"))
		return
	}
	if meta.room != room {
		e.from.send(protocol.MsgServerError, protocol.ErrorPayload("message does not belong to specified channel"))
		return
	}

	delete(h.messageOwners, e.msgID)
	for i, id := range h.messageOrder {
		if id == e.msgID {
			copy(h.messageOrder[i:], h.messageOrder[i+1:])
			h.messageOrder[len(h.messageOrder)-1] = ""
			h.messageOrder = h.messageOrder[:len(h.messageOrder)-1]
			break
		}
	}

	// Persist delete asynchronously
	if h.msgRepo != nil {
		h.dbWg.Add(1)
		go func(msgID string, uid uint32) {
			defer h.dbWg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), dbOperationTimeout)
			defer cancel()
			if err := h.msgRepo.Delete(ctx, msgID, uid); err != nil {
				slog.Error("failed to persist message delete", "msg_id", msgID, "error", err)
			}
		}(e.msgID, e.from.ID)
	}

	payload := protocol.ChatDeletedPayload(e.msgID, e.from.ID, room)
	h.broadcastToRoom(room, protocol.MsgChatDeleted, payload, 0)
}

