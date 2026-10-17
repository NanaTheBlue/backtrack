import { useEffect, useRef, useState } from 'react';
import { getAvatar } from '../utils/avatars';

export interface DisplayMessage {
  id?: string | number;
  userId?: number;
  isSystem?: boolean;
  text: string;
  username?: string;
  avatarId?: number;
  time?: string;
  room?: string;
  isDeleted?: boolean;
  isEdited?: boolean;
}

interface ContextMenuState {
  x: number;
  y: number;
  message: DisplayMessage;
}

interface TerminalScreenProps {
  messages: (string | DisplayMessage)[];
  activeRoom?: string;
  currentUserId?: number | null;
  onEditMessage?: (msgId: string, room: string, newBody: string) => void;
  onDeleteMessage?: (msgId: string, room: string) => void;
}

export default function TerminalScreen({
  messages,
  activeRoom = 'GLOBAL',
  currentUserId = null,
  onEditMessage,
  onDeleteMessage,
}: TerminalScreenProps) {
  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const [contextMenu, setContextMenu] = useState<ContextMenuState | null>(null);
  const [editingId, setEditingId] = useState<string | number | null>(null);
  const [editText, setEditText] = useState('');

  useEffect(() => {
    // Instant scroll to bottom: eliminates WebKit smooth-scroll animation frames & timer leaks
    if (scrollContainerRef.current) {
      scrollContainerRef.current.scrollTop = scrollContainerRef.current.scrollHeight;
    }
  }, [messages]);

  // Dismiss context menu on click outside or Escape
  useEffect(() => {
    const handleGlobalClick = () => setContextMenu(null);
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        setContextMenu(null);
      }
    };
    window.addEventListener('click', handleGlobalClick);
    window.addEventListener('keydown', handleKeyDown);
    return () => {
      window.removeEventListener('click', handleGlobalClick);
      window.removeEventListener('keydown', handleKeyDown);
    };
  }, []);

  return (
    <main className="relative flex-1 border-2 border-green-400 p-4 box-glow bg-black/75 overflow-hidden text-sm tracking-wide flex flex-col justify-between">
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

          const isOwn =
            !item.isSystem &&
            currentUserId != null &&
            item.userId === currentUserId;

          const avatar = getAvatar(item.avatarId);
          const editByteLen = editingId === item.id ? new TextEncoder().encode(editText).length : 0;
          const isEditOverLimit = editByteLen > 512;

          return (
            <div
              key={key}
              onContextMenu={(e) => {
                if (isOwn && !item.isDeleted && item.id) {
                  e.preventDefault();
                  e.stopPropagation();
                  const clampedX = Math.min(e.clientX, window.innerWidth - 150);
                  const clampedY = Math.min(e.clientY, window.innerHeight - 90);
                  setContextMenu({
                    x: Math.max(8, clampedX),
                    y: Math.max(8, clampedY),
                    message: item,
                  });
                }
              }}
              className={`leading-relaxed break-words flex items-start gap-2 ${
                isOwn && !item.isDeleted ? 'hover:bg-green-500/5 px-1 -mx-1 rounded-sm cursor-context-menu' : ''
              }`}
            >
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

              {item.isDeleted ? (
                <span className="text-green-700 italic select-none">
                  [message deleted]
                </span>
              ) : editingId === item.id ? (
                <div className="flex-1 flex flex-col gap-1 min-w-0">
                  <input
                    type="text"
                    value={editText}
                    onChange={(e) => setEditText(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter') {
                        if (isEditOverLimit) return;
                        if (editText.trim() && editText !== item.text && onEditMessage && item.id) {
                          onEditMessage(String(item.id), item.room || activeRoom, editText);
                        }
                        setEditingId(null);
                      } else if (e.key === 'Escape') {
                        setEditingId(null);
                      }
                    }}
                    autoFocus
                    className={`bg-black/90 border px-2 py-0.5 text-sm font-mono outline-none w-full ${
                      isEditOverLimit
                        ? 'border-red-500 text-red-200'
                        : 'border-green-400 text-green-200 box-glow'
                    }`}
                  />
                  <div className="flex items-center gap-2 text-[10px] text-green-500 font-mono select-none">
                    <span>[ENTER] SAVE</span>
                    <span>•</span>
                    <span>[ESC] CANCEL</span>
                    <span>•</span>
                    <span className={isEditOverLimit ? 'text-red-400 font-bold' : ''}>
                      [{editByteLen}/512B]
                    </span>
                  </div>
                </div>
              ) : (
                <span className="text-green-300 break-words flex-1 min-w-0">
                  {item.text}
                  {item.isEdited && (
                    <span className="text-green-600 text-xs ml-1.5 select-none">(edited)</span>
                  )}
                </span>
              )}
            </div>
          );
        })}
      </div>

      {/* Retro Right-Click Context Menu */}
      {contextMenu && (
        <div
          style={{ top: `${contextMenu.y}px`, left: `${contextMenu.x}px` }}
          className="fixed z-50 bg-black/95 border border-green-400 box-glow text-xs text-green-300 p-1 flex flex-col shadow-2xl rounded-none select-none min-w-[130px] font-mono"
          onClick={(e) => e.stopPropagation()}
        >
          <button
            onClick={() => {
              setEditingId(contextMenu.message.id ?? null);
              setEditText(contextMenu.message.text);
              setContextMenu(null);
            }}
            className="px-3 py-1.5 text-left hover:bg-green-500/20 hover:text-green-100 flex items-center gap-2 cursor-pointer transition-colors"
          >
            <span className="text-green-400 font-bold">[✎]</span> EDIT
          </button>
          <div className="border-t border-green-500/30 my-0.5" />
          <button
            onClick={() => {
              if (onDeleteMessage && contextMenu.message.id) {
                onDeleteMessage(String(contextMenu.message.id), contextMenu.message.room || activeRoom);
              }
              setContextMenu(null);
            }}
            className="px-3 py-1.5 text-left hover:bg-red-500/20 hover:text-red-300 text-red-400 flex items-center gap-2 cursor-pointer transition-colors"
          >
            <span className="font-bold">[✖]</span> DELETE
          </button>
        </div>
      )}
    </main>
  );
}
