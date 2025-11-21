# Implementation Checklist: Learn and Revise SaaS

## Phase 1: Backend Core (Go + gRPC + Postgres)

### 1.1. Project Initialization
- [ ] Initialize Go module (`go mod init github.com/youruser/landr`)
- [ ] Set up project structure (`cmd`, `internal`, `pkg`, `proto`)
- [ ] Configure `Makefile` for common tasks (build, run, proto-gen)

### 1.2. Protocol Buffers (gRPC)
- [ ] Define `auth.proto` (Login)
- [ ] Define `learning.proto` (AddMaterial, GetDueFlashcards, CompleteReview)
- [ ] Generate Go code from protos (`protoc`)

### 1.3. Database Setup
- [ ] Set up PostgreSQL locally or via Docker
- [ ] Define schema migration files (`users`, `materials`, `flashcards`)
- [ ] Implement DB connection and migration runner (using `golang-migrate` or `pgx`)
- [ ] Create `Store` interface and implementation for CRUD operations

### 1.4. Authentication
- [ ] Implement `AuthService.Login`
- [ ] Verify Google ID Token
- [ ] Create/Update User in DB
- [ ] Issue JWT Session Token

### 1.5. Core Logic: Learning Materials
- [ ] Implement `LearningService.AddMaterial`
- [ ] **Scraper Module**: Implement `goquery` logic to fetch and extract text from URLs
- [ ] **AI Module**: Implement Groq API client
    - [ ] Prompt engineering for Q/A generation
    - [ ] JSON parsing of Groq response
- [ ] Save Material and Flashcards to DB

### 1.6. Core Logic: Spaced Repetition & Review
- [ ] Implement `LearningService.GetDueFlashcards` (Query `next_review_at <= NOW()`)
- [ ] Implement `LearningService.CompleteReview`
    - [ ] Logic to calculate next interval based on fixed schedule: `[1, 3, 7, 15, 30]`
    - [ ] Update `stage` and `next_review_at`

## Phase 2: Frontend (Expo + React Native)

### 2.1. Project Initialization
- [ ] Initialize Expo project (TypeScript)
- [ ] Setup directory structure (`src/components`, `src/screens`, `src/services`)
- [ ] Install dependencies (`@react-navigation/native`, `protobufjs`, etc.)

### 2.2. gRPC Client Setup
- [ ] Generate TypeScript gRPC client from protos
- [ ] Configure API client with base URL and Auth interceptor

### 2.3. Authentication UI
- [ ] Implement "Sign in with Google" button
- [ ] Handle Google OAuth flow on device
- [ ] Call `AuthService.Login` and store JWT

### 2.4. Main UI: Home & Add Material
- [ ] **Home Screen**: Display "Due Today" count and list of cards
- [ ] **Add Material Screen**:
    - [ ] Text Input for raw notes
    - [ ] URL Input for links
    - [ ] "Add" button triggering `AddMaterial` API

### 2.5. Review Interface (Flashcards)
- [ ] **Flashcard Screen**:
    - [ ] Show Question
    - [ ] "Reveal Answer" button
    - [ ] Show Answer
    - [ ] "Done" button triggering `CompleteReview` API
- [ ] Implement simple animation for card flip (optional but nice)

### 2.6. Notifications
- [ ] Configure `expo-notifications`
- [ ] Request permissions
- [ ] Schedule local notifications based on due dates

## Phase 3: Polish & Deployment
- [ ] **Testing**: Verify end-to-end flow (Add -> AI Gen -> Review -> Next Schedule)
- [ ] **Docker**: Create `Dockerfile` for Backend
- [ ] **Readme**: Document how to run the stack locally
