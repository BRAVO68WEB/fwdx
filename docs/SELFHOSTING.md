# Self-hosting fwdx (step-by-step)

This guide walks you through running your own fwdx tunneling server so you can expose local apps via HTTPS (e.g. `myapp.yourserver.com` → your laptop).

---

## 1. Prerequisites

- **A server** with a public IP (VPS: DigitalOcean, Linode, Hetzner, AWS, etc.).
- **A domain name** you control (e.g. `example.com`). You’ll use a subdomain for the tunnel server (e.g. `tunnel.example.com`).
- **Ports open** on the server:
  - **443** – Public HTTPS (browsers and clients hit this for tunneled traffic).
  - **4443** – Tunnel registration (clients connect here to register tunnels). Optional if you use 443 for both (see below).

---

## 2. Point DNS at your server

1. In your domain’s DNS (where you manage `example.com`):
2. Add an **A record**:
   - **Name:** `tunnel` (or whatever you want, e.g. `fwdx`, `t`)
   - **Value:** your server’s **public IPv4** (e.g. `203.0.113.10`)
   - **TTL:** 300 or 3600 is fine

Result: `tunnel.example.com` resolves to your server’s IP.

(Optional) If you use a **subdomain for the server** (e.g. `tunnel.example.com`), you can later add a **wildcard** `*.tunnel.example.com` so that `myapp.tunnel.example.com`, `dev.tunnel.example.com`, etc. all go to the same server. For fwdx you don’t have to create individual A records for each subdomain—the server handles hostname routing.

---

## 3. Get TLS certificates (HTTPS)

The server **requires** TLS. Easiest is **Let’s Encrypt** with Certbot.

### 3.1 Install Certbot (on the server)

**Debian / Ubuntu:**

```bash
sudo apt update
sudo apt install -y certbot
```

**Alpine:**

```bash
sudo apk add certbot
```

**RHEL / Fedora / CentOS:**

```bash
sudo dnf install certbot
# or: sudo yum install certbot
```

### 3.2 Get a certificate for your hostname

Use the hostname you set in DNS (e.g. `tunnel.example.com`).

- **If you use nginx** (recommended): Use **webroot** so nginx can keep serving port 80 for ACME. See step 4.2 below; get the cert after nginx is installed and the webroot location exists.
- **If you run fwdx directly** (no nginx): Certbot must bind to port 80. Stop any web server on 80, then:

```bash
sudo certbot certonly --standalone -d tunnel.example.com
```

- Use a real email for renewal notices.
- Agree to terms; choose whether to share email with EFF.
- If successful, certificates are in:
  - Certificate: `/etc/letsencrypt/live/tunnel.example.com/fullchain.pem`
  - Private key: `/etc/letsencrypt/live/tunnel.example.com/privkey.pem`

### 3.3 Let fwdx read the key (permissions)

fwdx must read both files. Easiest is to run fwdx as root (e.g. systemd service as root), or copy certs to a directory the fwdx user can read:

```bash
sudo mkdir -p /etc/fwdx
sudo cp /etc/letsencrypt/live/tunnel.example.com/fullchain.pem /etc/fwdx/cert.pem
sudo cp /etc/letsencrypt/live/tunnel.example.com/privkey.pem /etc/fwdx/key.pem
sudo chmod 600 /etc/fwdx/key.pem
```

(If you run fwdx as a dedicated user, give that user read access to `/etc/fwdx`.)

**Renewal:** Certbot can renew with `sudo certbot renew`. Use a cron or systemd timer to run that periodically. After renewal, either restart fwdx or copy the new certs to `/etc/fwdx` again if you use copies.

---

## 4. Run behind nginx (recommended)

This section assumes **nginx is already running** on **443** with other sites. We add one more server block so nginx terminates TLS for the tunnel hostname and **reverse-proxies HTTP** to fwdx. fwdx runs with a **single port** in HTTP-only mode (no TLS); nginx holds the certificate.

- **One port:** Everything (public traffic and tunnel registration) goes over **443**. Clients use `https://tunnel.example.com` with no extra port.
- **No cert on fwdx:** Only nginx needs the certificate; fwdx listens on HTTP locally.

### 4.1 Certificate (webroot with existing nginx)

Use the same webroot your other certs use. Request a cert for the tunnel hostname:

```bash
sudo certbot certonly --webroot -w /var/www/acme -d tunnel.example.com --email you@example.com --agree-tos
```

