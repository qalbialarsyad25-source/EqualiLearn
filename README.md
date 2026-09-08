# EquiliLearn Backend

> **Comprehensive backend API** for EquiliLearn — featuring real-time Speech-to-Text (STT), Text-to-Speech (TTS), AI document summarization (Gemini), collaborative group chat (WebSocket), and JWT-based authentication with Google OAuth2.

---

## Tech Stack

| Layer | Technology |
|---|---|
| Language | Go 1.26 |
| Framework | Gin |
| Database | PostgreSQL 16 |
| Cache / PubSub | Redis 7 |
| AI / Speech | Deepgram (STT & TTS), Google Gemini |
| Auth | JWT + Google OAuth2 |
| WebSocket | gorilla/websocket |
| Container | Docker + Docker Compose |

---

## Prerequisites

- [Docker Desktop](https://www.docker.com/products/docker-desktop/) (v24+)
- [Docker Compose](https://docs.docker.com/compose/) (included with Docker Desktop)
- API keys for **Deepgram** and **Google Gemini**
- Google OAuth2 credentials (Client ID & Secret)

---

## Getting Started

### 1. Clone the Repository

```bash
git clone https://github.com/your-org/EquiliLearn.git
cd EquiliLearn
```

### 2. Configure Environment Variables

Copy the example file and fill in your values:

```bash
cp .env.example .env
```

Edit `.env`:

```env
# ── Database ──────────────────────────────────────────
DB_HOST=localhost          # Use "postgres" when running inside Docker
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_db_password
DB_NAME=EquilLearn

# ── App ───────────────────────────────────────────────
APP_PORT=8080
JWT_SECRET_KEY=your_jwt_secret_key
JWT_EXPIRED_TIME=24        # hours

# ── CORS / Frontend ───────────────────────────────────
FRONTEND_URL=http://localhost:3000,https://your-app.vercel.app
ALLOWED_ORIGINS=http://localhost:3000,https://*.vercel.app

# ── Google OAuth2 ─────────────────────────────────────
GOOGLE_CLIENT_ID=your_google_client_id
GOOGLE_CLIENT_SECRET=your_google_client_secret
GOOGLE_REDIRECT_URL=http://localhost:8080/api/v1/auth/google/callback

# ── Speech (Deepgram) ─────────────────────────────────
STT_PROVIDER=deepgram
TTS_PROVIDER=deepgram
DEEPGRAM_API_KEY=your_deepgram_api_key

# ── AI (Gemini) ───────────────────────────────────────
GEMINI_API_KEY=your_gemini_api_key
GEMINI_MODEL=gemini-2.0-flash
```

> **Note:** When running via Docker Compose, `DB_HOST` and `REDIS_HOST` are automatically overridden to the container service names (`postgres` and `redis`). You do not need to change them manually.

---

## Running with Docker

### Start all services (PostgreSQL + Redis + Backend)

```bash
docker compose up -d
```

This will:
1. Pull `postgres:16-alpine` and `redis:7-alpine` images
2. Build the Go backend image from the local `dockerfile`
3. Start all three containers with health checks
4. Expose the API at **`http://localhost:8080`**

### Check running containers

```bash
docker compose ps
```

### View backend logs

```bash
docker compose logs -f backend
```

### View all service logs

```bash
docker compose logs -f
```

### Stop all services

```bash
docker compose down
```

### Stop and remove volumes (deletes database data)

```bash
docker compose down -v
```

---

## Rebuilding After Code Changes

```bash
docker compose up -d --build backend
```

---

## Container Architecture

```
+--------------------------------------------------+
|               Docker Compose Network             |
|                                                  |
|  +--------------+    +----------------------+    |
|  |   postgres   |    |        redis         |    |
|  |   :5432      |    |   :6379 (internal)   |    |
|  +------+-------+    +----------+-----------+    |
|         |                       |                |
|         +----------+------------+                |
|                    |                             |
|           +--------v---------+                   |
|           |     backend      |                   |
|           |    :8080         |                   |
|           +------------------+                   |
+---------------------+----------------------------+
                      | exposed to host
                 localhost:8080
```

> Redis port 6379 is intentionally **not exposed to the host** — it is only accessible within the Docker network.

---

## API Overview

Base URL: `http://localhost:8080/api/v1`

### Authentication

| Method | Endpoint | Description |
|---|---|---|
| POST | `/auth/register` | Register new user |
| POST | `/auth/login` | Login with email & password |
| GET | `/auth/google/login` | Initiate Google OAuth2 login |
| GET | `/auth/google/callback` | Google OAuth2 callback |
| POST | `/auth/forgot-password` | Request password reset |
| POST | `/auth/reset-password` | Confirm password reset |

### Speech (STT / TTS)

| Method | Endpoint | Auth | Description |
|---|---|---|---|
| GET | `/ws/speech-to-text` | Optional | **WebSocket** real-time speech recognition |
| GET | `/speech/history` | Required | Get transcription history |
| DELETE | `/speech/history/:id` | Required | Delete a transcription |
| POST | `/speech/synthesize` | — | Text-to-Speech synthesis |
| GET | `/speech/voices` | — | List available TTS voices |

### AI Documents

| Method | Endpoint | Auth | Description |
|---|---|---|---|
| POST | `/documents/summarize` | — | Summarize uploaded document (Gemini) |
| POST | `/documents/summarize-text` | — | Summarize plain text (Gemini) |
| GET | `/documents/history` | Required | Get document summary history |
| GET | `/documents/:id` | — | Get a specific summary |
| DELETE | `/documents/:id` | Required | Delete a summary |

### Group Chat

| Method | Endpoint | Auth | Description |
|---|---|---|---|
| GET | `/ws/chat` | Required | **WebSocket** real-time group chat |
| POST | `/groups` | Required | Create a group |
| GET | `/groups` | Required | List user groups |
| GET | `/groups/:id` | Required | Get group details |
| POST | `/groups/:id/members` | Required | Add member to group |
| DELETE | `/groups/:id/members/:userId` | Required | Remove member |
| GET | `/groups/:id/messages` | Required | Get message history |

Full OpenAPI spec: `openapi.json` — import into [Apidog](https://apidog.com), Postman, or Swagger UI.

---

## WebSocket Endpoints

### Speech-to-Text

```
ws://localhost:8080/api/v1/ws/speech-to-text?lang=id-ID&token=<JWT>
```

**Send** raw PCM audio as binary frames (16kHz, 16-bit mono) or JSON control messages:

```json
{ "type": "audio_chunk", "payload": "<base64-encoded-audio>" }
{ "type": "config",      "payload": { "language_code": "en-US" } }
```

**Receive** JSON events:

```json
{ "type": "ready",      "payload": { "session_id": "...", "language": "id-ID", "sample_rate": 16000 } }
{ "type": "transcript", "payload": { "text": "...", "is_final": true, "confidence": 0.97 } }
{ "type": "finished",   "payload": { "session_id": "...", "duration_ms": 5000, "text": "..." } }
{ "type": "error",      "payload": { "message": "..." } }
```

> Authentication is optional for STT. Without a JWT token, the session transcript will not be saved to the database.

### Group Chat

```
ws://localhost:8080/api/v1/ws/chat?token=<JWT>&group_id=<UUID>
```

---

## Environment Variables Reference

| Variable | Required | Description |
|---|---|---|
| `DB_HOST` | Yes | PostgreSQL host (`postgres` in Docker) |
| `DB_PORT` | Yes | PostgreSQL port (default `5432`) |
| `DB_USER` | Yes | PostgreSQL username |
| `DB_PASSWORD` | Yes | PostgreSQL password |
| `DB_NAME` | Yes | PostgreSQL database name |
| `APP_PORT` | Yes | Backend HTTP port (default `8080`) |
| `JWT_SECRET_KEY` | Yes | Secret for signing JWT tokens |
| `JWT_EXPIRED_TIME` | Yes | Token expiry in hours |
| `FRONTEND_URL` | Yes | Allowed frontend origins (comma-separated) |
| `ALLOWED_ORIGINS` | Yes | CORS allowed origins (comma-separated) |
| `GOOGLE_CLIENT_ID` | Yes | Google OAuth2 Client ID |
| `GOOGLE_CLIENT_SECRET` | Yes | Google OAuth2 Client Secret |
| `GOOGLE_REDIRECT_URL` | Yes | Google OAuth2 redirect callback URL |
| `STT_PROVIDER` | Yes | Speech-to-Text provider (`deepgram`) |
| `TTS_PROVIDER` | Yes | Text-to-Speech provider (`deepgram`) |
| `DEEPGRAM_API_KEY` | Yes | Deepgram API key |
| `GEMINI_API_KEY` | Yes | Google Gemini API key |
| `GEMINI_MODEL` | Yes | Gemini model name (e.g. `gemini-2.0-flash`) |

---

## Development Without Docker

```bash
# Install Go dependencies
go mod download

# Make sure PostgreSQL and Redis are running locally, then:
go run main/main.go
```

Make sure `DB_HOST=localhost` in your `.env`.

---

## Troubleshooting

### Backend cannot connect to database

The backend waits for Postgres to be healthy before starting. If it still fails:

```bash
docker compose logs postgres
docker compose restart backend
```

### Port 5432 already in use

A local PostgreSQL instance is likely running. Stop it first:

```bash
# macOS / Linux
sudo service postgresql stop

# Windows (PowerShell as Administrator)
net stop postgresql-x64-16
```

### Rebuilding after `.env` changes

Docker Compose reads `.env` at startup. After changing it:

```bash
docker compose down && docker compose up -d
```

---

## License

MIT
