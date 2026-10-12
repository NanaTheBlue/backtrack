import { useEffect, useRef } from 'react';
import { getAvatar } from '../utils/avatars';

export interface DisplayMessage {
  id?: string | number;
  isSystem?: boolean;
  text: string;
  username?: string;
  avatarId?: number;
  time?: string;
  room?: string;
}

interface TerminalScreenProps {
  messages: (string | DisplayMessage)[];
  activeRoom?: string;
}

export default function TerminalScreen({ messages, activeRoom = 'GLOBAL' }: TerminalScreenProps) {
  const scrollContainerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    // Instant scroll to bottom: eliminates WebKit smooth-scroll animation frames & timer leaks
    if (scrollContainerRef.current) {
      scrollContainerRef.current.scrollTop = scrollContainerRef.current.scrollHeight;
    }
  }, [messages]);

  return (
    <main className="flex-1 border-2 border-green-400 p-4 box-glow bg-black/75 overflow-hidden text-sm tracking-wide flex flex-col justify-between">
      <div className="pb-2 mb-2 border-b border-green-500/40 text-xs text-green-300 font-bold tracking-wider flex justify-between select-none shrink-0">
        <span>ACTIVE CHANNEL: #{activeRoom}</span>
        <span className="text-green-400 font-normal">[STREAM: TCP-9000]</span>
      </div>

      <div ref={scrollContainerRef} className="overflow-y-auto space-y-2 pr-2 flex-1 flex flex-col justify-end">
        {messages.map((item, index) => {
          // Use stable unique ID instead of array index to prevent full DOM tree re-render on scrollback trims
          const key = typeof item === 'string' ? `sys-${index}-${item.slice(0, 16)}` : (item.id ?? `msg-${index}`);

          if (typeof item === 'string') {
            return (
              <div key={key} className="leading-relaxed break-all">
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
              <div key={key} className="leading-relaxed break-all text-green-200 font-bold">
                <span className="text-green-400 mr-2">&gt;</span>
                <span>{item.text}</span>
              </div>
            );
          }

          const avatar = getAvatar(item.avatarId);
          return (
            <div key={key} className="leading-relaxed break-words flex items-start gap-2">
              <div className="inline-flex items-center gap-1.5 shrink-0 select-none h-[1.625em]">
                {item.time && (
                  <span className="text-green-600 font-mono text-xs">
                    [{item.time}]
                  </span>
                )}
                <span
                  className="inline-flex items-center justify-center w-5 h-5 text-sm leading-none shrink-0"
                  title={avatar.label}
                >
                  {avatar.glyph}
                </span>
                <span className="text-green-200 font-bold">
                  {item.username}:
                </span>
              </div>
              <span className="text-green-300 break-words flex-1 min-w-0">
                {item.text}
              </span>
            </div>
          );
        })}
      </div>
    </main>
  );
}
