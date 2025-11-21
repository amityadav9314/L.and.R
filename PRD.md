# Product Requirements Document (PRD): Learn and Revise SaaS

## 1. Overview
"Learn and Revise" is a mobile-first SaaS application designed to help users retain information through active recall and spaced repetition. Users input learning materials (text or links), which are processed by AI (Groq) to generate Question/Answer flashcards. These cards are then presented to the user for review on a fixed spaced repetition schedule.

## 2. Technical Stack
*   **Backend Language:** Go (Golang) 1.25
*   **API Protocol:** gRPC (Protobuf)
*   **Database:** PostgreSQL
*   **AI Provider:** Groq API (LLM for summarization/Q&A generation)
*   **Web Scraper:** `goquery` (Go library for parsing HTML)
*   **Frontend Framework:** Expo with React Native
*   **Language (Frontend):** TypeScript
*   **Target Platform:** Android (Primary focus)
*   **Authentication:** Google OAuth
*   **Notifications:** Local Notifications (Expo Notifications)

## 3. Core Features & User Flow

### 3.1. Authentication
*   **User Action:** User opens the app and signs in using "Continue with Google".
*   **Backend:** Verify Google token, create/retrieve user in PostgreSQL.
*   **Constraint:** No email/password login. Google Auth only.

### 3.2. Add Learning Material
*   **User Action:** User clicks "+" to add material.
*   **Input Types:**
    1.  **Raw Text:** User types/pastes notes.
    2.  **Blog/Article Link:** User pastes a URL.
*   **Backend Processing:**
    *   If **Text**: Send directly to Groq.
    *   If **Link**: Fetch URL content -> Parse HTML with `goquery` to extract readable text -> Send text to Groq.
*   **AI Processing (Groq):**
    *   **Prompt:** Instruct Groq to analyze the content and generate a list of **Question & Answer** pairs suitable for flashcards.
    *   **Output Format:** Structured JSON (e.g., `[{ "question": "...", "answer": "..." }]`).
*   **Storage:** Save the original material and the generated flashcards to PostgreSQL.

### 3.3. Spaced Repetition System (SRS)
*   **Logic:** Fixed Schedule.
*   **Intervals:**
    *   1st Review: 1 day after creation.
    *   2nd Review: 3 days after creation.
    *   3rd Review: 7 days after creation.
    *   4th Review: 15 days after creation.
    *   5th Review: 30 days after creation.
*   **Mechanism:**
    *   When a flashcard is created, its `next_review_date` is set to `now() + 1 day`.
    *   When a user reviews a card, the system checks the current stage and advances the `next_review_date` to the next interval.

### 3.4. Review Interface
*   **User Action:** User sees a "Due Today" list on the home screen.
*   **Interaction:**
    1.  User taps a card.
    2.  Front shows the **Question**.
    3.  User taps "Reveal Answer".
    4.  Back shows the **Answer**.
    5.  User taps "Done" (or "Next").
*   **Backend:** Update the flashcard's `review_stage` and `next_review_date`.

### 3.5. Notifications
*   **Type:** Local Notifications (scheduled on the device).
*   **Trigger:** When the app syncs or a card is reviewed, schedule a local notification for the next review time (e.g., 9:00 AM on the due date).
*   **Content:** "You have X cards to revise today!"

## 4. Data Model (PostgreSQL)

### `users`
| Column | Type | Notes |
| :--- | :--- | :--- |
| `id` | UUID | PK |
| `email` | VARCHAR | Unique |
| `name` | VARCHAR | |
| `google_id` | VARCHAR | Unique |
| `created_at` | TIMESTAMP | |
| `updated_at` | TIMESTAMP | |

### `materials`
| Column | Type | Notes |
| :--- | :--- | :--- |
| `id` | UUID | PK |
| `user_id` | UUID | FK -> users.id |
| `type` | VARCHAR | 'TEXT' or 'LINK' |
| `content` | TEXT | Raw text or URL |
| `created_at` | TIMESTAMP | |
| `updated_at` | TIMESTAMP | |

### `flashcards`
| Column | Type | Notes |
| :--- | :--- | :--- |
| `id` | UUID | PK |
| `material_id` | UUID | FK -> materials.id |
| `question` | TEXT | |
| `answer` | TEXT | |
| `stage` | INT | Current step (0=New, 1=1d, 2=3d, etc.) |
| `next_review_at` | TIMESTAMP | When this card is due |
| `created_at` | TIMESTAMP | |
| `updated_at` | TIMESTAMP | |

## 5. API Definition (gRPC)

### Service: `AuthService`
*   `Login(LoginRequest) returns (LoginResponse)`
    *   Input: Google ID Token.
    *   Output: App Session Token / User Profile.

### Service: `LearningService`
*   `AddMaterial(AddMaterialRequest) returns (AddMaterialResponse)`
    *   Input: `type` (LINK/TEXT), `content`.
    *   Logic: Scrapes (if link), calls Groq, saves DB.
*   `GetDueFlashcards(Empty) returns (FlashcardList)`
    *   Logic: Returns cards where `next_review_at <= now()`.
*   `CompleteReview(CompleteReviewRequest) returns (Empty)`
    *   Input: `flashcard_id`.
    *   Logic: Increments `stage`, calculates new `next_review_at` based on fixed schedule.

## 6. Implementation Guidelines (Do's and Don'ts)

### DO:
*   **DO** use **gRPC** for all client-server communication. Define `.proto` files first.
*   **DO** use **Go 1.25** features if applicable.
*   **DO** separate the "Scraper", "AI", and "DB" logic into clean packages (e.g., `internal/scraper`, `internal/ai`, `internal/store`).
*   **DO** ensure the scraper handles basic errors (e.g., 404, timeouts) gracefully.
*   **DO** prompt Groq specifically to return valid JSON to avoid parsing errors.
*   **DO** implement the fixed schedule logic exactly as: `[1, 3, 7, 15, 30]` days offsets.

### DO NOT:
*   **DO NOT** use REST APIs.
*   **DO NOT** implement complex spaced repetition algorithms (like SM-2 or FSRS) yet. Stick to the fixed schedule.
*   **DO NOT** build a web frontend. Focus strictly on Android/Expo.
*   **DO NOT** add "Hard/Easy" buttons for review yet. Just a simple "Done" or "Next" to advance the schedule.
