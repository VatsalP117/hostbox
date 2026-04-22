# Self-Hosting Guide

Hostbox is designed to run on a Linux VM with Docker and a domain you control.

The default routing model is:

- `hostbox.example.com` -> dashboard
- `example.com` -> dashboard
- `project.example.com` -> production deployment
- `project-branch.example.com` and `project-suffix.example.com` -> preview deployments

That means the minimum DNS setup is still:

```text
A     example.com    -> YOUR_SERVER_IP
A     *.example.com  -> YOUR_SERVER_IP
```

If you choose a dashboard host outside `*.example.com`, add a separate record for that host too.

## Requirements

| Resource | Minimum | Recommended |
| --- | --- | --- |
| CPU | 1 vCPU | 2 vCPU |
| RAM | 512 MB | 1 GB+ |
| Disk | 10 GB | 20 GB+ |
| OS | Ubuntu 22.04+ / Debian 12+ | Ubuntu 24.04 |

You also need:

- Docker Engine and the Docker Compose plugin
- ports `80/tcp`, `443/tcp`, and `443/udp` open to the internet
- a public IPv4 address
- a GitHub App only if you want git-push deployments

## Before you install

1. Point your apex and wildcard DNS records at the VM.
2. Keep Cloudflare wildcard records set to **DNS only** if you use Cloudflare.
3. Decide whether you want wildcard certificates up front:
   - `DNS_PROVIDER=none`: Caddy issues certificates per host on demand with HTTP-01.
   - `DNS_PROVIDER=cloudflare|route53|digitalocean`: Caddy can request wildcard certificates with DNS-01.

Without a DNS provider, new subdomains still work, but each hostname must be publicly reachable on ports 80 and 443 when it is first requested.

## Automated install

The installer is the easiest path on a fresh Linux VM:

```bash
curl -fsSL https://raw.githubusercontent.com/VatsalP117/hostbox/main/scripts/install.sh | sudo bash
```

It will:

1. install Docker if needed
2. ask for your root domain, dashboard host, ACME email, and optional DNS-provider credentials
3. clone Hostbox source into `/opt/hostbox`
4. create `/opt/hostbox/{data,deployments,logs,cache,tmp}`
5. set runtime directory ownership for the Hostbox container user
6. generate secrets and a `.env`
7. detect the Docker socket group and wire it into compose so build containers can start
8. build and start Hostbox locally from source

After the install finishes, open `https://hostbox.example.com` and create the first admin account.

## Built-in monitoring

Hostbox now includes an admin monitoring view that surfaces the metrics you typically need first when operating a self-hosted deployment platform:

- component health for the API, database, Docker, and Caddy
- CPU, memory, disk, and build-queue pressure
- storage breakdown across deployments, logs, database, backups, and cache
- recent deployment success rate and average build duration
- 24-hour trends for resource usage and queued builds

This view is meant for fast operator diagnosis when builds start backing up, the VM is running hot, or storage pressure is approaching a deployment failure.

### Fresh install on a VM that already has the right DNS records

If DNS already points at this VM and you just want to replace an older or broken Hostbox install, you do **not** need to change DNS. The IP address stays the same, so the records can stay the same too.

For a testing VM where Hostbox data is disposable:

```bash
sudo systemctl stop caddy 2>/dev/null || true
cd /opt/hostbox 2>/dev/null && sudo docker compose down --remove-orphans || true
sudo rm -rf /opt/hostbox
curl -fsSL https://raw.githubusercontent.com/VatsalP117/hostbox/main/scripts/install.sh | sudo bash
```

### What the installer puts on disk

Hostbox now uses **host-visible absolute paths** for all build bind mounts. After install, these paths should exist:

```text
/opt/hostbox/data
/opt/hostbox/data/backups
/opt/hostbox/deployments
/opt/hostbox/logs
/opt/hostbox/cache
/opt/hostbox/tmp
```

The generated `.env` should point at those same paths.

## Manual install

Use this if you want to review everything yourself.

### 1. Clone the source

