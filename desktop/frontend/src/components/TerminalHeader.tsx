interface TerminalHeaderProps {
  username: string;
  isOnline?: boolean;
  onEditUsername?: () => void;
  ramUsage?: string;
  serverAddr?: string;
}

export default function TerminalHeader({
  username,
  isOnline = true,
  onEditUsername,
  ramUsage,
  serverAddr,
}: TerminalHeaderProps) {
  return (
    <header className="border-2 border-green-400 p-2.5 mb-3 box-glow bg-black/60 flex justify-between items-center text-xs tracking-wider select-none">
      <div className="flex items-center gap-3">
        <span className="font-bold text-green-300">=== BACKTRACK TCP TERMINAL ===</span>
        {serverAddr && (
          <span className="text-green-400 font-semibold">[TARGET: {serverAddr}]</span>
        )}
        {ramUsage && (
          <span className="text-green-400 font-semibold">[RAM: {ramUsage}]</span>
        )}
      </div>
      <div className="flex gap-4 text-xs font-semibold items-center">
        <button
          onClick={onEditUsername}
          title="Click to change operator callsign"
          className="hover:text-green-100 hover:border-b border-green-300 cursor-pointer transition-colors"
        >
          USER: <strong className="text-green-200">{username || 'UNIDENTIFIED'}</strong>
          <span className="text-[10px] text-green-400 ml-1">[EDIT]</span>
        </button>
        <span className={`flex items-center gap-1.5 ${isOnline ? 'text-green-300' : 'text-red-400'}`}>
          <span className="animate-pulse">●</span> {isOnline ? 'ONLINE' : 'OFFLINE'}
        </span>
      </div>
    </header>
  );
}
