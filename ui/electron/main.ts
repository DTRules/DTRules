import { app, BrowserWindow, ipcMain, dialog } from 'electron';
import path from 'path';
import { spawn, ChildProcess } from 'child_process';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

let mainWindow: BrowserWindow | null = null;
let goProcess: ChildProcess | null = null;

// Find the Go backend executable
function findBackendPath(): string {
  const isDev = !app.isPackaged;

  if (isDev) {
    // In development, look for the Go binary in the project directory
    return path.join(__dirname, '../../go/cmd/dtrules-server/dtrules-server');
  } else {
    // In production, look in the resources directory
    return path.join(process.resourcesPath, 'backend/dtrules-server');
  }
}

// Start the Go backend server
function startBackend(): void {
  const backendPath = findBackendPath();

  console.log('Starting backend from:', backendPath);

  goProcess = spawn(backendPath, ['-port', '8080'], {
    stdio: ['ignore', 'pipe', 'pipe'],
  });

  goProcess.stdout?.on('data', (data) => {
    console.log(`Backend: ${data}`);
  });

  goProcess.stderr?.on('data', (data) => {
    console.error(`Backend error: ${data}`);
  });

  goProcess.on('close', (code) => {
    console.log(`Backend exited with code ${code}`);
    goProcess = null;
  });

  goProcess.on('error', (err) => {
    console.error('Failed to start backend:', err);
    goProcess = null;
  });
}

// Stop the Go backend server
function stopBackend(): void {
  if (goProcess) {
    goProcess.kill();
    goProcess = null;
  }
}

function createWindow(): void {
  mainWindow = new BrowserWindow({
    width: 1400,
    height: 900,
    minWidth: 1024,
    minHeight: 600,
    webPreferences: {
      preload: path.join(__dirname, 'preload.js'),
      nodeIntegration: false,
      contextIsolation: true,
    },
    title: 'DTRules',
  });

  // Load the app
  if (process.env.VITE_DEV_SERVER_URL) {
    mainWindow.loadURL(process.env.VITE_DEV_SERVER_URL);
    mainWindow.webContents.openDevTools();
  } else {
    mainWindow.loadFile(path.join(__dirname, '../dist/index.html'));
  }

  mainWindow.on('closed', () => {
    mainWindow = null;
  });
}

// IPC handlers
ipcMain.handle('dialog:openDirectory', async () => {
  if (!mainWindow) return null;

  const result = await dialog.showOpenDialog(mainWindow, {
    properties: ['openDirectory'],
    title: 'Select DTRules Project Directory',
  });

  return result.canceled ? null : result.filePaths[0];
});

ipcMain.handle('dialog:openFile', async (_, filters?: { name: string; extensions: string[] }[]) => {
  if (!mainWindow) return null;

  const result = await dialog.showOpenDialog(mainWindow, {
    properties: ['openFile'],
    filters: filters || [
      { name: 'XML Files', extensions: ['xml'] },
      { name: 'All Files', extensions: ['*'] },
    ],
  });

  return result.canceled ? null : result.filePaths[0];
});

ipcMain.handle('dialog:saveFile', async (_, defaultPath?: string) => {
  if (!mainWindow) return null;

  const result = await dialog.showSaveDialog(mainWindow, {
    defaultPath,
    filters: [
      { name: 'XML Files', extensions: ['xml'] },
      { name: 'All Files', extensions: ['*'] },
    ],
  });

  return result.canceled ? null : result.filePath;
});

// App lifecycle
app.whenReady().then(() => {
  startBackend();

  // Wait a bit for the backend to start before creating the window
  setTimeout(createWindow, 1000);

  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) {
      createWindow();
    }
  });
});

app.on('window-all-closed', () => {
  stopBackend();
  if (process.platform !== 'darwin') {
    app.quit();
  }
});

app.on('before-quit', () => {
  stopBackend();
});
