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
  onCreateRoom: () => void;
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
  onOpenSettings,
  peers,
  isOnline,
  onConnectToggle,
}: TerminalSidebarProps) {
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
              return (
                <button
                  key={room}
                  onClick={() => onSelectRoom(room)}
                  className={`w-full text-left px-2.5 py-1.5 text-xs font-mono tracking-wider uppercase transition-all flex items-center justify-between cursor-pointer ${
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
              );
            })}
          </div>

          {/* Create Room Button */}
          <button
            onClick={onCreateRoom}
            className="w-full mt-2.5 py-1.5 px-2 border-2 border-dashed border-green-400/80 hover:border-solid hover:bg-green-400 hover:text-black text-green-300 text-xs font-bold tracking-widest uppercase transition-all flex items-center justify-center gap-1.5 cursor-pointer"
          >
            <span>[+]</span> CREATE ROOM
          </button>
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
                    <span className="flex items-center gap-1.5 truncate">
                      <span className="text-sm leading-none" title={av.label}>
                        {av.glyph}
                      </span>
                      <span className="truncate">{peer.username}</span>
                    </span>
                    <span className="text-[9px] text-green-600 font-mono">#{peer.userId}</span>
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
