import { useEffect, useRef } from 'react';

interface TerminalScreenProps {
  messages: string[];
  activeRoom?: string;
}

export default function TerminalScreen({ messages, activeRoom = 'GLOBAL' }: TerminalScreenProps) {
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  return (
    <main className="flex-1 border-2 border-green-400 p-4 box-glow bg-black/75 overflow-y-auto space-y-1.5 text-sm tracking-wide flex flex-col justify-between">
      <div className="pb-2 mb-2 border-b border-green-500/40 text-xs text-green-300 font-bold tracking-wider flex justify-between select-none">
        <span>ACTIVE CHANNEL: #{activeRoom}</span>
        <span className="text-green-400 font-normal">[STREAM: TCP-9000]</span>
      </div>

      <div className="overflow-y-auto space-y-2 pr-2 flex-1 flex flex-col justify-end">
        {messages.map((msg, i) => (
          <div key={i} className="leading-relaxed break-all">
            <span className="text-green-400 font-bold mr-2">&gt;</span>
            <span className={msg.startsWith("***") ? "text-green-200 font-bold" : "text-green-300"}>
              {msg}
            </span>
          </div>
        ))}
        <div ref={bottomRef} />
      </div>
    </main>
  );
}