```bash
git clone https://github.com/VatsalP117/hostbox.git /opt/hostbox
cd /opt/hostbox
sudo mkdir -p /opt/hostbox/{data/backups,deployments,logs,cache,tmp}
sudo chown -R 1000:1000 /opt/hostbox/data /opt/hostbox/deployments /opt/hostbox/logs /opt/hostbox/cache /opt/hostbox/tmp
```

### 2. Create secrets and detect the Docker socket group

```bash
JWT_SECRET="$(openssl rand -hex 32)"
ENCRYPTION_KEY="$(openssl rand -hex 32)"
GITHUB_WEBHOOK_SECRET="$(openssl rand -hex 32)"
DOCKER_GID="$(stat -c '%g' /var/run/docker.sock)"
```

### 3. Create `.env`

```bash
cat > .env <<EOF
PLATFORM_DOMAIN=example.com
DASHBOARD_DOMAIN=hostbox.example.com
PLATFORM_HTTPS=true
PLATFORM_NAME=Hostbox

JWT_SECRET=${JWT_SECRET}
ENCRYPTION_KEY=${ENCRYPTION_KEY}
GITHUB_WEBHOOK_SECRET=${GITHUB_WEBHOOK_SECRET}

DATABASE_PATH=/opt/hostbox/data/hostbox.db
DEPLOYMENTS_DIR=/opt/hostbox/deployments
LOGS_DIR=/opt/hostbox/logs
CACHE_DIR=/opt/hostbox/cache
BACKUP_DIR=/opt/hostbox/data/backups
CLONE_BASE_DIR=/opt/hostbox/tmp
DEPLOYMENT_BASE_DIR=/opt/hostbox/deployments
LOG_BASE_DIR=/opt/hostbox/logs
DOCKER_GID=${DOCKER_GID}

ACME_EMAIL=admin@example.com
DNS_PROVIDER=none

LOG_LEVEL=info
LOG_FORMAT=json
BUILD_MEMORY_MB=1024
EOF
```

### 4. Build and start Hostbox

```bash
docker compose up -d --build --remove-orphans
```

### 5. Verify the API

```bash
docker compose exec -T hostbox wget --no-verbose --tries=1 -O- http://127.0.0.1:8080/api/v1/health
docker compose exec -T hostbox wget --no-verbose --tries=1 -O- http://127.0.0.1:8080/api/v1/setup/status
```

### 6. Open the dashboard

Visit `https://hostbox.example.com`.

## Configuration

Hostbox reads runtime configuration from `.env`.

| Variable | Required | Notes |
| --- | --- | --- |
| `PLATFORM_DOMAIN` | Yes | Root domain used for production and preview hosts. |
| `DASHBOARD_DOMAIN` | No | Defaults to `hostbox.${PLATFORM_DOMAIN}` when blank. |
| `PLATFORM_HTTPS` | No | Keep `true` for real deployments. |
| `PLATFORM_NAME` | No | UI display name. |
| `JWT_SECRET` | Yes | At least 32 characters. |
| `ENCRYPTION_KEY` | Yes | Use `openssl rand -hex 32`. |
| `DATABASE_PATH` | Yes | For Docker installs, use a host-visible absolute path like `/opt/hostbox/data/hostbox.db`. |
| `DEPLOYMENTS_DIR` | No | For Docker installs, use a host-visible absolute path like `/opt/hostbox/deployments`. |
| `LOGS_DIR` | No | For Docker installs, use a host-visible absolute path like `/opt/hostbox/logs`. |
| `CACHE_DIR` | No | For Docker installs, use a host-visible absolute path like `/opt/hostbox/cache`. |
| `CLONE_BASE_DIR` | No | For Docker installs, use a host-visible absolute path like `/opt/hostbox/tmp`. |
| `DEPLOYMENT_BASE_DIR` | No | Usually matches `DEPLOYMENTS_DIR`. |
| `LOG_BASE_DIR` | No | Usually matches `LOGS_DIR`. |
| `DOCKER_GID` | Yes for Docker installs | Group ID of `/var/run/docker.sock` on the host. Needed for build access from inside the container. |
| `ACME_EMAIL` | Yes | Email used for Let's Encrypt / ACME. |
| `DNS_PROVIDER` | No | `none`, `cloudflare`, `route53`, or `digitalocean`. |
| `LOG_LEVEL` | No | `debug`, `info`, `warn`, or `error`. |
| `BUILD_MEMORY_MB` | No | Per-build container memory limit in MB. Increase this for large Node.js workspaces if builds are killed with exit code `137`. |
| `GITHUB_APP_ID` | No | Optional advanced override. The dashboard can create and store a GitHub App for you. |
| `GITHUB_APP_SLUG` | No | Optional advanced override. |
| `GITHUB_APP_PEM` | No | Optional advanced override. |
| `GITHUB_WEBHOOK_SECRET` | No | Optional advanced override. |

