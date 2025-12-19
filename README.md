# LandR – Learn and Revise SaaS

## Overview
LandR is a full‑stack **learning & revision** application that lets users add learning material (text or links) and automatically generates flashcards using AI.  Flashcards are organized by **materials**, and users can review them using a spaced‑repetition system.

- **Backend** – Go (clean architecture) with PostgreSQL, gRPC‑Web API.
- **Frontend** – React Native (Expo) + TypeScript, uses `nice-grpc-web` client.
- **AI** – Groq API for flashcard generation.
- **Database** – PostgreSQL with tables for users, materials, flashcards, tags, and material‑tags.

## Project Structure
```
LandR/
├─ backend/               # Go server
│   ├─ cmd/server/        # entry point
│   ├─ internal/          # core, service, store layers
│   ├─ db/migrations/     # SQL migrations
│   └─ proto/learning/    # protobuf definitions
├─ frontend/              # Expo React Native app
│   ├─ src/
│   │   ├─ screens/       # Home, MaterialDetail, Review, AddMaterial
│   │   ├─ services/      # gRPC client & definitions
│   │   ├─ navigation/    # React Navigation stack
│   │   └─ store/         # auth store (zustand)
│   └─ proto/             # generated TS from protobuf (learning, auth)
├─ Makefile               # helper commands (proto, migrate, build)
└─ README.md              # **THIS FILE**
```

## Prerequisites
- **Node.js** (>=18) and **npm**
- **Go** (>=1.22)
- **PostgreSQL** (running locally or via Docker)
- **Docker** (optional, for quick DB setup)
- **Groq API key** (set in `.env`)

## Setup & Installation
### 1. Clone the repository
```bash
git clone <repo-url>
cd LandR
```
### 2. Backend
```bash
# Install Go dependencies
cd backend
go mod tidy

# Create a .env file (copy from .env.example)
cp .env.example .env
# Edit .env with your DB credentials and GROQ_API_KEY

# Run migrations
make migrate-up

# Build the server (produces bin/server.exe on Windows)
make build-backend
```
### 3. Frontend
```bash
cd ../frontend
npm install

# Generate TypeScript protobuf files (if you changed .proto)
make proto   # runs protoc with ts‑proto plugin

# Start the Expo dev server
npm run dev   # or: npx expo start
```
The app will launch in the Expo Go app on your device or in a simulator.

## Development Commands (Makefile)
| Target | Description |
|--------|-------------|
| `make proto` | Regenerate Go and TS protobuf files |
| `make migrate-up` | Apply DB migrations |
| `make migrate-down` | Roll back last migration |
| `make build-backend` | Build the Go server |
| `make run-backend` | Run the server (`bin/server.exe`) |
| `make test` | Run Go unit tests |

## API Overview (gRPC‑Web)
The service is defined in `backend/proto/learning/learning.proto`.
### Service: `LearningService`
- **AddMaterial** – `AddMaterialRequest → AddMaterialResponse`
- **GetDueMaterials** – `google.protobuf.Empty → GetDueMaterialsResponse`
- **GetDueFlashcards** – `GetDueFlashcardsRequest (material_id) → FlashcardList`
- **CompleteReview** – `CompleteReviewRequest → google.protobuf.Empty`

### Service: `AuthService`
- **Login** – `LoginRequest → LoginResponse`

### Service: `FeedService` (Daily AI Feed)
- **GetFeedPreferences** – Get user's feed preferences
- **UpdateFeedPreferences** – Update interest prompt and enable/disable
- **GetDailyFeed** – Get articles for a specific date
- **GetFeedCalendarStatus** – Get calendar days with articles

### REST API: Manual Feed Trigger
Manually trigger daily feed generation for a user by email (requires API key):

```bash
# Set your API key and user email
curl -X POST "http://localhost:8080/api/feed/refresh?email=user@example.com" \
  -H "X-API-Key: your-secret-api-key"
```

**Response:**
```json
{"status": "success", "message": "Daily feed refreshed successfully"}
```

> **Note**: Set `FEED_API_KEY` in your `.env` file. This same key must be passed in the `X-API-Key` header.

The generated TypeScript client lives in `frontend/proto/...` and is used via `nice-grpc-web`.

## Running the Application
1. Start the backend server:
```bash
cd backend && bin/server.exe
```
2. In another terminal, start the Expo dev server (frontend):
```bash
cd frontend && npm run dev
```
3. Open the Expo app on your phone or emulator.  Sign in (Google) → Add material → Review flashcards.

## Testing
- **Backend** – `go test ./...` inside the `backend` folder.
- **Frontend** – Use Jest/React‑Native testing library (not set up yet, but you can add tests under `frontend/src/__tests__`).

## Architecture Highlights
- **Clean Architecture** – `core` contains business logic, `service` implements gRPC handlers, `store` abstracts DB access.
- **AI Integration** – `internal/ai/client.go` calls Groq to generate flashcards, title, and tags.
- **Spaced Repetition** – Simple SRS logic in `core.CompleteReview` (stage bump & next‑review date).
- **Material‑Based UI** – Home screen lists materials, clicking a material shows its flashcards, clicking a flashcard opens the review screen.

## Troubleshooting
- **"Flashcard not found"** – Ensure the flashcard ID exists in the cached query. The app now looks through all `dueFlashcards` queries for any material.
- **Database connection errors** – Verify `DATABASE_URL` in `.env` matches your local PostgreSQL instance.
- **Proto changes not reflected** – Run `make proto` again and restart both backend and frontend.

## License
MIT License – see `LICENSE` file.

---
*Happy learning!*
