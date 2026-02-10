# Self-hosting fwdx (step-by-step)

This guide walks you through running your own fwdx tunneling server so you can expose local apps via HTTPS (e.g. `myapp.yourserver.com` → your laptop).

**Architecture:** The server has two listeners:

- **Web port** – HTTP/HTTPS for public traffic (proxy) and admin API.
- **gRPC port** – Tunnel connections (clients open a bidirectional gRPC stream here).

Nginx in front terminates TLS and forwards **443** → web and **4443** (gRPC stream) → grpc.

---

## 1. Prerequisites

- **A server** with a public IP (VPS: DigitalOcean, Linode, Hetzner, AWS, etc.).
- **A domain name** you control (e.g. `example.com`). You’ll use a subdomain for the tunnel server (e.g. `tunnel.example.com`).
- **Ports open** on the server:
  - **443** – Public HTTPS (browsers hit this; nginx proxies to fwdx web).
  - **4443** – Tunnel gRPC (clients connect here; nginx proxies to fwdx grpc).

---

## 2. Point DNS at your server

1. In your domain’s DNS (where you manage `example.com`):
2. Add an **A record**:
   - **Name:** `tunnel` (or whatever you want, e.g. `fwdx`, `t`)
   - **Value:** your server’s **public IPv4** (e.g. `203.0.113.10`)
   - **TTL:** 300 or 3600 is fine

Result: `tunnel.example.com` resolves to your server’s IP.

(Optional) Add a **wildcard** `*.tunnel.example.com` so that `myapp.tunnel.example.com`, `dev.tunnel.example.com`, etc. all go to the same server. fwdx routes by hostname; you don’t need individual A records per subdomain.

---

## 3. Get TLS certificates (HTTPS)

The public-facing server **requires** TLS. Easiest is **Let’s Encrypt** with Certbot.

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

- **If you use nginx** (recommended): Use **webroot** so nginx can keep serving port 80 for ACME. See step 4.1; get the cert after nginx is installed and the webroot location exists.
- **If you run fwdx directly** (no nginx): Certbot must bind to port 80. Stop any web server on 80, then:

```bash
sudo certbot certonly --standalone -d tunnel.example.com
```

- Use a real email for renewal notices.
- Agree to terms; choose whether to share email with EFF.
- If successful, certificates are in:
  - Certificate: `/etc/letsencrypt/live/tunnel.example.com/fullchain.pem`
  - Private key: `/etc/letsencrypt/live/tunnel.example.com/privkey.pem`

### 3.3 Let fwdx read the key (when not using nginx)

If fwdx terminates TLS (no nginx), it must read both files. Easiest is to run fwdx as root (e.g. systemd service), or copy certs to a directory the fwdx user can read:

```bash
sudo mkdir -p /etc/fwdx
sudo cp /etc/letsencrypt/live/tunnel.example.com/fullchain.pem /etc/fwdx/cert.pem
sudo cp /etc/letsencrypt/live/tunnel.example.com/privkey.pem /etc/fwdx/key.pem
sudo chmod 600 /etc/fwdx/key.pem
```

(If you run fwdx as a dedicated user, give that user read access to `/etc/fwdx`.)

**Renewal:** Certbot can renew with `sudo certbot renew`. Use a cron or systemd timer. After renewal, restart fwdx (or copy the new certs if you use copies).

---

## 4. Run behind nginx (recommended)

nginx terminates TLS on **443** (HTTPS) and **4443** (gRPC). It forwards:

- **443** → fwdx **web port** (HTTP)
- **4443** → fwdx **gRPC port** (TCP/stream)

fwdx runs with **no TLS** (plain HTTP and plain gRPC on localhost).

### 4.1 Certificate (webroot with existing nginx)

Use the same webroot your other certs use. Request a cert for the tunnel hostname:

```bash
sudo certbot certonly --webroot -w /var/www/acme -d tunnel.example.com --email you@example.com --agree-tos
```

(Replace `-w /var/www/acme` with your actual webroot. Ensure the server that handles `tunnel.example.com` on port 80 has `location /.well-known/acme-challenge/` for renewal.)

nginx will use this cert in the server blocks below; you do **not** copy it into fwdx.

### 4.2 Nginx: reverse proxy for HTTPS and gRPC

Add a **new server block** (e.g. in `/etc/nginx/sites-available/fwdx`) so that:

