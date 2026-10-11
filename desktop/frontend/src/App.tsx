import { useState, useEffect, useRef } from 'react';
import CrtOverlay from './components/CrtOverlay';
import TerminalHeader from './components/TerminalHeader';
import TerminalSidebar from './components/TerminalSidebar';
import TerminalScreen, { DisplayMessage } from './components/TerminalScreen';
import TerminalInput from './components/TerminalInput';
import SettingsModal from './components/SettingsModal';
import IdentifyModal from './components/IdentifyModal';
import { Connect, Disconnect, SendMessage, IsConnected, GetMemoryUsage } from '../wailsjs/go/main/App';
import { EventsOn } from '../wailsjs/runtime/runtime';

interface Peer {
  userId: number;
  username: string;
  avatarId?: number;
}

export default function App() {
  const [rooms, setRooms] = useState<string[]>(['GLOBAL']);
  const [activeRoom, setActiveRoom] = useState<string>('GLOBAL');
  const [messages, setMessages] = useState<(string | DisplayMessage)[]>([
    "*** BACKTRACK TCP WORKSTATION v1.2 ***",
    "READY.",
  ]);
  const [username, setUsername] = useState<string>(() => {
    return localStorage.getItem('backtrack_username') || '';
  });
  const [avatarId, setAvatarId] = useState<number>(() => {
    const saved = localStorage.getItem('backtrack_avatar_id');
    return saved ? parseInt(saved, 10) : 1;
  });
  const [isIdentifyOpen, setIsIdentifyOpen] = useState<boolean>(() => {
    return !localStorage.getItem('backtrack_username');
  });
  const [isOnline, setIsOnline] = useState<boolean>(false);
  const [isRateLimited, setIsRateLimited] = useState<boolean>(false);
  const rateLimitTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [peers, setPeers] = useState<Peer[]>([]);
  const [isSettingsOpen, setIsSettingsOpen] = useState<boolean>(false);
  const [serverAddr, setServerAddr] = useState<string>("localhost:9000");
  const [ramUsage, setRamUsage] = useState<string>("");

  const appendMessage = (msg: string | DisplayMessage) => {
    setMessages((prev) => {
      const maxCount = 500;
      if (prev.length >= maxCount) {
        return [...prev.slice(prev.length - maxCount + 1), msg];
      }
      return [...prev, msg];
    });
  };

  const saveIdentity = (newName: string, newAvatarId: number) => {
    setUsername(newName);
    setAvatarId(newAvatarId);
    localStorage.setItem('backtrack_username', newName);
    localStorage.setItem('backtrack_avatar_id', newAvatarId.toString());
    setIsIdentifyOpen(false);
    appendMessage(`*** OPERATOR CALLSIGN SET: ${newName} (AVATAR #${newAvatarId}) ***`);
    appendMessage("READY TO CONNECT. CLICK [▶] CONNECT TCP TO LINK WITH SERVER.");
  };

  useEffect(() => {
    // 1. Connection status change listener
    const unsubStatus = EventsOn('status_change', (data: { connected: boolean }) => {
      setIsOnline(data.connected);
      if (!data.connected) {
        setPeers([]);
      }
    });

    // 2. System and protocol logs
    const unsubLog = EventsOn('log_message', (logMsg: string) => {
      appendMessage(logMsg);
    });

    // 3. Incoming TCP chat message listener with avatar metadata
    const unsubChat = EventsOn(
      'chat_message',
      (data: { userId: number; username: string; avatarId: number; body: string; time: string }) => {
        appendMessage({
          text: data.body,
          username: data.username,
          avatarId: data.avatarId,
          time: data.time,
          id: data.userId,
        });
      }
    );

    // 4. Online peer tracking from protocol events
    const unsubUserInfo = EventsOn(
      'user_info',
      (data: { userId: number; username: string; avatarId: number }) => {
        setPeers((prev) => {
          if (prev.some((p) => p.userId === data.userId)) return prev;
          return [
            ...prev,
            { userId: data.userId, username: data.username, avatarId: data.avatarId },
          ];
        });
      }
    );

    const unsubUserJoined = EventsOn(
      'user_joined',
      (data: { userId: number; username: string; avatarId: number }) => {
        setPeers((prev) => {
          if (prev.some((p) => p.userId === data.userId)) return prev;
          return [
            ...prev,
            { userId: data.userId, username: data.username, avatarId: data.avatarId },
          ];
        });
      }
    );

    const unsubUserLeft = EventsOn('user_left', (data: { userId: number; username: string }) => {
      setPeers((prev) => prev.filter((p) => p.userId !== data.userId));
    });

    // 5. Rate limit warning listener
    const unsubRateLimited = EventsOn('rate_limited', (data: { message: string }) => {
      setIsRateLimited(true);
      appendMessage(`*** [⚠ THROTTLED] ${data.message.toUpperCase()} — PAUSING TRANSMISSION ***`);
      if (rateLimitTimerRef.current) clearTimeout(rateLimitTimerRef.current);
      rateLimitTimerRef.current = setTimeout(() => {
        setIsRateLimited(false);
      }, 1500);
    });

    // Initial connection check
    IsConnected()
      .then((connected) => {
        setIsOnline(connected);
      })
      .catch(() => {});

    // Live actual RAM monitoring
    const pollMemory = () => {
      GetMemoryUsage()
        .then(setRamUsage)
        .catch(() => {});
    };
    pollMemory();
    const memInterval = setInterval(pollMemory, 4000);

    return () => {
      clearInterval(memInterval);
      if (rateLimitTimerRef.current) clearTimeout(rateLimitTimerRef.current);
      unsubStatus?.();
      unsubLog?.();
      unsubChat?.();
      unsubUserInfo?.();
      unsubUserJoined?.();
      unsubUserLeft?.();
      unsubRateLimited?.();
    };
  }, []);

  const handleConnectToggle = async () => {
    if (isOnline) {
      await Disconnect();
      appendMessage('*** DISCONNECTED FROM SERVER ***');
      return;
    }

    if (!username.trim()) {
      setIsIdentifyOpen(true);
      appendMessage('!!! IDENTIFICATION REQUIRED: SET USERNAME BEFORE CONNECTING !!!');
      return;
    }

    appendMessage(`*** DIALING TCP ${serverAddr} AS ${username} (AVATAR #${avatarId})... ***`);
    try {
      await Connect(serverAddr, username, avatarId);
    } catch (err: any) {
      appendMessage(`!!! CONNECTION FAILED: ${err} !!!`);
    }
  };

  const handleSendMessage = async (text: string) => {
    if (!isOnline) {
      appendMessage('!!! CANNOT TRANSMIT: NOT CONNECTED TO TCP SERVER. CLICK [▶] CONNECT TCP FIRST !!!');
      return;
    }

    if (isRateLimited) {
      appendMessage('!!! CANNOT TRANSMIT: RATE LIMIT COOLDOWN ACTIVE. PLEASE WAIT A SECOND !!!');
      return;
    }

    try {
      await SendMessage(text);
    } catch (err: any) {
      appendMessage(`!!! TRANSMISSION ERROR: ${err} !!!`);
    }
  };

  const handleCreateRoom = () => {
    const roomName = prompt("ENTER NEW CHANNEL NAME (E.G. 'TEST-ROOM'):");
    if (roomName && roomName.trim()) {
      const clean = roomName.trim().toUpperCase().replace(/[^A-Z0-9_-]/g, '');
      if (clean && !rooms.includes(clean)) {
        setRooms((prev) => [...prev, clean]);
        setActiveRoom(clean);
        appendMessage(`*** CHANNEL CREATED: #${clean} ***`);
        appendMessage(`*** SWITCHED CONTEXT TO #${clean} ***`);
      }
    }
  };

  const handleSelectRoom = (room: string) => {
    setActiveRoom(room);
    appendMessage(`*** SWITCHED TO CHANNEL: #${room} ***`);
  };

  return (
    <div className="relative h-screen w-screen bg-[#020603] text-green-300 font-mono flex flex-col p-4 select-none overflow-hidden text-glow">
      {/* CRT Scanline & Screen Effects */}
      <CrtOverlay />

      {/* Top Header */}
      <TerminalHeader
        username={username}
        avatarId={avatarId}
        isOnline={isOnline}
        onEditUsername={() => setIsIdentifyOpen(true)}
        ramUsage={ramUsage}
        serverAddr={serverAddr}
      />

      {/* Main Layout: Sidebar (Left) + Screen & Input (Right) */}
      <div className="flex-1 flex gap-3.5 overflow-hidden min-h-0">
        <TerminalSidebar
          activeRoom={activeRoom}
          rooms={rooms}
          onSelectRoom={handleSelectRoom}
          onCreateRoom={handleCreateRoom}
          onOpenSettings={() => setIsSettingsOpen(true)}
          peers={peers}
          isOnline={isOnline}
          onConnectToggle={handleConnectToggle}
        />

        <div className="flex-1 flex flex-col overflow-hidden min-h-0">
          <TerminalScreen messages={messages} activeRoom={activeRoom} />
          <TerminalInput
            username={username || 'GUEST'}
            onSendMessage={handleSendMessage}
            isRateLimited={isRateLimited}
          />
        </div>
      </div>

      {/* Identify / Username Selection Modal */}
      <IdentifyModal
        isOpen={isIdentifyOpen}
        initialUsername={username}
        initialAvatarId={avatarId}
        onSave={saveIdentity}
        canCancel={!!username}
        onCancel={() => setIsIdentifyOpen(false)}
      />

      {/* Settings Dialog */}
      <SettingsModal
        isOpen={isSettingsOpen}
        onClose={() => setIsSettingsOpen(false)}
        serverAddr={serverAddr}
        setServerAddr={setServerAddr}
        username={username}
        setUsername={(name) => {
          setUsername(name);
          localStorage.setItem('backtrack_username', name);
        }}
        avatarId={avatarId}
        setAvatarId={(id) => {
          setAvatarId(id);
          localStorage.setItem('backtrack_avatar_id', id.toString());
        }}
      />
    </div>
  );
}