### Why the path variables matter

Hostbox creates Docker build containers through the **host Docker daemon**. That means the source paths used for bind mounts must be:

1. absolute
2. valid on the VM host filesystem

That is why Docker installs should use `/opt/hostbox/...` for database, logs, deployments, cache, and clone paths.

The runtime directories also need to be writable by the Hostbox container user (`uid 1000`), which is why the installer and update script chown those directories during setup.

### DNS provider credentials

**Cloudflare**

```env
DNS_PROVIDER=cloudflare
CF_API_TOKEN=your-cloudflare-api-token
```

**Route53**

```env
DNS_PROVIDER=route53
AWS_ACCESS_KEY_ID=your-access-key-id
AWS_SECRET_ACCESS_KEY=your-secret-access-key
AWS_HOSTED_ZONE_ID=your-hosted-zone-id
```

**DigitalOcean**

```env
DNS_PROVIDER=digitalocean
DO_AUTH_TOKEN=your-digitalocean-token
```

## Networking model

In the production compose file:

- only Caddy publishes ports `80` and `443`
- the Hostbox API stays internal on the Docker network
- Caddy talks to Hostbox at `hostbox:8080`
- Hostbox talks to the Caddy admin API at `http://caddy:2019`

So from the VM host, use `docker compose exec ...` for direct API health checks instead of `curl http://localhost:8080/...`.

## GitHub setup

The dashboard handles GitHub setup for the normal single-user flow:

1. Open **New Project**.
2. Click **Connect GitHub**.
3. GitHub creates a private Hostbox GitHub App for this instance.
4. Choose the repositories Hostbox can access.
5. Return to Hostbox and select a repository from the dropdown.

Hostbox stores the generated GitHub App ID, slug, private key, and webhook secret in the database. The private key and webhook secret are encrypted with `ENCRYPTION_KEY`.

You can still provide `GITHUB_APP_ID`, `GITHUB_APP_SLUG`, `GITHUB_APP_PEM`, and `GITHUB_WEBHOOK_SECRET` in `.env` if you want to manage the GitHub App manually.

## Operations

### View logs

```bash
docker compose logs -f hostbox
docker compose logs -f caddy
```

### Update

For normal updates:

```bash
bash scripts/update.sh
```

During testing, if you want to rebuild from current `main` and do **not** care about preserving Hostbox runtime data:

```bash
bash scripts/update.sh --fresh
```

That fresh update will:

1. stop current Hostbox containers
2. update the checkout to current `main`
3. remove the SQLite database and runtime artifacts
4. rebuild everything from source
5. start a clean Hostbox on the same VM and IP

### Replace an older manual install with a clean one

If your VM already has an old `/opt/hostbox` checkout and you just want to start over:

```bash
cd /opt/hostbox 2>/dev/null && sudo docker compose down --remove-orphans || true
sudo rm -rf /opt/hostbox
git clone https://github.com/VatsalP117/hostbox.git /opt/hostbox
cd /opt/hostbox
sudo mkdir -p /opt/hostbox/{data/backups,deployments,logs,cache,tmp}
cp .env.production.example .env
# edit .env
docker compose up -d --build --remove-orphans
```

### Stop

```bash
docker compose down
```

### Backups

Backups are written under `/opt/hostbox/data/backups` on the VM.

