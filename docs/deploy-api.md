# Deploying the DTRules API

The DTRules API server powers the interactive poker demo on dtrules.com. This guide explains how to deploy it.

## Architecture

```
┌─────────────────────┐     ┌─────────────────────┐
│   Static Website    │     │   DTRules API       │
│   (GitHub Pages)    │────▶│   (Fly.io/VPS)      │
│   dtrules.com       │     │   dtrules-api.fly.dev│
└─────────────────────┘     └─────────────────────┘
```

- **Website**: Static Astro site deployed to GitHub Pages
- **API**: Go server running decision tables, deployed separately

## Option 1: Deploy to Fly.io (Recommended)

Fly.io offers a generous free tier perfect for the API server.

### First-time Setup

1. Install the Fly CLI:
   ```bash
   curl -L https://fly.io/install.sh | sh
   ```

2. Login to Fly:
   ```bash
   fly auth login
   ```

3. Launch the app (from repo root):
   ```bash
   fly launch --name dtrules-api
   ```
   - Select a region close to your users
   - Don't create a PostgreSQL database (not needed)
   - Don't create a Redis database (not needed)

4. Deploy:
   ```bash
   fly deploy
   ```

5. Check it's running:
   ```bash
   curl https://dtrules-api.fly.dev/api/health
   ```

### Subsequent Deployments

```bash
fly deploy
```

### GitHub Actions Auto-Deploy

To enable automatic deployment when API code changes:

1. Create a deploy token:
   ```bash
   fly tokens create deploy -x 999999h
   ```

2. Add the token to GitHub:
   - Go to repo Settings > Secrets > Actions
   - Add secret: `FLY_API_TOKEN` with the token value

3. Enable deployments:
   - Go to repo Settings > Variables > Actions
   - Add variable: `DEPLOY_API` with value `true`

## Option 2: Deploy to a VPS

If you prefer a traditional VPS (DigitalOcean, Linode, etc.):

### Build the Binary

```bash
# For Linux server
GOOS=linux GOARCH=amd64 go build -o dtrules-api ./cmd/api/
```

### Server Setup

1. Copy files to server:
   ```bash
   scp dtrules-api user@server:/opt/dtrules/
   scp -r sampleprojects user@server:/opt/dtrules/
   ```

2. Create systemd service (`/etc/systemd/system/dtrules-api.service`):
   ```ini
   [Unit]
   Description=DTRules API Server
   After=network.target

   [Service]
   Type=simple
   User=www-data
   WorkingDirectory=/opt/dtrules
   ExecStart=/opt/dtrules/dtrules-api -port 8080 -project-root /opt/dtrules/sampleprojects -cors-origin https://dtrules.com
   Restart=always
   RestartSec=5

   [Install]
   WantedBy=multi-user.target
   ```

3. Start the service:
   ```bash
   sudo systemctl enable dtrules-api
   sudo systemctl start dtrules-api
   ```

4. Configure nginx reverse proxy:
   ```nginx
   server {
       listen 443 ssl;
       server_name api.dtrules.com;

       ssl_certificate /etc/letsencrypt/live/api.dtrules.com/fullchain.pem;
       ssl_certificate_key /etc/letsencrypt/live/api.dtrules.com/privkey.pem;

       location / {
           proxy_pass http://127.0.0.1:8080;
           proxy_http_version 1.1;
           proxy_set_header Host $host;
           proxy_set_header X-Real-IP $remote_addr;
       }
   }
   ```

5. Point `api.dtrules.com` DNS to your server.

## Option 3: Run Locally for Development

```bash
# Build and run
go build -o dtrules-api ./cmd/api/
./dtrules-api -port 8080

# Or with Go directly
go run ./cmd/api/
```

The website will automatically use `localhost:8080` during development.

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/health` | GET | Health check |
| `/api/project/open` | POST | Load a project |
| `/api/execute` | POST | Run decision table |
| `/api/samples` | GET | List sample projects |

## Customizing the API URL

The website checks for the API URL in this order:

1. `window.__DTRULES_API__` - Runtime override
2. `PUBLIC_DTRULES_API` env var - Build-time config
3. `https://dtrules-api.fly.dev` - Production default
4. `http://localhost:8080` - Development fallback

To use a custom API URL, set the environment variable when building:

```bash
PUBLIC_DTRULES_API=https://api.yourdomain.com npm run build
```
