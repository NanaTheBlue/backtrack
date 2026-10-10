interface SettingsModalProps {
  isOpen: boolean;
  onClose: () => void;
  serverAddr: string;
  setServerAddr: (addr: string) => void;
  username: string;
  setUsername: (name: string) => void;
}

export default function SettingsModal({
  isOpen,
  onClose,
  serverAddr,
  setServerAddr,
  username,
  setUsername,
}: SettingsModalProps) {
  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/85 p-4 select-none">
      <div className="w-full max-w-md border-2 border-green-400 bg-black p-5 box-glow text-green-300 font-mono space-y-4">
        {/* Title */}
        <div className="flex items-center justify-between border-b-2 border-green-400 pb-2">
          <span className="font-bold text-sm tracking-wider text-green-200">
            [SYS] CONFIGURATION MENU
          </span>
          <button
            onClick={onClose}
            className="text-xs border border-green-400 px-1.5 py-0.5 hover:bg-green-400 hover:text-black cursor-pointer font-bold"
          >
            [X]
          </button>
        </div>

        {/* Options */}
        <div className="space-y-3.5 text-xs tracking-wider">
          <div>
            <label className="block text-green-400 mb-1 font-bold">
              OPERATOR USERNAME / CALLSIGN:
            </label>
            <input
              type="text"
              value={username}
              maxLength={24}
              onChange={(e) => setUsername(e.target.value.toUpperCase())}
              placeholder="YOUR CALLSIGN"
              className="w-full bg-black border-2 border-green-500/80 p-1.5 text-green-200 outline-none focus:border-green-300"
            />
          </div>

          <div>
            <label className="block text-green-400 mb-1 font-bold">
              TCP SERVER TARGET:
            </label>
            <input
              type="text"
              value={serverAddr}
              onChange={(e) => setServerAddr(e.target.value)}
              className="w-full bg-black border-2 border-green-500/80 p-1.5 text-green-200 outline-none focus:border-green-300"
            />
          </div>

          <div className="pt-2 border-t border-green-500/30 flex justify-between items-center">
            <span>CRT SCANLINE EMULATION:</span>
            <span className="text-green-400 font-bold">[ACTIVE]</span>
          </div>

        

          <div className="flex justify-between items-center">
            <span>PROTOCOL SPEC:</span>
            <span className="text-green-400 font-bold">BACKTRACK v1.0</span>
          </div>
        </div>

        {/* Footer */}
        <div className="pt-3 border-t-2 border-green-400 flex justify-end">
          <button
            onClick={onClose}
            className="border-2 border-green-400 px-4 py-1.5 text-xs font-bold tracking-widest bg-green-950 hover:bg-green-400 hover:text-black transition-colors cursor-pointer"
          >
            APPLY & CLOSE [↵]
          </button>
        </div>
      </div>
    </div>
  );
}
