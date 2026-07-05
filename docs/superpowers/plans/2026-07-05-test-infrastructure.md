# 实施计划：Infrastructure 测试全覆盖

参考：`docs/specs/2026-07-05-test-infrastructure.md`

## Task 1: internal/notify 补充测试

### `internal/notify/inapp_test.go`

```go
package notify

import (
    "testing"
)

func TestInAppNotifier_Send(t *testing.T) {
    n := NewInAppNotifier()
    err := n.Send("Test Title", "Test Body")
    if err != nil { t.Fatal(err) }
    
    msgs := n.Messages()
    if len(msgs) != 1 {
        t.Fatal("expected 1 message")
    }
    if msgs[0].Title != "Test Title" || msgs[0].Body != "Test Body" {
        t.Error("message content mismatch")
    }
}

func TestInAppNotifier_MultipleMessages(t *testing.T) {
    n := NewInAppNotifier()
    n.Send("A", "1")
    n.Send("B", "2")
    if len(n.Messages()) != 2 {
        t.Error("expected 2 messages")
    }
}
```

### `internal/notify/telegram_test.go`

```go
package notify

import (
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestTelegramNotifier_Send_Success(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Method != "POST" {
            t.Errorf("expected POST, got %s", r.Method)
        }
        w.WriteHeader(http.StatusOK)
    }))
    defer server.Close()
    
    n := &TelegramNotifier{
        apiKey:  "test-key",
        chatID:  "test-chat",
        apiBase: server.URL,
        client:  server.Client(),
    }
    err := n.Send("title", "body")
    if err != nil {
        t.Error("expected no error, got:", err)
    }
}

func TestTelegramNotifier_Send_HTTPError(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusForbidden)
    }))
    defer server.Close()
    
    n := &TelegramNotifier{
        apiKey:  "test-key",
        chatID:  "test-chat",
        apiBase: server.URL,
        client:  server.Client(),
    }
    err := n.Send("title", "body")
    if err == nil {
        t.Error("expected error for HTTP 403")
    }
}
```

## Task 2: internal/schedule 补充测试

### `internal/schedule/repo_test.go`

```go
package schedule

import (
    "testing"
    "quantflow/internal/storage"
)

func TestScheduleRepo_SaveListDelete(t *testing.T) {
    db, err := storage.NewSQLiteDB(":memory:")
    if err != nil { t.Fatal(err) }
    defer db.Close()
    
    r := NewRepo(db)
    
    s := &Schedule{
        ID:         "s1",
        Type:       "cron",
        Expression: "0 * * * *",
        Action:     "notify",
        Enabled:    true,
    }
    if err := r.Save(s); err != nil {
        t.Fatal(err)
    }
    
    list, err := r.List()
    if err != nil { t.Fatal(err) }
    if len(list) != 1 || list[0].ID != "s1" {
        t.Error("expected [s1], got", list)
    }
    
    if err := r.Delete("s1"); err != nil {
        t.Fatal(err)
    }
    list, _ = r.List()
    if len(list) != 0 {
        t.Error("expected empty list after delete")
    }
}

func TestScheduleRepo_GetByID(t *testing.T) {
    db, _ := storage.NewSQLiteDB(":memory:")
    defer db.Close()
    r := NewRepo(db)
    
    r.Save(&Schedule{ID: "s1", Type: "once", Expression: "now"})
    s, err := r.GetByID("s1")
    if err != nil { t.Fatal(err) }
    if s.ID != "s1" {
        t.Error("expected s1")
    }
    
    _, err = r.GetByID("nonexistent")
    if err == nil {
        t.Error("expected error for nonexistent")
    }
}

func TestScheduleRepo_Update(t *testing.T) {
    db, _ := storage.NewSQLiteDB(":memory:")
    defer db.Close()
    r := NewRepo(db)
    
    r.Save(&Schedule{ID: "s1", Enabled: true})
    r.Save(&Schedule{ID: "s1", Enabled: false})
    s, _ := r.GetByID("s1")
    if s.Enabled {
        t.Error("expected disabled after update")
    }
}
```

## Task 3: internal/ws/topics 测试

### `internal/ws/topics/depth_test.go`

```go
package topics

import (
    "encoding/json"
    "testing"
)

func TestDepthTopic_Marshal(t *testing.T) {
    topic := NewDepthTopic("AAPL")
    data := DepthData{
        Symbol: "AAPL",
        Bids:   []PriceLevel{{Price: 150.0, Size: 100}},
        Asks:   []PriceLevel{{Price: 151.0, Size: 200}},
    }
    msg, err := topic.Marshal(data)
    if err != nil { t.Fatal(err) }
    
    var decoded map[string]any
    if err := json.Unmarshal(msg, &decoded); err != nil {
        t.Fatal("invalid JSON:", err)
    }
    if decoded["symbol"] != "AAPL" {
        t.Error("symbol missing in output")
    }
}

func TestDepthTopic_Empty(t *testing.T) {
    topic := NewDepthTopic("AAPL")
    data := DepthData{Symbol: "AAPL"}
    msg, err := topic.Marshal(data)
    if err != nil { t.Fatal(err) }
    if len(msg) == 0 {
        t.Error("expected non-empty JSON")
    }
}
```

### `internal/ws/topics/kline_test.go` + `tick_test.go`

同样模式：序列化 → JSON 验证 → 字段检查。

## 验证

```bash
go test ./internal/notify/... -v -count=1
go test ./internal/schedule/... -v -count=1
go test ./internal/ws/... -v -count=1
```