- Requests to `tunnel.example.com` and `*.tunnel.example.com` on **443** are proxied to fwdx’s web port.
- Connections to **4443** are proxied as a **stream** to fwdx’s gRPC port.

**HTTP/HTTPS (proxy + admin):**

```nginx
# fwdx: TLS on 443, reverse proxy to fwdx web port
server {
  listen 443 ssl;
  listen [::]:443 ssl;
  server_name tunnel.example.com *.tunnel.example.com;

  ssl_certificate     /etc/letsencrypt/live/tunnel.example.com/fullchain.pem;
  ssl_certificate_key /etc/letsencrypt/live/tunnel.example.com/privkey.pem;

  location / {
    proxy_pass http://127.0.0.1:8080;
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

**gRPC (tunnel) on 4443:**  
Add a **stream** block (in the main `nginx.conf`, top-level alongside `http { }`). Nginx terminates TLS on 4443 and forwards plain gRPC to fwdx’s grpc port (e.g. 4440):

```nginx
stream {
  server {
    listen 4443 ssl;
    listen [::]:4443 ssl;
    ssl_certificate     /etc/letsencrypt/live/tunnel.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/tunnel.example.com/privkey.pem;
    proxy_pass 127.0.0.1:4440;
    proxy_connect_timeout 10s;
    proxy_timeout 86400s;
  }
}
```

Use the same grpc port (e.g. **4440**) as in fwdx’s `--grpc-port`.

Enable the site if you use a separate file (e.g. `ln -s /etc/nginx/sites-available/fwdx /etc/nginx/sites-enabled/`), then:

```bash
sudo nginx -t && sudo systemctl reload nginx
```

### 4.3 fwdx: HTTP web + plain gRPC (no TLS)

Install fwdx (binary from [Releases](https://github.com/BRAVO68WEB/fwdx/releases) or build from source). Generate tokens:

```bash
CLIENT_TOKEN=$(openssl rand -hex 24)
ADMIN_TOKEN=$(openssl rand -hex 24)
echo "FWDX_CLIENT_TOKEN=$CLIENT_TOKEN"
echo "FWDX_ADMIN_TOKEN=$ADMIN_TOKEN"
```

Create the systemd service. fwdx listens on **web** (e.g. 8080) and **grpc** (e.g. 4440) with **no TLS**:

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
  --web-port 8080 \
  --grpc-port 4440 \
  --data-dir /var/lib/fwdx
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF
```

Replace `YOUR_CLIENT_TOKEN` and `YOUR_ADMIN_TOKEN` (and `tunnel.example.com` if different). Ports **8080** and **4440** must match what you used in nginx (`proxy_pass` and `proxy_pass 127.0.0.1:4440`). Then:

```bash
sudo systemctl daemon-reload
sudo systemctl enable fwdx
sudo systemctl start fwdx
sudo systemctl status fwdx
```

### 4.4 Firewall

Allow **80** (ACME), **443** (HTTPS), and **4443** (gRPC). Do **not** expose 8080 or 4440 publicly.

```bash
# If using ufw:
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw allow 4443/tcp
sudo ufw reload
```

### 4.5 Verify

From your laptop:

```bash
curl -sI https://tunnel.example.com/
```

You should get an HTTP response. **“no tunnel for this hostname”** is normal until a tunnel is registered. Then:

1. Set `FWDX_SERVER=https://tunnel.example.com` and `FWDX_TOKEN=your-client-token`.
2. Set **`FWDX_TUNNEL_PORT=4443`** (or in `~/.fwdx/client.json`: `"tunnel_port": 4443`) so the client connects to gRPC on 4443.
3. Run `fwdx tunnel create -l localhost:8080 -s myapp --name myapp` then `fwdx tunnel start myapp`.
4. Open **https://myapp.tunnel.example.com** in a browser.

Traffic flow: browser → nginx (443) → fwdx web → tunnel (gRPC over 4443) → your laptop → localhost:8080.

---

## 5. Option A: Run the binary directly (no nginx)

