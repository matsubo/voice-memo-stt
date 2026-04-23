package watcher_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/matsubo/voice-memo-stt/internal/watcher"
)

func TestWatch_DetectsNewM4A(t *testing.T) {
	dir := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()

	var called atomic.Int32
	done := make(chan struct{})

	go func() {
		_ = watcher.Watch(ctx, dir, func(_ context.Context, path string) error {
			if filepath.Ext(path) == ".m4a" {
				if called.Add(1) == 1 {
					close(done)
				}
			}
			return nil
		})
	}()

	time.Sleep(100 * time.Millisecond) // let watcher start
	f, err := os.Create(filepath.Join(dir, "test.m4a"))
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Error("timeout: watcher did not call fn within 5s")
	}
	if called.Load() != 1 {
		t.Errorf("fn called %d times, want 1", called.Load())
	}
}

func TestWatch_IgnoresNonM4A(t *testing.T) {
	dir := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var called atomic.Int32
	go func() {
		_ = watcher.Watch(ctx, dir, func(_ context.Context, _ string) error {
			called.Add(1)
			return nil
		})
	}()

	time.Sleep(100 * time.Millisecond)
	f, _ := os.Create(filepath.Join(dir, "test.txt"))
	f.Close()

	time.Sleep(2500 * time.Millisecond) // wait past debounce
	if called.Load() != 0 {
		t.Errorf("fn should not be called for .txt, called %d times", called.Load())
	}
}

func TestWatch_InvalidDir(t *testing.T) {
	ctx := context.Background()
	err := watcher.Watch(ctx, "/nonexistent-dir-vmt-test", func(_ context.Context, _ string) error {
		return nil
	})
	if err == nil {
		t.Error("expected error for non-existent directory")
	}
}

func TestWatch_ContextCancel(t *testing.T) {
	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- watcher.Watch(ctx, dir, func(_ context.Context, _ string) error {
			return nil
		})
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Watch should return nil on context cancel, got: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("Watch did not return after context cancel")
	}
}

func TestWatch_FnError(t *testing.T) {
	// fn returning an error should not crash Watch; error is logged.
	dir := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()

	done := make(chan struct{})
	var once atomic.Bool

	go func() {
		_ = watcher.Watch(ctx, dir, func(_ context.Context, _ string) error {
			if once.CompareAndSwap(false, true) {
				close(done)
			}
			return fmt.Errorf("simulated transcribe error")
		})
	}()

	time.Sleep(100 * time.Millisecond)
	f, err := os.Create(filepath.Join(dir, "err.m4a"))
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Error("timeout: fn was not called")
	}
}

func TestWatch_DebounceDeduplicate(t *testing.T) {
	// Writing the same file rapidly should trigger fn only once.
	dir := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	var called atomic.Int32
	done := make(chan struct{})
	var once atomic.Bool

	go func() {
		_ = watcher.Watch(ctx, dir, func(_ context.Context, _ string) error {
			n := called.Add(1)
			if n == 1 && once.CompareAndSwap(false, true) {
				close(done)
			}
			return nil
		})
	}()

	time.Sleep(100 * time.Millisecond)
	p := filepath.Join(dir, "rapid.m4a")
	// Write multiple times in quick succession to exercise the debounce reset path.
	for i := 0; i < 3; i++ {
		f, err := os.Create(p)
		if err != nil {
			t.Fatal(err)
		}
		f.Close()
		time.Sleep(50 * time.Millisecond)
	}

	select {
	case <-done:
	case <-time.After(6 * time.Second):
		t.Error("timeout waiting for debounced callback")
	}
	// After debounce window, exactly 1 call should have occurred.
	if n := called.Load(); n != 1 {
		t.Errorf("fn called %d times, want 1", n)
	}
}
