import { useEffect, useRef } from 'react';
import { getAvatar } from '../utils/avatars';

export interface DisplayMessage {
  id?: string | number;
  isSystem?: boolean;
  text: string;
  username?: string;
  avatarId?: number;
  time?: string;
}

interface TerminalScreenProps {
  messages: (string | DisplayMessage)[];
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
        {messages.map((item, i) => {
          if (typeof item === 'string') {
            return (
              <div key={i} className="leading-relaxed break-all">
                <span className="text-green-400 font-bold mr-2">&gt;</span>
                <span
                  className={item.startsWith('***') ? 'text-green-200 font-bold' : 'text-green-300'}
                >
                  {item}
                </span>
              </div>
            );
          }

          if (item.isSystem) {
            return (
              <div key={i} className="leading-relaxed break-all text-green-200 font-bold">
                <span className="text-green-400 mr-2">&gt;</span>
                <span>{item.text}</span>
              </div>
            );
          }

          const avatar = getAvatar(item.avatarId);
          return (
            <div key={i} className="leading-relaxed break-all flex items-start gap-1.5">
              {item.time && (
                <span className="text-green-600 font-mono text-xs select-none">
                  [{item.time}]
                </span>
              )}
              <span className="text-base leading-none select-none" title={avatar.label}>
                {avatar.glyph}
              </span>
              <span className="text-green-200 font-bold select-none">
                {item.username}:
              </span>
              <span className="text-green-300">{item.text}</span>
            </div>
          );
        })}
        <div ref={bottomRef} />
      </div>
    </main>
  );
}
