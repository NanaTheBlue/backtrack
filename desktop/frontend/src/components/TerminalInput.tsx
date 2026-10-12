import { useState } from 'react';

interface TerminalInputProps {
  username: string;
  onSendMessage: (message: string) => void;
  isRateLimited?: boolean;
}

const MAX_BYTES = 512;

export default function TerminalInput({
  username,
  onSendMessage,
  isRateLimited = false,
}: TerminalInputProps) {
  const [inputVal, setInputVal] = useState('');

  const byteLength = new TextEncoder().encode(inputVal).length;
  const isOverLimit = byteLength > MAX_BYTES;
  const overflowBytes = byteLength - MAX_BYTES;

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (isRateLimited || isOverLimit) return;
    if (!inputVal.trim()) return;
    onSendMessage(inputVal);
    setInputVal('');
  };

  const containerStyles = () => {
    if (isOverLimit) {
      return 'border-red-500 bg-red-950/50 shadow-[0_0_18px_rgba(239,68,68,0.7)] animate-pulse';
    }
    if (isRateLimited) {
      return 'border-amber-400 bg-amber-950/40 shadow-[0_0_16px_rgba(251,191,36,0.6)]';
    }
    return 'border-green-400 bg-black/80';
  };

  return (
    <footer className={`mt-3 border-2 p-3 box-glow transition-all duration-200 ${containerStyles()}`}>
      <form onSubmit={handleSubmit} className="flex items-center gap-2.5 text-sm">
        {isOverLimit ? (
          <span className="text-red-400 font-bold tracking-tight animate-pulse flex items-center gap-1.5 select-none shrink-0">
            <span>[⚠ OVERFLOW]</span>
            <span className="text-xs">-{overflowBytes}B OVER LIMIT</span>
          </span>
        ) : isRateLimited ? (
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
            isOverLimit
              ? `Message exceeds 512-byte limit by ${overflowBytes} bytes — trim text to send...`
              : isRateLimited
              ? 'Throttled by server — pausing transmission...'
              : 'Type command or message...'
          }
          autoFocus
          maxLength={2048}
          disabled={isRateLimited}
          className={`flex-1 bg-transparent border-none outline-none font-mono tracking-wider transition-colors ${
            isOverLimit
              ? 'text-red-200 placeholder-red-400/70'
              : isRateLimited
              ? 'text-amber-200 placeholder-amber-500/70 cursor-not-allowed'
              : 'text-green-200 placeholder-green-600/70'
          }`}
        />

        {/* Real-time Byte Counter */}
        <span
          className={`text-xs font-mono select-none shrink-0 px-1.5 py-0.5 rounded transition-colors ${
            isOverLimit
              ? 'text-red-300 bg-red-950 border border-red-500/80 font-bold animate-pulse'
              : byteLength > 400
              ? 'text-amber-400 bg-amber-950/40'
              : 'text-green-500/80'
          }`}
          title={`${byteLength} bytes of ${MAX_BYTES} byte limit`}
        >
          [{byteLength}/{MAX_BYTES}B]
        </span>

        <button
          type="submit"
          disabled={isRateLimited || isOverLimit || !inputVal.trim()}
          className={`px-4 py-1.5 border-2 font-bold text-xs tracking-widest uppercase transition-all select-none shrink-0 ${
            isOverLimit
              ? 'border-red-500 bg-red-950/80 text-red-300 cursor-not-allowed opacity-90 animate-pulse'
              : isRateLimited
              ? 'border-amber-500 bg-amber-950/80 text-amber-300 cursor-not-allowed opacity-90 animate-pulse'
              : !inputVal.trim()
              ? 'bg-green-950/40 border-green-700/50 text-green-700 cursor-not-allowed opacity-60'
              : 'bg-green-950 border-green-400 hover:bg-green-900 text-green-200 cursor-pointer shadow-[0_0_8px_rgba(74,222,128,0.5)]'
          }`}
        >
          {isOverLimit ? 'TRIM [⚠]' : isRateLimited ? 'WAIT [⌛]' : 'SEND [↵]'}
        </button>
      </form>
    </footer>
  );
}
