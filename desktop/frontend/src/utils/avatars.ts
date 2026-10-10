export interface AvatarDef {
  id: number;
  label: string;
  glyph: string;
}

export const RETRO_AVATARS: AvatarDef[] = [
  { id: 1, label: 'INVADER', glyph: '👾' },
  { id: 2, label: 'CYBORG', glyph: '🤖' },
  { id: 3, label: 'SKULL', glyph: '💀' },
  { id: 4, label: 'TERMINAL', glyph: '📟' },
  { id: 5, label: 'FLOPPY', glyph: '💾' },
  { id: 6, label: 'JOYSTICK', glyph: '🕹️' },
  { id: 7, label: 'WIZARD', glyph: '🧙' },
  { id: 8, label: 'OPERATOR', glyph: '⚡' },
  { id: 9, label: 'BINGUS', glyph: '🐱' },
  { id: 10, label: 'GHOST', glyph: '👻' },
];

export function getAvatar(id?: number): AvatarDef {
  return (
    RETRO_AVATARS.find((a) => a.id === id) || {
      id: id || 1,
      label: 'OPERATOR',
      glyph: '👾',
    }
  );
}

