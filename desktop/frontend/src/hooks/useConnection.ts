import { useState, useEffect } from 'react';
import { Connect, Disconnect, IsConnected, GetMemoryUsage, GetUserID } from '../../wailsjs/go/main/App';
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime';

export interface Peer {
  userId: number;
  username: string;
  avatarId?: number;
}

interface UseConnectionOptions {
  onLog?: (msg: string) => void;
  onDisconnected?: () => void;
}

export function useConnection(options?: UseConnectionOptions) {
  const [isOnline, setIsOnline] = useState<boolean>(false);
  const [currentUserId, setCurrentUserId] = useState<number | null>(null);
  const [peers, setPeers] = useState<Peer[]>([]);
  const [serverAddr, setServerAddr] = useState<string>('localhost:9000');
  const [ramUsage, setRamUsage] = useState<string>('');

  useEffect(() => {
    const unsubStatus = EventsOn('status_change', (data: { connected: boolean }) => {
      setIsOnline(data.connected);
      if (!data.connected) {
        setPeers([]);
        setCurrentUserId(null);
        options?.onDisconnected?.();
      }
    });

    const unsubWelcome = EventsOn('welcome', (data: { userId: number }) => {
      setCurrentUserId(data.userId);
    });

    const unsubUserInfo = EventsOn('user_info', (data: { userId: number; username: string; avatarId: number }) => {
      setPeers((prev) => {
        if (prev.some((p) => p.userId === data.userId)) return prev;
        return [...prev, { userId: data.userId, username: data.username, avatarId: data.avatarId }];
      });
    });

    const unsubUserJoined = EventsOn('user_joined', (data: { userId: number; username: string; avatarId: number }) => {
      setPeers((prev) => {
        if (prev.some((p) => p.userId === data.userId)) return prev;
        return [...prev, { userId: data.userId, username: data.username, avatarId: data.avatarId }];
      });
    });

    const unsubUserLeft = EventsOn('user_left', (data: { userId: number; username: string }) => {
      setPeers((prev) => prev.filter((p) => p.userId !== data.userId));
    });

    // Check existing connection
    IsConnected()
      .then((connected) => {
        setIsOnline(connected);
        if (connected) {
          GetUserID().then((uid) => {
            if (uid) setCurrentUserId(uid);
          }).catch(() => {});
        }
      })
      .catch(() => {});

    // RAM monitoring
    const pollMemory = () => {
      GetMemoryUsage().then(setRamUsage).catch(() => {});
    };
    pollMemory();
    const memInterval = setInterval(pollMemory, 4000);

    return () => {
      clearInterval(memInterval);
      unsubStatus?.();
      unsubWelcome?.();
      unsubUserInfo?.();
      unsubUserJoined?.();
      unsubUserLeft?.();
      EventsOff('status_change');
      EventsOff('welcome');
      EventsOff('user_info');
      EventsOff('user_joined');
      EventsOff('user_left');
    };
  }, [options]);

  const toggleConnect = async (username: string, avatarId: number) => {
    if (isOnline) {
      await Disconnect();
      options?.onLog?.('*** DISCONNECTED FROM SERVER ***');
      return;
    }

    if (!username.trim()) {
      options?.onLog?.('!!! IDENTIFICATION REQUIRED: SET USERNAME BEFORE CONNECTING !!!');
      return;
    }

    options?.onLog?.(`*** DIALING TCP ${serverAddr} AS ${username} (AVATAR #${avatarId})... ***`);
    try {
      await Connect(serverAddr, username, avatarId);
    } catch (err: any) {
      options?.onLog?.(`!!! CONNECTION FAILED: ${err} !!!`);
    }
  };

  return {
    isOnline,
    currentUserId,
    peers,
    serverAddr,
    setServerAddr,
    ramUsage,
    toggleConnect,
  };
}

