# DTRules UI Quick Start Guide

Get the visual UI running in under 5 minutes.

## Prerequisites

- **Node.js 18+** - [Download](https://nodejs.org/)
- **Go 1.21+** - [Download](https://go.dev/dl/)

Verify installation:
```bash
node --version   # Should show v18.x.x or higher
go version       # Should show go1.21.x or higher
```

## Step 1: Start the Backend

Open a terminal and run:

```bash
cd go
go run ./cmd/api
```

You should see:
```
DTRules API Server starting on :8080
```

Leave this terminal running.

## Step 2: Start the Frontend

Open a **new terminal** and run:

```bash
cd ui
npm install      # First time only - installs dependencies
npm run dev
```

You should see:
```
VITE v5.x.x  ready in xxx ms

➜  Local:   http://localhost:5173/
```

## Step 3: Open the UI

1. Open your browser to **http://localhost:5173**
2. You'll see the DTRules welcome screen

## Step 4: Open a Project

Click **"Open CHIP Sample Project"** (or "Open Custom Project") and enter the path to a DTRules project:

| OS | Example Path |
|----|--------------|
| Linux | `/home/username/DTRules/sampleprojects/CHIP/xml` |
| macOS | `/Users/username/DTRules/sampleprojects/CHIP/xml` |
| Windows | `C:\Users\username\DTRules\sampleprojects\CHIP\xml` |

**Tip:** Use the absolute path to your cloned DTRules repository.

## Step 5: Explore

Once the project loads, you can:

- **Left Panel** - Browse entities and decision tables
- **Entity Tab** - View/edit entity definitions (EDD)
- **Decision Table Tab** - View/edit decision tables with color-coded cells
- **Test Tab** - Execute rules with test data and see traces
- **Tree Tab** - Visualize decision table call hierarchy

## Optional: Pre-configure the CHIP Path

To skip entering the path each time:

1. Copy the example environment file:
   ```bash
   cd ui
   cp .env.example .env.local
   ```

2. Edit `.env.local` and set your path:
   ```
   VITE_API_URL=http://localhost:8080/api
   VITE_CHIP_PROJECT_PATH=/absolute/path/to/DTRules/sampleprojects/CHIP/xml
   ```

3. Restart the frontend (`npm run dev`)

Now "Open CHIP Sample Project" will load automatically.

## Troubleshooting

### "Failed to fetch" or "Network Error"

The backend isn't running. Make sure you started it:
```bash
cd go
go run ./cmd/api
```

### "npm install" fails

Try clearing the cache:
```bash
cd ui
rm -rf node_modules package-lock.json
npm install
```

### Port already in use

Backend (8080):
```bash
# Find and kill the process using port 8080
lsof -i :8080
kill <PID>
```

Frontend (5173):
```bash
# Vite will automatically try the next port (5174, etc.)
# Or kill the existing process
lsof -i :5173
kill <PID>
```

### Project won't open

- Verify the path exists and contains `*_edd.xml` and `*_dt.xml` files
- Use an absolute path (starting with `/` on Linux/Mac or `C:\` on Windows)
- Check the backend terminal for error messages

## Architecture

```
┌─────────────┐     HTTP      ┌─────────────┐
│   Browser   │◄────────────►│  Go Backend │
│  (React UI) │   :5173      │  (API)      │
│             │              │   :8080     │
└─────────────┘              └──────┬──────┘
                                    │
                                    ▼
                             ┌─────────────┐
                             │  XML Files  │
                             │  (EDD, DT)  │
                             └─────────────┘
```

## Next Steps

- Read the [UI README](ui/README.md) for detailed feature documentation
- Explore the [CHIP sample project](sampleprojects/CHIP/) to understand DTRules concepts
- Check the [Go README](go/README.md) for CLI usage and advanced options

## Quick Reference

| Action | Command |
|--------|---------|
| Start backend | `cd go && go run ./cmd/api` |
| Start frontend | `cd ui && npm run dev` |
| Build frontend | `cd ui && npm run build` |
| Run Go tests | `cd go && go test ./...` |
| Type check UI | `cd ui && npm run typecheck` |
