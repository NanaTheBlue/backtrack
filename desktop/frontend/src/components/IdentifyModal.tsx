import { useState } from 'react';
import { RETRO_AVATARS } from '../utils/avatars';

interface IdentifyModalProps {
  isOpen: boolean;
  initialUsername: string;
  initialAvatarId?: number;
  onSave: (username: string, avatarId: number) => void;
  onCancel?: () => void;
  canCancel?: boolean;
}

export default function IdentifyModal({
  isOpen,
  initialUsername,
  initialAvatarId = 1,
  onSave,
  onCancel,
  canCancel = false,
}: IdentifyModalProps) {
  const [val, setVal] = useState(initialUsername || '');
  const [selectedAvatarId, setSelectedAvatarId] = useState<number>(initialAvatarId || 1);
  const [error, setError] = useState('');

  if (!isOpen) return null;

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const clean = val.trim();
    if (!clean) {
      setError('ERROR: USERNAME CANNOT BE BLANK');
      return;
    }
    if (clean.length > 24) {
      setError('ERROR: MAXIMUM 24 CHARACTERS');
      return;
    }
    setError('');
    onSave(clean, selectedAvatarId);
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/85 p-4 select-none">
      <div className="w-full max-w-md border-2 border-green-400 bg-black p-6 box-glow text-green-300 font-mono space-y-4">
        {/* Header */}
        <div className="border-b-2 border-green-400 pb-2 flex justify-between items-center">
          <span className="font-bold text-sm tracking-wider text-green-200">
            === OPERATOR IDENTIFICATION ===
          </span>
          {canCancel && onCancel && (
            <button
              onClick={onCancel}
              className="text-xs border border-green-400 px-1.5 py-0.5 hover:bg-green-400 hover:text-black cursor-pointer font-bold"
            >
              [X]
            </button>
          )}
        </div>

        <p className="text-xs text-green-400/90 leading-relaxed">
          THE BACKTRACK PROTOCOL (0x01 IDENTIFY) REQUIRES AN OPERATOR CALLSIGN AND AVATAR ID (UINT16).
        </p>

        {/* Input Form */}
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-xs font-bold text-green-300 mb-1.5">
              1. CALLSIGN / USERNAME:
            </label>
            <div className="flex items-center gap-2 border-2 border-green-400 bg-black/80 px-3 py-2">
              <span className="text-green-400 font-bold">&gt;</span>
              <input
                type="text"
                value={val}
                onChange={(e) => {
                  setVal(e.target.value.toUpperCase());
                  setError('');
                }}
                placeholder="CALLSIGN (E.G. NANA-01)"
                autoFocus
                maxLength={24}
                className="flex-1 bg-transparent border-none outline-none text-green-200 font-mono text-sm tracking-wider placeholder-green-800"
              />
            </div>
            {error && (
              <p className="text-[11px] text-red-400 font-bold mt-1 tracking-wider animate-pulse">
                {error}
              </p>
            )}
          </div>

          {/* Avatar Selector Grid */}
          <div>
            <label className="block text-xs font-bold text-green-300 mb-1.5">
              2. SELECT RETRO AVATAR:
            </label>
            <div className="grid grid-cols-5 gap-2 bg-black/60 border-2 border-green-500/50 p-2">
              {RETRO_AVATARS.map((av) => {
                const isSelected = selectedAvatarId === av.id;
                return (
                  <button
                    key={av.id}
                    type="button"
                    onClick={() => setSelectedAvatarId(av.id)}
                    title={`${av.label} (ID: ${av.id})`}
                    className={`flex flex-col items-center justify-center p-1.5 border transition-all cursor-pointer ${
                      isSelected
                        ? 'border-green-300 bg-green-400 text-black shadow-[0_0_8px_rgba(74,222,128,0.8)] scale-105'
                        : 'border-green-800/80 bg-black/60 hover:border-green-400 hover:bg-green-950/40 text-green-300'
                    }`}
                  >
                    <span className="w-7 h-7 flex items-center justify-center text-xl leading-none select-none">
                      {av.glyph}
                    </span>
                    <span className="text-[9px] mt-1 tracking-tighter truncate w-full text-center">
                      #{av.id}
                    </span>
                  </button>
                );
              })}
            </div>
          </div>

          <div className="pt-2 flex justify-end gap-2 border-t border-green-500/40">
            {canCancel && onCancel && (
              <button
                type="button"
                onClick={onCancel}
                className="border-2 border-green-500/50 px-3 py-1.5 text-xs text-green-400 hover:border-green-300 transition-colors cursor-pointer"
              >
                CANCEL
              </button>
            )}
            <button
              type="submit"
              className="border-2 border-green-400 px-5 py-1.5 text-xs font-bold tracking-widest bg-green-950 hover:bg-green-400 hover:text-black transition-colors cursor-pointer shadow-[0_0_10px_rgba(74,222,128,0.5)]"
            >
              INITIALIZE IDENTITY [↵]
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
