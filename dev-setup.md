# Zedu Backend — Local Development Setup

This guide explains how to run the Zedu backend and its dependencies locally using Docker Compose. Following it from a fresh clone should get you a running backend with no extra steps.

## Prerequisites

- Docker ≥ 24.x
- Docker Compose ≥ 2.x
- `make` (optional — every command below has a plain Docker Compose equivalent)

## 1. Create your environment file

The app reads its configuration from `app.env`, which is **not** committed. Copy the sample and use it as-is for local dev (the defaults line up with the dev Compose file):

```bash
cp app-sample.env app.env
```

You can edit values later, but the defaults are enough to boot everything locally.

## 2. Start everything

```bash
make start-dev
```

or, without make:

```bash
docker compose -f docker-compose.dev.yml up --build
```

This builds the backend image and starts it alongside Postgres, Redis, MongoDB, MinIO, Elasticsearch, RabbitMQ and Centrifugo. The first run pulls images and can take a few minutes.

When it's up:

- Backend / API: http://localhost:8019
- Swagger docs: http://localhost:8019/api/docs/index.html

Wait until the `telex-backend` container logs that it is listening before opening those URLs — the other services must finish starting first.

## 3. Stop and clean up

```bash
make dev-clean
```

or:

```bash
docker compose -f docker-compose.dev.yml down -v
```

`-v` also removes the data volumes, giving you a clean slate on the next start.

## Configuration files

- `app.env` — your local environment (created in step 1; never commit it).
- `.air.toml` — hot-reload configuration used by the backend container. It **is** committed; if it is missing after a clone, see Troubleshooting.
- `config.dev.json` — Centrifugo (realtime) configuration used by the dev stack.

## Inspecting logs

```bash
docker exec -it telex-backend sh
cat logs/app.log
```

or follow the live container output:

```bash
docker compose -f docker-compose.dev.yml logs -f backend
```

## Troubleshooting

**`Makefile: app.env: No such file or directory`**
You skipped step 1. Run `cp app-sample.env app.env` and try again.

**`make: *** No rule to make target 'start-dev'`**
The target is `start-dev` (hyphen), not `start:dev`. Check available targets with:
```bash
grep -E '^[a-zA-Z][a-zA-Z0-9_.-]*:' Makefile
```

**Backend build fails on `go install github.com/air-verse/air@latest`**
The build needs `git` in the image and a pinned `air` version. `Dockerfile.dev` should contain:
```dockerfile
RUN apk add --no-cache git
RUN go install github.com/air-verse/air@v1.61.5
```

**`telex-backend | open .air.toml: no such file or directory` (container restarts in a loop)**
`.air.toml` is missing from the container. It must be committed to the repo. Note that `.gitignore` ignores `*.toml`, which excludes it by accident — the ignore rule must make an exception:
```gitignore
*.toml
!.air.toml
```
Ensure `.air.toml` exists at the repo root and is tracked by git. A minimal working file:
```toml
root = "."
tmp_dir = "tmp"

[build]
  cmd = "go build -o ./tmp/main ."
  bin = "./tmp/main"
  include_ext = ["go"]
```

**`ERR_CONNECTION_REFUSED` when opening the Swagger URL**
The backend isn't up yet or is crash-looping. Check its logs (see Inspecting logs) and resolve any errors above before retrying.