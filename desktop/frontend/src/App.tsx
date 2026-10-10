import { useState, useEffect } from 'react';
import CrtOverlay from './components/CrtOverlay';
import TerminalHeader from './components/TerminalHeader';
import TerminalSidebar from './components/TerminalSidebar';
import TerminalScreen from './components/TerminalScreen';
import TerminalInput from './components/TerminalInput';
import SettingsModal from './components/SettingsModal';
import IdentifyModal from './components/IdentifyModal';
import { Connect, Disconnect, SendMessage, IsConnected, GetMemoryUsage } from '../wailsjs/go/main/App';
import { EventsOn } from '../wailsjs/runtime/runtime';

interface Peer {
  userId: number;
  username: string;
}

export default function App() {
  const [rooms, setRooms] = useState<string[]>(['GLOBAL', 'RETRO-LOUNGE', 'HARDWARE']);
  const [activeRoom, setActiveRoom] = useState<string>('GLOBAL');
  const [messages, setMessages] = useState<string[]>([
    "*** BACKTRACK TCP WORKSTATION v1.0 ***",
    "READY.",
  ]);
  const [username, setUsername] = useState<string>(() => {
    return localStorage.getItem('backtrack_username') || '';
  });
  const [isIdentifyOpen, setIsIdentifyOpen] = useState<boolean>(() => {
    return !localStorage.getItem('backtrack_username');
  });
  const [isOnline, setIsOnline] = useState<boolean>(false);
  const [peers, setPeers] = useState<Peer[]>([]);
  const [isSettingsOpen, setIsSettingsOpen] = useState<boolean>(false);
  const [serverAddr, setServerAddr] = useState<string>("localhost:9000");
  const [ramUsage, setRamUsage] = useState<string>("");

  const saveUsername = (newName: string) => {
    setUsername(newName);
    localStorage.setItem('backtrack_username', newName);
    setIsIdentifyOpen(false);
    setMessages((prev) => [
      ...prev,
      `*** OPERATOR CALLSIGN SET TO: ${newName} ***`,
      "READY TO CONNECT. CLICK [▶] CONNECT TCP TO LINK WITH SERVER."
    ]);
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
      setMessages((prev) => [...prev, logMsg]);
    });

    // 3. Incoming TCP chat message listener
    const unsubChat = EventsOn(
      'chat_message',
      (data: { userId: number; username: string; body: string; time: string }) => {
        setMessages((prev) => [...prev, `[${data.time}] ${data.username}: ${data.body}`]);
      }
    );

    // 4. Online peer tracking from protocol events
    const unsubUserInfo = EventsOn('user_info', (data: { userId: number; username: string }) => {
      setPeers((prev) => {
        if (prev.some((p) => p.userId === data.userId)) return prev;
        return [...prev, { userId: data.userId, username: data.username }];
      });
    });

    const unsubUserJoined = EventsOn('user_joined', (data: { userId: number; username: string }) => {
      setPeers((prev) => {
        if (prev.some((p) => p.userId === data.userId)) return prev;
        return [...prev, { userId: data.userId, username: data.username }];
      });
    });

    const unsubUserLeft = EventsOn('user_left', (data: { userId: number; username: string }) => {
      setPeers((prev) => prev.filter((p) => p.userId !== data.userId));
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
      unsubStatus?.();
      unsubLog?.();
      unsubChat?.();
      unsubUserInfo?.();
      unsubUserJoined?.();
      unsubUserLeft?.();
    };
  }, []);

  const handleConnectToggle = async () => {
    if (isOnline) {
      await Disconnect();
      setMessages((prev) => [...prev, '*** DISCONNECTED FROM SERVER ***']);
      return;
    }

    if (!username.trim()) {
      setIsIdentifyOpen(true);
      setMessages((prev) => [
        ...prev,
        '!!! IDENTIFICATION REQUIRED: SET USERNAME BEFORE CONNECTING !!!',
      ]);
      return;
    }

    setMessages((prev) => [...prev, `*** DIALING TCP ${serverAddr} AS ${username}... ***`]);
    try {
      await Connect(serverAddr, username);
    } catch (err: any) {
      setMessages((prev) => [...prev, `!!! CONNECTION FAILED: ${err} !!!`]);
    }
  };

  const handleSendMessage = async (text: string) => {
    if (!isOnline) {
      setMessages((prev) => [
        ...prev,
        '!!! CANNOT TRANSMIT: NOT CONNECTED TO TCP SERVER. CLICK [▶] CONNECT TCP FIRST !!!',
      ]);
      return;
    }

    try {
      await SendMessage(text);
    } catch (err: any) {
      setMessages((prev) => [...prev, `!!! TRANSMISSION ERROR: ${err} !!!`]);
    }
  };

  const handleCreateRoom = () => {
    const roomName = prompt("ENTER NEW CHANNEL NAME (E.G. 'TEST-ROOM'):");
    if (roomName && roomName.trim()) {
      const clean = roomName.trim().toUpperCase().replace(/[^A-Z0-9_-]/g, '');
      if (clean && !rooms.includes(clean)) {
        setRooms((prev) => [...prev, clean]);
        setActiveRoom(clean);
        setMessages((prev) => [
          ...prev,
          `*** CHANNEL CREATED: #${clean} ***`,
          `*** SWITCHED CONTEXT TO #${clean} ***`,
        ]);
      }
    }
  };

  const handleSelectRoom = (room: string) => {
    setActiveRoom(room);
    setMessages((prev) => [...prev, `*** SWITCHED TO CHANNEL: #${room} ***`]);
  };

  return (
    <div className="relative h-screen w-screen bg-[#020603] text-green-300 font-mono flex flex-col p-4 select-none overflow-hidden text-glow">
      {/* CRT Scanline & Screen Effects */}
      <CrtOverlay />

      {/* Top Header */}
      <TerminalHeader
        username={username}
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
          <TerminalInput username={username || 'GUEST'} onSendMessage={handleSendMessage} />
        </div>
      </div>

      {/* Identify / Username Selection Modal */}
      <IdentifyModal
        isOpen={isIdentifyOpen}
        initialUsername={username}
        onSave={saveUsername}
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
      />
    </div>
  );
}
