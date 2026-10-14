import { useRef, useCallback } from 'react';
import { useConnection } from './useConnection';
import { useRooms } from './useRooms';
import { useMessages } from './useMessages';
import { DisplayMessage } from '../components/TerminalScreen';

export function useBacktrack() {
  const appendRef = useRef<(msg: string | DisplayMessage, room?: string) => void>(() => {});
  const resetRoomsRef = useRef<() => void>(() => {});

  const connection = useConnection({
    onLog: useCallback((msg: string) => appendRef.current(msg), []),
    onDisconnected: useCallback(() => resetRoomsRef.current(), []),
  });

  const rooms = useRooms({
    isOnline: connection.isOnline,
    onLog: useCallback((msg: string, room?: string) => appendRef.current(msg, room), []),
  });
  resetRoomsRef.current = rooms.resetRooms;

  const messages = useMessages({
    activeRoom: rooms.activeRoom,
    isOnline: connection.isOnline,
  });
  appendRef.current = messages.appendMessage;

  return {
    ...connection,
    ...rooms,
    ...messages,
  };
}
