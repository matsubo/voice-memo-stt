package watcher

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

const debounceDelay = 2 * time.Second

// TranscribeFunc is the callback invoked for each new .m4a file detected.
type TranscribeFunc func(ctx context.Context, audioPath string) error

// Watch monitors dir for new .m4a files and calls fn for each, with a 2s debounce.
// It blocks until ctx is cancelled or a fatal watcher error occurs.
func Watch(ctx context.Context, dir string, fn TranscribeFunc) error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create watcher: %w", err)
	}
	defer w.Close()

	if err := w.Add(dir); err != nil {
		return fmt.Errorf("watch %q: %w", dir, err)
	}

	var mu sync.Mutex
	pending := map[string]*time.Timer{}

	for {
		select {
		case <-ctx.Done():
			mu.Lock()
			for _, t := range pending {
				t.Stop()
			}
			mu.Unlock()
			return nil
		case event, ok := <-w.Events:
			if !ok {
				return nil
			}
			if event.Op&(fsnotify.Create|fsnotify.Write) == 0 {
				continue
			}
			if !strings.HasSuffix(event.Name, ".m4a") {
				continue
			}
			path := event.Name

			mu.Lock()
			if t, exists := pending[path]; exists {
				t.Reset(debounceDelay)
				mu.Unlock()
			} else {
				pending[path] = time.AfterFunc(debounceDelay, func() {
					mu.Lock()
					delete(pending, path)
					mu.Unlock()
					log.Printf("[%s] Transcribing: %s", time.Now().Format("2006-01-02 15:04"), filepath.Base(path))
					if err := fn(ctx, path); err != nil {
						log.Printf("transcribe error: %v", err)
					}
				})
				mu.Unlock()
			}
		case err, ok := <-w.Errors:
			if !ok {
				return nil
			}
			log.Printf("watcher error: %v", err)
		}
	}
}
