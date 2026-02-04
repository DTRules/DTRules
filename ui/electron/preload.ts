import { contextBridge, ipcRenderer } from 'electron';

// Expose a limited API to the renderer process
contextBridge.exposeInMainWorld('electronAPI', {
  // Dialog methods
  openDirectory: () => ipcRenderer.invoke('dialog:openDirectory'),
  openFile: (filters?: { name: string; extensions: string[] }[]) =>
    ipcRenderer.invoke('dialog:openFile', filters),
  saveFile: (defaultPath?: string) =>
    ipcRenderer.invoke('dialog:saveFile', defaultPath),

  // Platform info
  platform: process.platform,
});

// Type definitions for the exposed API
declare global {
  interface Window {
    electronAPI: {
      openDirectory: () => Promise<string | null>;
      openFile: (filters?: { name: string; extensions: string[] }[]) => Promise<string | null>;
      saveFile: (defaultPath?: string) => Promise<string | null>;
      platform: NodeJS.Platform;
    };
  }
}
