# 实施计划：Test Coverage — Zero-Coverage Packages

参考：`docs/specs/2026-07-05-test-coverage.md`

## Task 1: internal/ws — Hub 测试

**新建 `internal/ws/hub_test.go`**：

```go
package ws

import (
    "testing"
    "time"
)

func TestHub_SubscribeBroadcast(t *testing.T) {
    h := NewHub()
    defer h.Close()

    ch := make(chan []byte, 5)
    id := h.Subscribe(ch)
    defer h.Unsubscribe(id)

    h.Broadcast(Message{Type: "test", Data: "hello"})

    select {
    case msg := <-ch:
        if string(msg) != `{"type":"test","data":"hello"}` {
            t.Errorf("unexpected message: %s", msg)
        }
    case <-time.After(time.Second):
        t.Error("timeout")
    }
}

func TestHub_SubscribeUnsubscribe(t *testing.T) {
    h := NewHub()
    defer h.Close()

    ch := make(chan []byte, 5)
    id := h.Subscribe(ch)
    h.Unsubscribe(id)

    h.Broadcast(Message{Type: "test", Data: "data"})
    select {
    case <-ch:
        t.Error("should not receive after unsubscribe")
    case <-time.After(100 * time.Millisecond):
        // ok
    }
}

func TestHub_BroadcastBufferFull(t *testing.T) {
    h := NewHub()
    defer h.Close()

    ch := make(chan []byte, 0) // unbuffered
    h.Subscribe(ch)
    // should not block
    h.Broadcast(Message{Type: "test", Data: "data"})
}

func TestHub_Close(t *testing.T) {
    h := NewHub()
    h.Close()
    // should not panic on double close
    h.Close()
}
```

---

## Task 2: internal/auth — 凭证测试

**新建 `internal/auth/credential_test.go`**：

```go
package auth

import (
    "testing"
)

func TestCredentialEncryptDecrypt(t *testing.T) {
    cred := &Credential{APIKey: "test-key", SecretKey: "test-secret"}
    encrypted, err := Encrypt(cred)
    if err != nil {
        t.Fatal(err)
    }
    decrypted, err := Decrypt(encrypted)
    if err != nil {
        t.Fatal(err)
    }
    if decrypted.APIKey != cred.APIKey || decrypted.SecretKey != cred.SecretKey {
        t.Error("round trip failed")
    }
}

func TestTokenGeneration(t *testing.T) {
    token, err := GenerateToken("user1")
    if err != nil {
        t.Fatal(err)
    }
    if len(token) < 20 {
        t.Error("token too short")
    }
}
```

---

## Task 3: internal/logging — 日志测试

**新建 `internal/logging/logger_test.go`**：

```go
package logging

import (
    "bytes"
    "testing"
)

func TestLoggerWrite(t *testing.T) {
    var buf bytes.Buffer
    logger := New(&buf)
    logger.Info("test message")
    if !bytes.Contains(buf.Bytes(), []byte("test message")) {
        t.Error("log message not found")
    }
}
```

---

## Task 4: internal/schedule — 调度测试

**新建 `internal/schedule/scheduler_test.go`**：

```go
package schedule

import (
    "sync/atomic"
    "testing"
    "time"
)

func TestScheduler_RunOnce(t *testing.T) {
    s := New()
    var count atomic.Int32
    s.Add("test", func() { count.Add(1) }, time.Millisecond*10)
    time.Sleep(time.Millisecond * 25)
    s.Remove("test")
    if count.Load() < 1 {
        t.Error("expected at least 1 execution")
    }
}
```

---

## Task 5: internal/notify — 通知测试

**新建 `internal/notify/manager_test.go`**：

```go
package notify

import (
    "testing"
)

type mockNotifier struct{ messages []string }

func (m *mockNotifier) Send(title, body string) error {
    m.messages = append(m.messages, title+": "+body)
    return nil
}

func TestManager_Send(t *testing.T) {
    m := &mockNotifier{}
    mgr := New(m)
    mgr.Send("test", "body")
    if len(m.messages) != 1 {
        t.Error("expected 1 message")
    }
}
```

---

## Task 6: internal/trading/oms — 订单测试

**新建 `internal/trading/oms/order_test.go`**：

```go
package oms

import (
    "testing"
)

func TestOrderLifecycle(t *testing.T) {
    o := NewOrder("AAPL", Buy, Market, 100)
    if o.Status != New {
        t.Error("new order should have New status")
    }
    o.Submit()
    if o.Status != Submitted {
        t.Error("should be Submitted after Submit()")
    }
    o.Fill(100.50)
    if o.Status != Filled {
        t.Error("should be Filled after Fill()")
    }
    if o.AvgPrice != 100.50 {
        t.Error("avg price should be 100.50")
    }
}
```

---

## Task 7: internal/research — 补充测试

**新建 `internal/research/govdata_service_test.go`**（仅测试缓存逻辑，不调用真实 API）：

```go
package research

import (
    "testing"
    "time"
)

func TestGovDataService_CacheHit(t *testing.T) {
    s := NewGovDataService(nil)
    cacheKey := "test-key"
    s.cache[cacheKey] = &cacheEntry{
        data:      []adapters.SignalEntry{{Signal: "test"}},
        expiresAt: time.Now().Add(time.Hour),
    }
    result, err := s.GetSignals("test-key", nil)
    if err != nil || len(result) != 1 || result[0].Signal != "test" {
        t.Error("cache hit failed")
    }
}
```

---

## Task 8: 验证

```bash
go test ./internal/ws/ -v -count=1
go test ./internal/auth/ -v -count=1
go test ./internal/logging/ -v -count=1
go test ./internal/schedule/ -v -count=1
go test ./internal/notify/ -v -count=1
go test ./internal/trading/oms/ -v -count=1
go test ./internal/research/ -v -count=1
go test ./... -count=1
```