```bash
docker compose exec -T hostbox hostbox-cli admin backup
```

## Troubleshooting

### Hostbox container keeps restarting

This is usually a bad or incomplete `.env`.

```bash
docker compose logs --tail=100 hostbox
```

Common startup errors:

- `JWT_SECRET must be at least 32 characters`
- `ENCRYPTION_KEY is required`
- `PLATFORM_DOMAIN is required`

### Health checks

```bash
docker compose exec -T hostbox wget --no-verbose --tries=1 -O- http://127.0.0.1:8080/api/v1/health
docker compose exec -T hostbox wget --no-verbose --tries=1 -O- http://127.0.0.1:8080/api/v1/setup/status
docker compose exec -T caddy wget --no-verbose --tries=1 -O- http://127.0.0.1:2019/config/
```

### Docker build access is disabled

If Hostbox logs `docker client not available` or `permission denied` for `/var/run/docker.sock`, check the socket group:

```bash
stat -c '%g %n' /var/run/docker.sock
grep '^DOCKER_GID=' .env
docker compose exec -T hostbox sh -lc 'id && ls -ln /var/run/docker.sock'
```

On Linux, `DOCKER_GID` in `.env` should match the socket's group ID on the host. After changing it:

```bash
docker compose up -d
```

### Hostbox container restarts with `unable to open database file`

If Hostbox starts and immediately restarts with errors like:

```text
failed to open database
apply pragmas: exec "PRAGMA journal_mode = WAL": unable to open database file
```

the runtime directories on the host are usually owned by `root` instead of the Hostbox container user. Fix ownership and restart:

```bash
sudo chown -R 1000:1000 /opt/hostbox/data /opt/hostbox/deployments /opt/hostbox/logs /opt/hostbox/cache /opt/hostbox/tmp
cd /opt/hostbox
docker compose up -d --remove-orphans
```

### Installer fails with registry `denied`

Current Hostbox installs do **not** need `docker compose pull` from GHCR anymore. The installer now clones the source and builds locally with:

```bash
docker compose up -d --build --remove-orphans
```

If you hit a GHCR pull error, you are almost certainly running an older installer or an older local checkout. Update the source and rerun the current installer or:

```bash
cd /opt/hostbox
git fetch --depth 1 origin main
git checkout -B main origin/main
bash scripts/update.sh --fresh
```

### Builds fail with exit code 137

If install or build logs end with `command exited with code 137`, Docker killed the build container. Increase the build memory limit in `.env`, then restart Hostbox:

```env
BUILD_MEMORY_MB=1024
```

```bash
docker compose up -d
```

### Build container bind mounts fail with `mount path must be absolute`

Hostbox passes clone, artifact, and log directories to Docker as bind-mount sources. Docker requires those paths to be absolute.

- When you run Hostbox directly on the VM, relative paths in `.env` are resolved from the current working directory.
- When you run Hostbox inside Docker and point it at the host Docker socket, those paths must be valid on the Docker daemon host, not just inside the Hostbox container.

For Docker installs, mount the host directories into the Hostbox and Caddy containers at the same absolute paths you configure in `.env`.

### Build fails with `EACCES` on `/app/src/_tmp_*`

If a Node package manager fails with an error like:

```text
EACCES: permission denied, open '/app/src/_tmp_...'
```

you are likely running an older Hostbox build-container configuration. Current `main` keeps the hardened container model but restores the narrow capabilities needed for package managers to write temp files into the checked-out source directory.

Update Hostbox on the VM:

```bash
cd /opt/hostbox
bash scripts/update.sh --fresh
```

### TLS or certificate issues

```bash
docker compose logs --tail=200 caddy
dig +short example.com
dig +short hostbox.example.com
```

If you are using Cloudflare with wildcard DNS, make sure the wildcard record is **DNS only**, not proxied.

### Dashboard host is outside the root zone

If `DASHBOARD_DOMAIN` is not under `PLATFORM_DOMAIN`, the wildcard record will not cover it. Add a separate `A` or `AAAA` record for that host.