If you do **not** use nginx, fwdx can listen on 443 and 4443 with TLS. Get the certificate with `certbot certonly --standalone` (see 3.2) and run fwdx with `--tls-cert` and `--tls-key`.

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
CLIENT_TOKEN=$(openssl rand -hex 24)
ADMIN_TOKEN=$(openssl rand -hex 24)
echo "FWDX_CLIENT_TOKEN=$CLIENT_TOKEN"
echo "FWDX_ADMIN_TOKEN=$ADMIN_TOKEN"
```

Save these; you’ll use the client token on your laptop and the admin token for admin API and `fwdx domains add`.

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
  --web-port 443 \
  --grpc-port 4443 \
  --data-dir /var/lib/fwdx
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF
```

- Replace `YOUR_CLIENT_TOKEN` and `YOUR_ADMIN_TOKEN` with the values from step 5.2.
- Create data directory: `sudo mkdir -p /var/lib/fwdx`

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

Check logs: `sudo journalctl -u fwdx -f`. All server messages are prefixed with `[fwdx]`.

---

## 6. Option B: Run with Docker

You can run fwdx in Docker **behind nginx**: use the nginx config from section 4.2 and run the container with **web** and **grpc** ports, no TLS:

```bash
docker run -d --name fwdx \
  --restart unless-stopped \
  -p 127.0.0.1:8080:8080 \
  -p 127.0.0.1:4440:4440 \
  -v fwdx-data:/var/lib/fwdx \
  -e FWDX_HOSTNAME=tunnel.example.com \
  -e FWDX_CLIENT_TOKEN=your-client-token \
  -e FWDX_ADMIN_TOKEN=your-admin-token \
  ghcr.io/BRAVO68WEB/fwdx:latest \
  serve \
  --hostname tunnel.example.com \
  --web-port 8080 \
  --grpc-port 4440 \
  --data-dir /var/lib/fwdx
```

Nginx on the host proxies 443 → `127.0.0.1:8080` and 4443 → `127.0.0.1:4440`.

**Without nginx** (container binds 443 and 4443 with TLS): mount certs and pass TLS flags:

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
  --web-port 443 \
  --grpc-port 4443 \
  --data-dir /var/lib/fwdx
```

Open firewall for 443 and 4443 as in step 5.4.

---

## 7. Verify the server

From your **local machine** (not the server):

```bash
curl -sI https://tunnel.example.com/
```

You should get HTTP (e.g. 404 or “no tunnel for this hostname”). That’s expected until you register a tunnel. If you get a TLS or connection error, check DNS, firewall, and that fwdx (or nginx) is listening on 443.

---

## 8. Use a tunnel from your laptop (client)

### 8.1 Install fwdx locally

Download the binary for your OS from [Releases](https://github.com/BRAVO68WEB/fwdx/releases) (e.g. macOS or Windows), or build from source, and put it in your PATH.

### 8.2 Set server and token

```bash
export FWDX_SERVER=https://tunnel.example.com
export FWDX_TOKEN=your-client-token
```

- **Behind nginx (ports 443 + 4443):** Set the tunnel port so the client connects to gRPC on 4443:
  ```bash
  export FWDX_TUNNEL_PORT=4443
  ```
  Or in `~/.fwdx/client.json`: `"tunnel_port": 4443`. The client will use `https://tunnel.example.com:4443` for the gRPC tunnel.
- **Direct (fwdx on 443 and 4443):** Same: `FWDX_SERVER=https://tunnel.example.com` and `FWDX_TUNNEL_PORT=4443`.

### 8.3 Create and start a tunnel

Example: expose a local app on port 8080 as `myapp.tunnel.example.com`:

```bash
fwdx tunnel create -l localhost:8080 -s myapp --name myapp
fwdx tunnel start myapp
```

Then open in a browser: `https://myapp.tunnel.example.com`. Traffic goes: browser → server (443) → fwdx → gRPC tunnel to your laptop → localhost:8080.

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
| 3 | Certbot (webroot) for `tunnel.example.com`; nginx uses cert on 443 and 4443 |
| 4 | Nginx: server block for 443 → `proxy_pass` to fwdx web (e.g. 8080); stream block for 4443 → fwdx grpc (e.g. 4440) |
| 5 | Install fwdx; generate tokens; systemd with `--web-port 8080 --grpc-port 4440` (no TLS) |
| 6 | Start nginx and fwdx; from laptop set `FWDX_SERVER=https://tunnel.example.com`, `FWDX_TUNNEL_PORT=4443`, create and start tunnel |
| 7 | (Optional) Add custom domain with `fwdx domains add` and CNAME |

For more commands (list tunnels, manage domains, config, health), see the main [README](../README.md).