(Replace `-w /var/www/acme` with your actual webroot. Ensure the server that handles `tunnel.example.com` on port 80 has `location /.well-known/acme-challenge/` for renewal.)

You do **not** copy the cert to fwdx; nginx will use it in the server block below.

### 4.2 Nginx: reverse proxy for the tunnel hostname

Add a **new server block** (e.g. in `/etc/nginx/sites-available/fwdx` or inside an existing config) so that requests to `tunnel.example.com` and `*.tunnel.example.com` are proxied to fwdx. Nginx terminates TLS; fwdx receives plain HTTP on one port (e.g. **4430**).

```nginx
# fwdx: TLS on 443, reverse proxy to local HTTP backend (single port)
server {
  listen 443 ssl;
  listen [::]:443 ssl;
  server_name tunnel.example.com *.tunnel.example.com;

  ssl_certificate     /etc/letsencrypt/live/tunnel.example.com/fullchain.pem;
  ssl_certificate_key /etc/letsencrypt/live/tunnel.example.com/privkey.pem;

  location / {
    proxy_pass http://127.0.0.1:4430;
    proxy_http_version 1.1;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_read_timeout 86400s;
    proxy_send_timeout 86400s;
  }
}
```

Enable the site if you use a separate file (e.g. `ln -s /etc/nginx/sites-available/fwdx /etc/nginx/sites-enabled/`), then:

```bash
sudo nginx -t && sudo systemctl reload nginx
```

### 4.3 fwdx: HTTP-only, single port

