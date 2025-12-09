# LandR - Implementation Checklist

## ✅ Completed
- [x] Backend Core (Go + gRPC + Postgres)
- [x] Frontend (Expo + React Native)
- [x] Authentication (Google OAuth)
- [x] Add Material + AI Flashcard Generation
- [x] Spaced Repetition System
- [x] Material Summary Feature

---

## 🚀 Improvements (Priority Order)

### 1. Parallel AI Generation ✅
- [x] Generate Q&A and Summary in parallel using goroutines
- [x] Update `AddMaterial` in core layer
- [x] Save summary immediately on material creation

### 2. "Wrong" Button for Spaced Repetition ✅
- [x] Add `FailReview` RPC - resets card to earlier stage
- [x] Update `MaterialDetailScreen` UI with "Wrong" button
- [x] Adjust stage logic (wrong = go back 1 stage, min 0)

### 3. Delete Material ✅
- [x] Add `DeleteMaterial` RPC
- [x] Implement soft delete in DB (`deleted_at` column)
- [x] Add delete button to `HomeScreen` list items
- [x] Filter out deleted items in queries

### 4. Edit Flashcard ✅
- [x] Add `UpdateFlashcard` RPC
- [x] Add edit UI in MaterialDetailScreen

### 5. Search & Filter
- [ ] Add search by title on HomeScreen
- [ ] Filter by tags

### 6. DB Transactions
- [ ] Wrap AddMaterial operations in transaction

### 7. Statistics Dashboard
- [ ] Track reviews completed
- [ ] Show learning streak
- [ ] Add stats screen

---

## 🔧 Technical Debt
- [ ] Add unit tests for core logic
- [ ] Error retry logic for AI calls
- [ ] Rate limiting on backend
- [ ] Remove hardcoded DB credentials

