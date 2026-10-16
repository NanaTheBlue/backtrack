import { useState, useEffect, useRef, useCallback } from 'react';
import { DisplayMessage } from '../components/TerminalScreen';
import { SendMessage, EditMessage, DeleteMessage } from '../../wailsjs/go/main/App';
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime';

const MAX_MESSAGES_PER_ROOM = 250;
let msgSequence = 0;
const nextMsgId = () => `bt-${Date.now()}-${++msgSequence}`;

interface UseMessagesOptions {
  activeRoom: string;
  isOnline: boolean;
}

export function useMessages({ activeRoom, isOnline }: UseMessagesOptions) {
  const [roomMessages, setRoomMessages] = useState<Record<string, DisplayMessage[]>>({
    GLOBAL: [
      { id: nextMsgId(), text: '*** BACKTRACK TCP WORKSTATION v1.5 ***', isSystem: true, room: 'GLOBAL' },
      { id: nextMsgId(), text: 'READY.', isSystem: true, room: 'GLOBAL' },
    ],
  });
  const [isRateLimited, setIsRateLimited] = useState<boolean>(false);
  const rateLimitTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const activeRoomRef = useRef(activeRoom);
  activeRoomRef.current = activeRoom;

  const appendMessage = useCallback((msg: string | DisplayMessage, explicitRoom?: string) => {
    setRoomMessages((prev) => {
      const targetRoom = (
        explicitRoom ||
        (typeof msg !== 'string' && msg.room) ||
        activeRoomRef.current ||
        'GLOBAL'
      ).toUpperCase();

      const item: DisplayMessage =
        typeof msg === 'string'
          ? { id: nextMsgId(), text: msg, isSystem: true, room: targetRoom }
          : { ...msg, id: msg.id || nextMsgId(), room: targetRoom };

      const existing = prev[targetRoom] || [];
      if (!item.isSystem && item.id && existing.some((m) => m.id === item.id)) {
        return prev;
      }
      const updated =
        existing.length >= MAX_MESSAGES_PER_ROOM
          ? [...existing.slice(existing.length - MAX_MESSAGES_PER_ROOM + 1), item]
          : [...existing, item];

      return {
        ...prev,
        [targetRoom]: updated,
      };
    });
  }, []);

  useEffect(() => {
    const unsubLog = EventsOn('log_message', (logMsg: string) => {
      appendMessage(logMsg);
    });

    const unsubChat = EventsOn(
      'chat_message',
      (data: { id?: string; userId: number; username: string; avatarId: number; room: string; body: string; time: string }) => {
        appendMessage({
          text: data.body,
          username: data.username,
          avatarId: data.avatarId,
          time: data.time,
          id: data.id || nextMsgId(),
          userId: data.userId,
          room: data.room || 'GLOBAL',
        });
      }
    );

    const unsubChatEdited = EventsOn(
      'chat_edited',
      (data: { id: string; userId: number; room: string; body: string }) => {
        setRoomMessages((prev) => {
          const room = (data.room || 'GLOBAL').toUpperCase();
          const list = prev[room];
          if (!list) return prev;
          return {
            ...prev,
            [room]: list.map((m) =>
              m.id === data.id
                ? { ...m, text: data.body, isEdited: true }
                : m
            ),
          };
        });
      }
    );

    const unsubChatDeleted = EventsOn(
      'chat_deleted',
      (data: { id: string; userId: number; room: string }) => {
        setRoomMessages((prev) => {
          const room = (data.room || 'GLOBAL').toUpperCase();
          const list = prev[room];
          if (!list) return prev;
          return {
            ...prev,
            [room]: list.map((m) =>
              m.id === data.id
                ? { ...m, isDeleted: true }
                : m
            ),
          };
        });
      }
    );

    const unsubRateLimited = EventsOn('rate_limited', (data: { message: string }) => {
      setIsRateLimited(true);
      appendMessage(`*** [⚠ THROTTLED] ${data.message.toUpperCase()} — PAUSING TRANSMISSION ***`);
      if (rateLimitTimerRef.current) clearTimeout(rateLimitTimerRef.current);
      rateLimitTimerRef.current = setTimeout(() => {
        setIsRateLimited(false);
      }, 1500);
    });

    return () => {
      if (rateLimitTimerRef.current) clearTimeout(rateLimitTimerRef.current);
      unsubLog?.();
      unsubChat?.();
      unsubChatEdited?.();
      unsubChatDeleted?.();
      unsubRateLimited?.();
      EventsOff('log_message');
      EventsOff('chat_message');
      EventsOff('chat_edited');
      EventsOff('chat_deleted');
      EventsOff('rate_limited');
    };
  }, [appendMessage]);

  const sendMessage = async (text: string) => {
    if (!isOnline) {
      appendMessage('!!! CANNOT TRANSMIT: NOT CONNECTED TO TCP SERVER. CLICK [▶] CONNECT TCP FIRST !!!');
      return;
    }
    if (isRateLimited) {
      appendMessage('!!! CANNOT TRANSMIT: RATE LIMIT COOLDOWN ACTIVE. PLEASE WAIT A SECOND !!!');
      return;
    }
    const byteLen = new TextEncoder().encode(text).length;
    if (byteLen > 512) {
      appendMessage(`!!! TRANSMISSION BLOCKED: MESSAGE EXCEEDS 512-BYTE LIMIT (${byteLen} BYTES) — PLEASE TRIM TEXT !!!`);
      return;
    }
    try {
      await SendMessage(activeRoom, text);
    } catch (err: any) {
      appendMessage(`!!! TRANSMISSION ERROR: ${err} !!!`);
    }
  };

  const editMessage = async (msgId: string, room: string, newBody: string) => {
    if (!isOnline) {
      appendMessage('!!! CANNOT EDIT: NOT CONNECTED TO TCP SERVER !!!');
      return;
    }
    try {
      await EditMessage(msgId, room, newBody);
    } catch (err: any) {
      appendMessage(`!!! EDIT ERROR: ${err} !!!`);
    }
  };

  const deleteMessage = async (msgId: string, room: string) => {
    if (!isOnline) {
      appendMessage('!!! CANNOT DELETE: NOT CONNECTED TO TCP SERVER !!!');
      return;
    }
    try {
      await DeleteMessage(msgId, room);
    } catch (err: any) {
      appendMessage(`!!! DELETE ERROR: ${err} !!!`);
    }
  };

  const visibleMessages = roomMessages[activeRoom.toUpperCase()] || [];

  return {
    visibleMessages,
    isRateLimited,
    appendMessage,
    sendMessage,
    editMessage,
    deleteMessage,
  };
}

