import { useState } from 'react';

interface TerminalInputProps {
  username: string;
  onSendMessage: (message: string) => void;
}

export default function TerminalInput({ username, onSendMessage }: TerminalInputProps) {
  const [inputVal, setInputVal] = useState('');

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!inputVal.trim()) return;
    onSendMessage(inputVal);
    setInputVal('');
  };

  return (
    <footer className="mt-3 border-2 border-green-400 p-3 box-glow bg-black/80">
      <form onSubmit={handleSubmit} className="flex items-center gap-2.5 text-sm">
        <span className="text-green-300 font-bold tracking-tight">
          {username}@BACKTRACK:&gt;
        </span>
        <input
          type="text"
          value={inputVal}
          onChange={(e) => setInputVal(e.target.value)}
          placeholder="Type command or message..."
          autoFocus
          className="flex-1 bg-transparent border-none outline-none text-green-200 placeholder-green-600/70 font-mono tracking-wider"
        />
        <button
          type="submit"
          className="px-4 py-1.5 bg-green-950 border-2 border-green-400 hover:bg-green-900 text-green-200 font-bold text-xs tracking-widest uppercase transition-colors cursor-pointer shadow-[0_0_8px_rgba(74,222,128,0.5)]"
        >
          SEND [↵]
        </button>
      </form>
    </footer>
  );
}
