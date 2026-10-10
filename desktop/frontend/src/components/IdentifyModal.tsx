import { useState } from 'react';

interface IdentifyModalProps {
  isOpen: boolean;
  initialUsername: string;
  onSave: (username: string) => void;
  onCancel?: () => void;
  canCancel?: boolean;
}

export default function IdentifyModal({
  isOpen,
  initialUsername,
  onSave,
  onCancel,
  canCancel = false,
}: IdentifyModalProps) {
  const [val, setVal] = useState(initialUsername || '');
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
    onSave(clean);
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/85 p-4 select-none">
      <div className="w-full max-w-md border-2 border-green-400 bg-black p-6 box-glow text-green-300 font-mono space-y-4">
        {/* Header */}
        <div className="border-b-2 border-green-400 pb-2 flex justify-between items-center">
          <span className="font-bold text-sm tracking-wider text-green-200">
            === OPERATOR CALLSIGN REQUIRED ===
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
          THE BACKTRACK PROTOCOL REQUIRES A VALID OPERATOR CALLSIGN PRIOR TO ESTABLISHING TCP TRANSMISSION.
        </p>

        {/* Input Form */}
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-xs font-bold text-green-300 mb-1.5">
              SET USERNAME / HANDLE:
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

          <div className="pt-2 flex justify-end gap-2">
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