Install fwdx (binary from [Releases](https://github.com/BRAVO68WEB/fwdx/releases) or build from source). Generate tokens:

```bash
CLIENT_TOKEN=$(openssl rand -hex 24)
ADMIN_TOKEN=$(openssl rand -hex 24)
echo "FWDX_CLIENT_TOKEN=$CLIENT_TOKEN"
echo "FWDX_ADMIN_TOKEN=$ADMIN_TOKEN"
```

Create the systemd service. fwdx listens **only HTTP** on one port (**4430**); no TLS flags.

```bash
sudo mkdir -p /var/lib/fwdx
sudo tee /etc/systemd/system/fwdx.service << 'EOF'
[Unit]
Description=fwdx tunneling server (behind nginx)
After=network.target nginx.service

[Service]
Type=simple
ExecStart=/usr/local/bin/fwdx serve \
  --hostname tunnel.example.com \
  --client-token YOUR_CLIENT_TOKEN \
  --admin-token YOUR_ADMIN_TOKEN \
  --http-port 4430 \
  --data-dir /var/lib/fwdx
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF
```

Replace `YOUR_CLIENT_TOKEN` and `YOUR_ADMIN_TOKEN` (and `tunnel.example.com` if different). Then:

```bash
sudo systemctl daemon-reload
sudo systemctl enable fwdx
sudo systemctl start fwdx
sudo systemctl status fwdx
```

### 4.4 Firewall

If nginx already listens on 443, no new ports are needed. Ensure 80 (for ACME) and 443 are allowed. Do **not** expose 4430 publicly.

### 4.5 Verify

From your laptop:

```bash
curl -sI https://tunnel.example.com/
```

You should get an HTTP response (e.g. 404 or “no tunnel”). Then continue with **section 8** to create a tunnel. Clients use **one URL:** `FWDX_SERVER=https://tunnel.example.com` (port 443); tunnel registration uses the same port automatically.

---

## 5. Option A: Run the binary directly (no nginx)

If you do **not** use nginx, fwdx can listen on 443 and 4443 directly. Get the certificate with `certbot certonly --standalone` (see 3.2) and run fwdx on those ports.

### 5.1 Install fwdx on the server

Either download the latest binary from the [Releases](https://github.com/BRAVO68WEB/fwdx/releases) page (e.g. `fwdx-linux-amd64`) or build from source:

```bash
git clone https://github.com/BRAVO68WEB/fwdx.git
cd fwdx
go build -o fwdx .
sudo mv fwdx /usr/local/bin/
```

### 5.2 Create tokens

Generate two random secrets: one for **clients** (tunnel registration) and one for **admin** (manage tunnels, domains). Keep them safe.

```bash
# Example (use your own in production):
CLIENT_TOKEN=$(openssl rand -hex 24)
ADMIN_TOKEN=$(openssl rand -hex 24)
echo "FWDX_CLIENT_TOKEN=$CLIENT_TOKEN"
echo "FWDX_ADMIN_TOKEN=$ADMIN_TOKEN"
```

Save these; you’ll use the client token on your laptop and the admin token for `fwdx manage` and `fwdx domains add`.

### 5.3 Create a systemd service

Create a service file (replace `tunnel.example.com` and paths if different):

```bash
sudo tee /etc/systemd/system/fwdx.service << 'EOF'
[Unit]
Description=fwdx tunneling server
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/fwdx serve \
  --hostname tunnel.example.com \
  --client-token YOUR_CLIENT_TOKEN \
  --admin-token YOUR_ADMIN_TOKEN \
  --tls-cert /etc/fwdx/cert.pem \
  --tls-key /etc/fwdx/key.pem \
  --https-port 443 \
  --tunnel-port 4443 \
  --data-dir /var/lib/fwdx
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF
```

- Replace `YOUR_CLIENT_TOKEN` and `YOUR_ADMIN_TOKEN` with the values from step 5.2.
- Create data directory: `sudo mkdir -p /var/lib/fwdx`

If you want to use **only port 443** for both public and tunnel traffic (no 4443), set `--tunnel-port 443` and ensure no other service uses 443. Then clients will connect to `https://tunnel.example.com` (port 443) for registration.

### 5.4 Open firewall ports

```bash
# If using ufw:
sudo ufw allow 443/tcp
sudo ufw allow 4443/tcp
sudo ufw reload

# If using firewalld:
sudo firewall-cmd --permanent --add-service=https
sudo firewall-cmd --permanent --add-port=4443/tcp
sudo firewall-cmd --reload
```

### 5.5 Start and enable fwdx

```bash
sudo systemctl daemon-reload
sudo systemctl enable fwdx
sudo systemctl start fwdx
sudo systemctl status fwdx
```

Check logs: `sudo journalctl -u fwdx -f`

---

## 6. Option B: Run with Docker

You can run fwdx in Docker **behind nginx** (recommended): use the nginx reverse-proxy config from section 4.2 and run the container with **one** port, HTTP-only:

```bash
docker run -d --name fwdx \
  --restart unless-stopped \
  -p 127.0.0.1:4430:4430 \
  -v fwdx-data:/var/lib/fwdx \
  -e FWDX_HOSTNAME=tunnel.example.com \
  -e FWDX_CLIENT_TOKEN=your-client-token \
  -e FWDX_ADMIN_TOKEN=your-admin-token \
  ghcr.io/BRAVO68WEB/fwdx:latest \
  serve \
  --hostname tunnel.example.com \
  --http-port 4430 \
  --data-dir /var/lib/fwdx
```

Nginx on the host proxies `https://tunnel.example.com` to `http://127.0.0.1:4430`. No TLS inside the container.

Or run **without nginx** (container binds 443/4443 on the host): use the instructions below and publish 443 and 4443.

### 6.1 Prepare TLS and tokens

On the server, create a directory for config and put certs and a script (or env file) there:

```bash
sudo mkdir -p /opt/fwdx
sudo cp /etc/fwdx/cert.pem /opt/fwdx/
sudo cp /etc/fwdx/key.pem /opt/fwdx/
```

Create an env file with your tokens (replace with your real tokens):

```bash
sudo tee /opt/fwdx/env << 'EOF'
FWDX_HOSTNAME=tunnel.example.com
FWDX_CLIENT_TOKEN=your-client-token-here
FWDX_ADMIN_TOKEN=your-admin-token-here
EOF
sudo chmod 600 /opt/fwdx/env
```

### 6.2 Run the container

The fwdx image reads `FWDX_HOSTNAME`, `FWDX_CLIENT_TOKEN`, and `FWDX_ADMIN_TOKEN` from the environment. Use the same image from GitHub Container Registry (replace `BRAVO68WEB` with the repo owner if different):

```bash
docker run -d --name fwdx \
  --restart unless-stopped \
  -p 443:443 -p 4443:4443 \
  -v /opt/fwdx/cert.pem:/etc/fwdx/cert.pem:ro \
  -v /opt/fwdx/key.pem:/etc/fwdx/key.pem:ro \
  -v fwdx-data:/var/lib/fwdx \
  --env-file /opt/fwdx/env \
  ghcr.io/BRAVO68WEB/fwdx:latest \
  serve \
  --hostname tunnel.example.com \
  --tls-cert /etc/fwdx/cert.pem \
  --tls-key /etc/fwdx/key.pem \
  --data-dir /var/lib/fwdx
```

Or pass variables explicitly instead of `--env-file`:

```bash
docker run -d --name fwdx \
  --restart unless-stopped \
  -p 443:443 -p 4443:4443 \
  -v /opt/fwdx/cert.pem:/etc/fwdx/cert.pem:ro \
  -v /opt/fwdx/key.pem:/etc/fwdx/key.pem:ro \
  -v fwdx-data:/var/lib/fwdx \
  -e FWDX_HOSTNAME=tunnel.example.com \
  -e FWDX_CLIENT_TOKEN=your-client-token \
  -e FWDX_ADMIN_TOKEN=your-admin-token \
  ghcr.io/BRAVO68WEB/fwdx:latest \
  serve \
  --hostname tunnel.example.com \
  --tls-cert /etc/fwdx/cert.pem \
  --tls-key /etc/fwdx/key.pem \
  --data-dir /var/lib/fwdx
```

Open firewall for 443 and 4443 as in step 5.4.

---

## 7. Verify the server

From your **local machine** (not the server):

```bash
curl -sI https://tunnel.example.com/
```

You should get HTTP (e.g. 404 or “no tunnel for this hostname”). That’s expected until you register a tunnel. If you get a TLS or connection error, check DNS, firewall, and that fwdx is listening on 443.

---

## 8. Use a tunnel from your laptop (client)

### 8.1 Install fwdx locally

Download the binary for your OS from [Releases](https://github.com/BRAVO68WEB/fwdx/releases) (e.g. macOS or Windows), or build from source, and put it in your PATH.

### 8.2 Set server and token

```bash
export FWDX_SERVER=https://tunnel.example.com
export FWDX_TOKEN=your-client-token
```

Use the same **client token** you set on the server. When the server is behind nginx on 443, the client uses this URL only; the tunnel port (443) is taken from the URL.

### 8.3 Create and start a tunnel

Example: expose a local app on port 8080 as `myapp.tunnel.example.com`:

```bash
fwdx tunnel create -l localhost:8080 -s myapp --name myapp
fwdx tunnel start myapp
```

Then open in a browser: `https://myapp.tunnel.example.com`. Traffic goes: browser → nginx (443) → fwdx (HTTP backend) → long-lived connection from your laptop → localhost:8080.

### 8.4 Optional: run tunnel in background

`fwdx tunnel start myapp` runs in the foreground. To run in the background, use your OS’s usual way (e.g. `nohup ... &`, or a systemd user service / launchd / Windows service).

---

## 9. Optional: Custom domains

If you want a tunnel on your own domain (e.g. `app.mycompany.com` instead of `myapp.tunnel.example.com`):

### 9.1 Add the domain on the server (one-time)

From any machine that can reach the server and has the **admin token**:

```bash
export FWDX_ADMIN_TOKEN=your-admin-token
fwdx domains add mycompany.com --server https://tunnel.example.com --admin-token $FWDX_ADMIN_TOKEN
```

The command prints DNS instructions: create a **CNAME** record:

- Name: `*` (or `app` if you only want `app.mycompany.com`)
- Value: `tunnel.example.com`

### 9.2 Create a tunnel with the custom URL

On your laptop:

```bash
fwdx tunnel create -l localhost:3000 -u app.mycompany.com --name myapp
fwdx tunnel start myapp
```

Then `https://app.mycompany.com` will proxy to your local port 3000.

---

## 10. Summary checklist

| Step | Action |
|------|--------|
| 1 | Get a VPS and a domain |
| 2 | A record: `tunnel.example.com` → server IP |
| 3 | Certbot (webroot) for `tunnel.example.com`; nginx keeps using 443 for other sites |
| 4 | Add nginx server block: TLS for `tunnel.example.com` + `*.tunnel.example.com`, `proxy_pass http://127.0.0.1:4430` |
| 5 | Install fwdx; generate tokens; systemd with `--http-port 4430` (no TLS) |
| 6 | Start nginx and fwdx; from laptop set `FWDX_SERVER=https://tunnel.example.com`, create and start tunnel |
| 7 | (Optional) Add custom domain with `fwdx domains add` and CNAME |

For more commands (list tunnels, manage domains, config, health), see the main [README](../README.md).
