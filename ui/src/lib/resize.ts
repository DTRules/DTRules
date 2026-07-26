/**
 * useStoredWidth - a draggable panel width persisted to localStorage.
 *
 * Returns the current width and a mouse-down handler for a divider bar.
 * `fromRight` measures from the window's right edge (for right-docked
 * panels like the debugger's entity-stack rail).
 */
import { useCallback, useState } from 'react';

export function useStoredWidth(key: string, initial: number, min: number, max: number, fromRight = false) {
  const [width, setWidth] = useState(() => {
    const saved = parseInt(localStorage.getItem(key) || '', 10);
    return Number.isFinite(saved) ? Math.min(max, Math.max(min, saved)) : initial;
  });

  const onDragStart = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault();
      const move = (ev: MouseEvent) => {
        const w = fromRight ? window.innerWidth - ev.clientX : ev.clientX;
        const clamped = Math.min(max, Math.max(min, w));
        setWidth(clamped);
        localStorage.setItem(key, String(clamped));
      };
      const up = () => {
        window.removeEventListener('mousemove', move);
        window.removeEventListener('mouseup', up);
        document.body.style.cursor = '';
        document.body.style.userSelect = '';
      };
      document.body.style.cursor = 'col-resize';
      document.body.style.userSelect = 'none';
      window.addEventListener('mousemove', move);
      window.addEventListener('mouseup', up);
    },
    [key, min, max, fromRight]
  );

  return { width, onDragStart };
}
