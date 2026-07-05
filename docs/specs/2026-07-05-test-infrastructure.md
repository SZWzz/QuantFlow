# Test Coverage: Infrastructure Packages (notify, schedule, ws/topics)

## Motivation

三个基础设施包测试覆盖不足：

| 包 | 源文件 | 测试文件 | 现状 |
|---|:---:|:---:|---|
| `internal/notify` | 4 | 1 | `manager_test.go` 已有，缺 `inapp.go`、`telegram.go` 测试 |
| `internal/schedule` | 3 | 1 | `scheduler_test.go` 已有，缺 `repo.go` 持久化测试 |
| `internal/ws/topics` | 3 | **0** | 完全零测试 — 行情 topic 分发逻辑 |

## Design

### 1. internal/notify — 通知引擎

**`types.go`** — 通知类型、级别定义，不需要测试。

**`manager.go`** — 已有测试 `manager_test.go`，补充：
- 多 `Notifier` 注册和广播
- 单个 notifier 失败不影响其他 notifier
- `Send` 超时不阻塞

**`inapp.go`** — 应用内通知存储：
```go
func TestInAppNotifier_Send(t *testing.T) {
    n := NewInAppNotifier()
    err := n.Send("title", "body")
    if err != nil { t.Fatal(err) }
    msgs := n.Messages()
    if len(msgs) != 1 || msgs[0].Title != "title" {
        t.Error("unexpected messages")
    }
}
```

**`telegram.go`** — Telegram bot 通知：
- 测试 HTTP 请求构造（mock HTTP server）
- 测试 API Key 缺失时的优雅降级
- 不发送真实 Telegram 消息

```go
func TestTelegramNotifier_Send(t *testing.T) {
    mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
    }))
    defer mockServer.Close()
    
    n := NewTelegramNotifier("fake-key", "fake-chat")
    n.apiBase = mockServer.URL // 注入 mock URL
    err := n.Send("test", "body")
    if err != nil { t.Fatal(err) }
}
```

### 2. internal/schedule — 定时调度

**`types.go`** — 调度任务定义，不需要测试。

**`scheduler.go`** — 已有测试 `scheduler_test.go`，补充：
- 多个任务同时运行
- 任务 panic 不影响其他任务（recovery）
- `Stop` 后不执行

**`repo.go`** — 调度持久化（SQLite）：
```go
func TestScheduleRepo_SaveListDelete(t *testing.T) {
    db, err := storage.NewSQLiteDB(":memory:")
    if err != nil { t.Fatal(err) }
    r := NewRepo(db)
    
    s := &Schedule{ID: "s1", Type: "cron", Expression: "0 * * * *"}
    err = r.Save(s)
    if err != nil { t.Fatal(err) }
    
    list, err := r.List()
    if err != nil { t.Fatal(err) }
    if len(list) != 1 { t.Fatal("expected 1 schedule") }
    
    err = r.Delete("s1")
    if err != nil { t.Fatal(err) }
}
```

### 3. internal/ws/topics — WebSocket 行情主题

**`depth.go`**、**`kline.go`**、**`tick.go`** — 行情数据订阅主题：

```go
func TestDepthTopic_Serialize(t *testing.T) {
    topic := NewDepthTopic("AAPL")
    data := DepthData{Bids: []PriceLevel{{Price: 150, Size: 100}}}
    msg, err := topic.Marshal(data)
    if err != nil { t.Fatal(err) }
    if !bytes.Contains(msg, []byte("150")) {
        t.Error("expected price in serialized message")
    }
}
```

```go
func TestKlineTopic_SubscribeUnsubscribe(t *testing.T) {
    hub := ws.NewHub()
    topic := NewKlineTopic("AAPL", "1m")
    client := newMockClient()
    topic.Subscribe(hub, client)
    // 验证 client 注册到了 hub 的对应 topic
}
```

**测试总数**：notify ~10 个、schedule ~6 个、ws/topics ~9 个 = 约 25 个测试

## Acceptance Criteria

- [ ] `internal/notify/inapp.go` 测试通过
- [ ] `internal/notify/telegram.go` 测试通过（mock HTTP）
- [ ] `internal/schedule/repo.go` SQLite CRUD 测试通过
- [ ] `internal/ws/topics/*.go` 各有一个序列化/反序列化测试
- [ ] `go test ./internal/notify/... -count=1` 全部通过
- [ ] `go test ./internal/schedule/... -count=1` 全部通过
- [ ] `go test ./internal/ws/... -count=1` 全部通过

## Risks / Trade-offs

- `telegram.go` 测试需要 mock HTTP server。注意不要在测试中硬编码真实 Telegram API URL。
- `schedule/repo.go` 依赖 SQLite，用 `:memory:` 数据库避免文件泄漏。
- `ws/topics` 测试可能依赖 `ws.Hub` 的现有测试设施（`NewHub`、`Subscribe`），需确认 `hub_test.go` 导出这些功能。
