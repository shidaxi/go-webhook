package engine

import (
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/shidaxi/go-webhook/internal/config"
)

// RuleStore provides thread-safe access to compiled rules.
// It uses atomic.Value for lock-free reads and atomic swaps on reload.
type RuleStore struct {
	rules atomic.Value // holds []CompiledRule
}

// NewRuleStore creates a new empty RuleStore.
func NewRuleStore() *RuleStore {
	s := &RuleStore{}
	s.rules.Store([]CompiledRule{})
	return s
}

// GetRules returns the current compiled rules snapshot.
func (s *RuleStore) GetRules() []CompiledRule {
	return s.rules.Load().([]CompiledRule)
}

// LoadAndCompile loads rules from file, compiles them, and atomically replaces
// the current rule set. On file load failure, old rules are preserved.
func (s *RuleStore) LoadAndCompile(path string) error {
	rawRules, err := config.LoadRulesFromFile(path)
	if err != nil {
		return fmt.Errorf("failed to load rules: %w", err)
	}

	compiled := CompileRules(rawRules)
	s.rules.Store(compiled)
	return nil
}

// WatchRules starts watching the rules file for changes using fsnotify.
// On change, it reloads and recompiles rules. On failure, old rules are kept.
// Returns a stop function to cancel watching.
func (s *RuleStore) WatchRules(path string) (func(), error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	if err := watcher.Add(path); err != nil {
		watcher.Close()
		return nil, fmt.Errorf("failed to watch %s: %w", path, err)
	}

	done := make(chan struct{})
	go func() {
		var debounceTimer *time.Timer
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
					// Debounce: editors may trigger multiple events
					if debounceTimer != nil {
						debounceTimer.Stop()
					}
					debounceTimer = time.AfterFunc(200*time.Millisecond, func() {
						if err := s.LoadAndCompile(path); err != nil {
							log.Printf("rules reload failed (keeping old rules): %v", err)
						} else {
							log.Printf("rules reloaded successfully from %s", path)
						}
					})
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("file watcher error: %v", err)
			case <-done:
				return
			}
		}
	}()

	stop := func() {
		close(done)
		watcher.Close()
	}
	return stop, nil
}
