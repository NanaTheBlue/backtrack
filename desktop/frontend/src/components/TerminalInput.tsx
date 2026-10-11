import { useState } from 'react';

interface TerminalInputProps {
  username: string;
  onSendMessage: (message: string) => void;
  isRateLimited?: boolean;
}

export default function TerminalInput({
  username,
  onSendMessage,
  isRateLimited = false,
}: TerminalInputProps) {
  const [inputVal, setInputVal] = useState('');

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (isRateLimited) return;
    if (!inputVal.trim()) return;
    onSendMessage(inputVal);
    setInputVal('');
  };

  return (
    <footer
      className={`mt-3 border-2 p-3 box-glow transition-all duration-200 ${
        isRateLimited
          ? 'border-amber-400 bg-amber-950/40 shadow-[0_0_16px_rgba(251,191,36,0.6)]'
          : 'border-green-400 bg-black/80'
      }`}
    >
      <form onSubmit={handleSubmit} className="flex items-center gap-2.5 text-sm">
        {isRateLimited ? (
          <span className="text-amber-300 font-bold tracking-tight animate-pulse flex items-center gap-1.5 select-none shrink-0">
            <span>[⚠ THROTTLED]</span>
            <span className="text-xs">COOLING DOWN...</span>
          </span>
        ) : (
          <span className="text-green-300 font-bold tracking-tight select-none shrink-0">
            {username}@BACKTRACK:&gt;
          </span>
        )}
        <input
          type="text"
          value={inputVal}
          onChange={(e) => setInputVal(e.target.value)}
          placeholder={
            isRateLimited
              ? 'Throttled by server — pausing transmission...'
              : 'Type command or message...'
          }
          autoFocus
          maxLength={4000}
          disabled={isRateLimited}
          className={`flex-1 bg-transparent border-none outline-none font-mono tracking-wider transition-colors ${
            isRateLimited
              ? 'text-amber-200 placeholder-amber-500/70 cursor-not-allowed'
              : 'text-green-200 placeholder-green-600/70'
          }`}
        />
        <button
          type="submit"
          disabled={isRateLimited}
          className={`px-4 py-1.5 border-2 font-bold text-xs tracking-widest uppercase transition-all select-none shrink-0 ${
            isRateLimited
              ? 'border-amber-500 bg-amber-950/80 text-amber-300 cursor-not-allowed opacity-90 animate-pulse'
              : 'bg-green-950 border-green-400 hover:bg-green-900 text-green-200 cursor-pointer shadow-[0_0_8px_rgba(74,222,128,0.5)]'
          }`}
        >
          {isRateLimited ? 'WAIT [⌛]' : 'SEND [↵]'}
        </button>
      </form>
    </footer>
  );
}
