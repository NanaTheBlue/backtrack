export default function CrtOverlay() {
  return (
    <>
      {/* Horizontal CRT Scanline grid */}
      <div className="pointer-events-none fixed inset-0 crt-scanlines z-50 opacity-90" />
      {/* Curved glass screen vignette shadow */}
      <div className="pointer-events-none fixed inset-0 crt-vignette z-50" />
      {/* Subtle sweeping electron beam */}
      <div className="pointer-events-none fixed inset-0 crt-beam h-32 z-40" />
    </>
  );
}

