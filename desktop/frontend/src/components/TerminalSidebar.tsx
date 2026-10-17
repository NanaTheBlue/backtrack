import { useState } from 'react';
import { getAvatar } from '../utils/avatars';

interface Peer {
  userId: number;
  username: string;
  avatarId?: number;
}

interface TerminalSidebarProps {
  activeRoom: string;
  rooms: string[];
  onSelectRoom: (room: string) => void;
  onCreateRoom: (roomName: string) => void;
  onLeaveRoom?: (room: string) => void;
  onOpenSettings: () => void;
  peers: Peer[];
  isOnline: boolean;
  onConnectToggle: () => void;
}

export default function TerminalSidebar({
  activeRoom,
  rooms,
  onSelectRoom,
  onCreateRoom,
  onLeaveRoom,
  onOpenSettings,
  peers,
  isOnline,
  onConnectToggle,
}: TerminalSidebarProps) {
  const [isCreatingRoom, setIsCreatingRoom] = useState(false);
  const [newRoomInput, setNewRoomInput] = useState('');

  const handleCreateSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const clean = newRoomInput.trim().toUpperCase().replace(/[^A-Z0-9_-]/g, '').slice(0, 24);
    if (clean) {
      onCreateRoom(clean);
    }
    setNewRoomInput('');
    setIsCreatingRoom(false);
  };

  return (
    <aside className="w-64 border-2 border-green-400 p-3.5 box-glow bg-black/80 flex flex-col justify-between shrink-0 select-none">
      <div className="space-y-4">
        {/* Connection Control */}
        <div>
          <button
            onClick={onConnectToggle}
            className={`w-full py-1.5 px-2 border-2 text-xs font-bold tracking-widest uppercase transition-all cursor-pointer flex items-center justify-center gap-2 ${
              isOnline
                ? 'border-red-400 bg-red-950/40 text-red-300 hover:bg-red-400 hover:text-black'
                : 'border-green-400 bg-green-950/40 text-green-300 hover:bg-green-400 hover:text-black animate-pulse'
            }`}
          >
            <span>{isOnline ? '[✕] DISCONNECT' : '[▶] CONNECT TCP'}</span>
          </button>
        </div>

        {/* Section: Rooms Header */}
        <div>
          <div className="text-xs text-green-300 font-bold tracking-wider mb-2.5 pb-1 border-b border-green-500/40 flex items-center justify-between">
            <span>[#] ROOM DIRECTORY</span>
            <span className="text-[10px] text-green-400">{rooms.length} ACTIVE</span>
          </div>

          {/* Rooms List */}
          <div className="space-y-1.5">
            {rooms.map((room) => {
              const isActive = activeRoom === room;
              const isGlobal = room === 'GLOBAL';
              return (
                <div key={room} className="flex items-center gap-1">
                  <button
                    onClick={() => onSelectRoom(room)}
                    className={`flex-1 min-w-0 text-left px-2.5 py-1.5 text-xs font-mono tracking-wider uppercase transition-all flex items-center justify-between cursor-pointer ${
                      isActive
                        ? 'bg-green-400 text-black font-extrabold shadow-[0_0_12px_rgba(74,222,128,0.7)]'
                        : 'border border-green-500/30 text-green-300 hover:border-green-400 hover:bg-green-950/60'
                    }`}
                  >
                    <span className="truncate">
                      {isActive ? '► ' : '  '}#{room}
                    </span>
                    {isActive && <span className="text-[10px] animate-pulse">●</span>}
                  </button>
                  {!isGlobal && onLeaveRoom && (
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        onLeaveRoom(room);
                      }}
                      title={`Leave #${room}`}
                      className="px-1.5 py-1.5 text-[10px] font-mono border border-green-500/30 text-green-500 hover:border-red-400 hover:bg-red-950/50 hover:text-red-300 transition-colors cursor-pointer shrink-0"
                    >
                      ✕
                    </button>
                  )}
                </div>
              );
            })}
          </div>

          {/* Inline Create Room Input or Button */}
          {isCreatingRoom ? (
            <form onSubmit={handleCreateSubmit} className="mt-2.5 space-y-1">
              <div className="flex items-center gap-1 border-2 border-green-400 bg-black px-2 py-1">
                <span className="text-green-400 font-bold text-xs">#</span>
                <input
                  type="text"
                  value={newRoomInput}
                  onChange={(e) => setNewRoomInput(e.target.value.toUpperCase().replace(/[^A-Z0-9_-]/g, ''))}
                  onKeyDown={(e) => {
                    if (e.key === 'Escape') {
                      setNewRoomInput('');
                      setIsCreatingRoom(false);
                    }
                  }}
                  placeholder="CHANNEL-NAME"
                  maxLength={24}
                  autoFocus
                  className="flex-1 min-w-0 bg-transparent border-none outline-none text-green-200 font-mono text-xs tracking-wider placeholder-green-800"
                />
              </div>
              <div className="flex justify-between text-[9px] text-green-500 font-mono px-0.5">
                <span>[↵] JOIN/CREATE</span>
                <button
                  type="button"
                  onClick={() => {
                    setNewRoomInput('');
                    setIsCreatingRoom(false);
                  }}
                  className="hover:text-green-300 cursor-pointer"
                >
                  [ESC] CANCEL
                </button>
              </div>
            </form>
          ) : (
            <button
              onClick={() => setIsCreatingRoom(true)}
              className="w-full mt-2.5 py-1.5 px-2 border-2 border-dashed border-green-400/80 hover:border-solid hover:bg-green-400 hover:text-black text-green-300 text-xs font-bold tracking-widest uppercase transition-all flex items-center justify-center gap-1.5 cursor-pointer"
            >
              <span>[+]</span> CREATE ROOM
            </button>
          )}
        </div>

        {/* Section: Active Users from TCP Protocol */}
        <div>
          <div className="text-xs text-green-300 font-bold tracking-wider mb-2 pb-1 border-b border-green-500/40 flex items-center justify-between">
            <span>[?] PEERS ONLINE</span>
            <span className="text-[10px] text-green-400">{peers.length} NODES</span>
          </div>
          {peers.length === 0 ? (
            <div className="text-[11px] text-green-600 italic">No peers connected</div>
          ) : (
            <ul className="space-y-1 text-xs text-green-300/90 font-mono max-h-36 overflow-y-auto pr-1">
              {peers.map((peer) => {
                const av = getAvatar(peer.avatarId);
                return (
                  <li key={peer.userId} className="flex items-center justify-between truncate py-0.5">
                    <span className="flex items-center gap-2 truncate">
                      <span
                        className="inline-flex items-center justify-center w-4.5 h-4.5 text-xs leading-none shrink-0 select-none"
                        title={av.label}
                      >
                        {av.glyph}
                      </span>
                      <span className="truncate">{peer.username}</span>
                    </span>
                    <span className="text-[9px] text-green-600 font-mono shrink-0">#{peer.userId}</span>
                  </li>
                );
              })}
            </ul>
          )}
        </div>
      </div>

      {/* Footer: Settings Button */}
      <div className="pt-3 border-t border-green-500/40">
        <button
          onClick={onOpenSettings}
          className="w-full py-2 px-3 border-2 border-green-400 bg-green-950/40 hover:bg-green-400 hover:text-black text-green-200 font-bold text-xs tracking-widest uppercase transition-all flex items-center justify-center gap-2 cursor-pointer shadow-[0_0_8px_rgba(74,222,128,0.35)]"
        >
          <span>[⚙]</span> SYSTEM CONFIG
        </button>
      </div>
    </aside>
  );
}
