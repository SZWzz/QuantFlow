package schedule

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	cron    *cron.Cron
	repo    *Repo
	exec    WorkflowExecutor
	notify  Notifier
	mu      sync.Mutex
	running map[string]context.CancelFunc
}

func New(db *sql.DB, exec WorkflowExecutor, notify Notifier) *Scheduler {
	return &Scheduler{
		cron:    cron.New(cron.WithSeconds()),
		repo:    NewRepo(db),
		exec:    exec,
		notify:  notify,
		running: make(map[string]context.CancelFunc),
	}
}

func (s *Scheduler) Start() error {
	tasks, err := s.repo.List()
	if err != nil {
		return fmt.Errorf("scheduler start: %w", err)
	}
	for _, task := range tasks {
		if !task.Enabled {
			continue
		}
		if err := s.addTask(task); err != nil {
			slog.Error("failed to schedule task", "id", task.ID, "name", task.Name, "error", err)
		}
	}
	s.cron.Start()
	slog.Info("scheduler started", "tasks", len(tasks))
	return nil
}

func (s *Scheduler) Stop() { s.cron.Stop() }

func (s *Scheduler) CreateTask(task *Task) error {
	if err := s.repo.Create(task); err != nil {
		return fmt.Errorf("create task: %w", err)
	}
	if task.Enabled {
		return s.addTask(task)
	}
	return nil
}

func (s *Scheduler) UpdateTask(task *Task) error {
	s.removeAllEntries()
	if err := s.repo.Update(task); err != nil {
		return fmt.Errorf("update task: %w", err)
	}
	if task.Enabled {
		return s.addTask(task)
	}
	return nil
}

func (s *Scheduler) DeleteTask(id string) error {
	s.removeAllEntries()
	return s.repo.Delete(id)
}

func (s *Scheduler) ListTasks() ([]*Task, error) { return s.repo.List() }

func (s *Scheduler) addTask(task *Task) error {
	_, err := s.cron.AddFunc(task.CronExpr, func() { s.executeTask(task) })
	if err != nil {
		return fmt.Errorf("add cron: %w", err)
	}
	slog.Info("task scheduled", "id", task.ID, "name", task.Name, "cron", task.CronExpr)
	return nil
}

func (s *Scheduler) removeAllEntries() {
	for _, entry := range s.cron.Entries() {
		s.cron.Remove(entry.ID)
	}
}

func (s *Scheduler) executeTask(task *Task) {
	s.mu.Lock()
	if _, ok := s.running[task.ID]; ok {
		s.mu.Unlock()
		slog.Warn("task still running", "id", task.ID)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(task.TimeoutSec)*time.Second)
	s.running[task.ID] = cancel
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.running, task.ID)
		s.mu.Unlock()
		cancel()
	}()

	execID, err := s.exec.Execute(ctx, task.WorkflowID)
	status := "success"
	if err != nil {
		status = "error"
		slog.Error("task failed", "id", task.ID, "error", err)
	}
	s.repo.RecordRun(task.ID, status)
	if s.notify != nil {
		s.notify.SendTaskCompleted(task.Name, status == "success", execID)
	}
}
