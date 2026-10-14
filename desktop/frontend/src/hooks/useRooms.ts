import { useState, useEffect } from 'react';
import { JoinRoom } from '../../wailsjs/go/main/App';
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime';

interface UseRoomsOptions {
  isOnline: boolean;
  onLog?: (msg: string, room?: string) => void;
}

export function useRooms({ isOnline, onLog }: UseRoomsOptions) {
  const [rooms, setRooms] = useState<string[]>(['GLOBAL']);
  const [activeRoom, setActiveRoom] = useState<string>('GLOBAL');

  useEffect(() => {
    const unsubRoomList = EventsOn('room_list', (serverRooms: string[]) => {
      if (Array.isArray(serverRooms)) {
        setRooms((prev) => {
          const combined = ['GLOBAL', ...serverRooms.filter((r) => r !== 'GLOBAL')];
          return Array.from(new Set([...combined, ...prev]));
        });
      }
    });

    const unsubRoomJoined = EventsOn('room_joined', (data: { userId: number; username: string; room: string }) => {
      onLog?.(`*** ${data.username} JOINED CHANNEL #${data.room} ***`, data.room);
    });

    const unsubRoomLeft = EventsOn('room_left', (data: { userId: number; username: string; room: string }) => {
      onLog?.(`*** ${data.username} LEFT CHANNEL #${data.room} ***`, data.room);
    });

    return () => {
      unsubRoomList?.();
      unsubRoomJoined?.();
      unsubRoomLeft?.();
      EventsOff('room_list');
      EventsOff('room_joined');
      EventsOff('room_left');
    };
  }, [onLog]);

  const selectRoom = async (room: string) => {
    setActiveRoom(room);
    if (isOnline) {
      try {
        await JoinRoom(room);
      } catch (err) {
        console.error('JoinRoom failed:', err);
      }
    }
    onLog?.(`*** SWITCHED TO CHANNEL: #${room} ***`);
  };

  const createRoom = async () => {
    const roomName = prompt("ENTER NEW CHANNEL NAME (E.G. 'TEST-ROOM'):");
    if (roomName && roomName.trim()) {
      const clean = roomName.trim().toUpperCase().replace(/[^A-Z0-9_-]/g, '');
      if (clean) {
        if (!rooms.includes(clean)) {
          setRooms((prev) => [...prev, clean]);
        }
        setActiveRoom(clean);
        if (isOnline) {
          try {
            await JoinRoom(clean);
          } catch (err) {
            console.error('JoinRoom failed:', err);
          }
        }
        onLog?.(`*** CHANNEL CREATED: #${clean} ***`);
        onLog?.(`*** SWITCHED CONTEXT TO #${clean} ***`);
      }
    }
  };

  const resetRooms = () => {
    setRooms(['GLOBAL']);
    setActiveRoom('GLOBAL');
  };

  return {
    rooms,
    activeRoom,
    selectRoom,
    createRoom,
    resetRooms,
  };
}

