import { useState } from 'react';
import CrtOverlay from './components/CrtOverlay';
import TerminalHeader from './components/TerminalHeader';
import TerminalSidebar from './components/TerminalSidebar';
import TerminalScreen from './components/TerminalScreen';
import TerminalInput from './components/TerminalInput';
import SettingsModal from './components/SettingsModal';
import IdentifyModal from './components/IdentifyModal';
import { useBacktrack } from './hooks/useBacktrack';

export default function App() {
  const {
    rooms,
    activeRoom,
    visibleMessages,
    peers,
    isOnline,
    currentUserId,
    isRateLimited,
    ramUsage,
    serverAddr,
    setServerAddr,
    appendMessage,
    toggleConnect,
    sendMessage,
    editMessage,
    deleteMessage,
    createRoom,
    leaveRoom,
    selectRoom,
  } = useBacktrack();

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
  const [isSettingsOpen, setIsSettingsOpen] = useState<boolean>(false);

  const saveIdentity = (newName: string, newAvatarId: number) => {
    setUsername(newName);
    setAvatarId(newAvatarId);
    localStorage.setItem('backtrack_username', newName);
    localStorage.setItem('backtrack_avatar_id', newAvatarId.toString());
    setIsIdentifyOpen(false);
    appendMessage(`*** OPERATOR CALLSIGN SET: ${newName} (AVATAR #${newAvatarId}) ***`);
    appendMessage('READY TO CONNECT. CLICK [▶] CONNECT TCP TO LINK WITH SERVER.');
  };

  const handleConnectToggle = () => {
    if (!username.trim()) {
      setIsIdentifyOpen(true);
    }
    toggleConnect(username, avatarId);
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
          onSelectRoom={selectRoom}
          onCreateRoom={createRoom}
          onLeaveRoom={leaveRoom}
          onOpenSettings={() => setIsSettingsOpen(true)}
          peers={peers}
          isOnline={isOnline}
          onConnectToggle={handleConnectToggle}
        />

        <div className="flex-1 flex flex-col overflow-hidden min-h-0">
          <TerminalScreen
            messages={visibleMessages}
            activeRoom={activeRoom}
            currentUserId={currentUserId}
            onEditMessage={editMessage}
            onDeleteMessage={deleteMessage}
          />
          <TerminalInput
            username={username || 'GUEST'}
            onSendMessage={sendMessage}
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
