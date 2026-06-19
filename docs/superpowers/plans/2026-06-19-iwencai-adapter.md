# iwencai Adapter Implementation Plan

## Tasks

### Task 1: Create iwencai adapter
- **File**: `internal/market/adapters/iwencai.go` (NEW)
- **Content**: `IwencaiArticle` type, `IwencaiAdapter` struct, `Search()`, `Query()`, `dedupArticles()`, `clawHeaders()`

### Task 2: Create iwencai tests
- **File**: `internal/market/adapters/iwencai_test.go` (NEW)
- **Content**: Live search/query tests (skip if no key), dedup test, clawHeaders test

### Task 3: Wire into app.go
- **File**: `app.go` (MODIFY)
- **Content**: Add `iwencaiAdapter` field, create in `startup()`, add `SearchResearch()` exported method

### Task 4: Update CHANGELOG
- **File**: `CHANGELOG.md` (MODIFY)
