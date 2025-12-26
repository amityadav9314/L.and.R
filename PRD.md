# Daily Feed Agent V2 – Product Requirements Document

## 1. Overview
The goal is to upgrade the daily feed generation from a simple **Search & Store** agent to a robust **Content-Aware Evaluation Loop**.  
The new agent will not just trust headlines; it will **read (scrape)** the article, **summarize** it, and **evaluate** its relevance against user-specific goals before showing it to the user.

---

## 2. Terminology
- **V1 Agent**: Current implementation (Search → Store list)
- **V2 Agent**: New implementation (Search → Scrape → Summarize → Evaluate → Store)
- **Evaluation Prompt**: User-specific intent used to score articles  
  Example: _"I want to learn advanced physics concepts"_

---

## 3. Workflow Logic
Runs as a background job (cron or on-demand).

### Step 1: Initialization
- Fetch Feed Status: Check `users.feed_enabled`. If `false`, exit.
- Fetch Preferences:
  - `users.interest_prompt` (Topics)
  - `users.feed_eval_prompt` (Evaluation Criteria – **NEW**)

---

### Step 2: Query Generation
- Call LLM to generate **ONE** high-quality search query.
- Input:
  ```
  User likes: AI, Python. Give me one search query.
  ```
- Output:
  ```
  latest python ai libraries 2025
  ```

---

### Step 3: Execute Search (Multi-Provider)
Iterate through all configured providers:

- **Tavily**
  - `Tavily.Search(query, news=true, days=7)`
  - Max 10 results
- **SerpApi**
  - `SerpApi.SearchNews(query, num=10)`
  - Max 10 results

**Aggregation**
- Combine all results into a single candidate URL list.

**Deduplication**
- Remove duplicates using normalized URLs.

---

### Step 4: Content Processing Loop (V2 Core Logic)
For each candidate URL (parallel execution recommended):

1. **Scrape**
   - `scraper.Scrape(url)`
   - Reuse existing scraper service.

2. **Summarize**
   - `ai.GenerateSummary(content)`
   - Reuse existing summary logic.

3. **Evaluate (LLM)**
   - Input: Article Summary + `feed_eval_prompt`
   - Prompt:
     ```
     Does this summary fulfill the user's wish? Rate 0.0 to 1.0.
     ```
   - Output: Float score.

---

### Step 5: Filtering & Storage
- **Filter**
  - Discard articles with `score < 0.6`.

- **Store**
  - Save accepted articles into `daily_articles` table.
  - Fields:
    - Title
    - URL
    - Generated Summary (Snippet)
    - Score
    - Provider (`tavily` / `google`)

- **Goal Check**
  - If stored articles ≥ 10 → STOP.
  - If < 10 and candidates exhausted:
    - Repeat from Step 2.
    - Max loops: **3**

---

## 4. Technical Requirements

### Database Changes
**users**
- Add column: `feed_eval_prompt TEXT NULL`
- Migration required.

---

### Component Architecture
- `FeedCore` needs access to:
  - Scraper
  - AIProvider (currently inside `LearningCore`)
- Refactor:
  - Inject `LearningCore` dependencies into `FeedCore`.

---

### Concurrency
- Use `errgroup` / goroutines.
- Process up to 10 URLs in parallel.
- Be mindful of LLM TPM limits during summarization + evaluation.

---

### Failure Handling
- Scrape failure → Skip + log warning.
- Summarization failure → Skip article.
- Loop limit → Hard cap at 3 iterations to avoid cost explosion.

---

## 5. UI Requirements
- **Settings Page**
  - Add input for `feed_eval_prompt`
  - Label ideas:
    - “Evaluation Criteria”
    - “What’s your learning goal?”

- **Frontend**
  - Send `feed_eval_prompt` via `UpdatePreferences` API.

---

## 6. Implementation Plan
1. DB Migration: Add new column.
2. Backend Refactor:
   - Create `internal/adk/feed_v2/` (or update `core/feed.go`)
3. Implement deterministic workflow in Go.
4. Use LLM only for:
   - Query generation
   - Summarization
   - Evaluation
5. Explicitly **avoid ReAct / autonomous agents**.
   - This is a **workflow**, not an agent.
6. Testing:
   - Mock Scraper
   - Mock AIProvider
