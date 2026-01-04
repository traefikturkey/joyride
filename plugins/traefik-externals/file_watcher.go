package traefikexternals

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/coredns/coredns/plugin/pkg/log"
	"github.com/fsnotify/fsnotify"
)

// FileWatcher monitors a directory for Traefik external config changes.
type FileWatcher struct {
	directory string
	hostIP    string
	records   *Records
	parser    *Parser

	watcher *fsnotify.Watcher
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	mu      sync.RWMutex
	running bool

	// debounce prevents rapid reloads on multiple file changes
	debounceTimer *time.Timer
	debounceMu    sync.Mutex
}

// NewFileWatcher creates a new file watcher for Traefik external configs.
func NewFileWatcher(directory, hostIP string, records *Records) *FileWatcher {
	return &FileWatcher{
		directory: directory,
		hostIP:    hostIP,
		records:   records,
		parser:    NewParser(),
	}
}

// Start begins watching the directory for changes.
func (fw *FileWatcher) Start(ctx context.Context) error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if fw.running {
		return nil
	}

	// Create fsnotify watcher while holding the lock to prevent race
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	// Add directory to watch
	if err := watcher.Add(fw.directory); err != nil {
		watcher.Close()
		return err
	}

	// All setup succeeded, now update state
	fw.watcher = watcher
	fw.running = true
	fw.ctx, fw.cancel = context.WithCancel(ctx)

	// Initial load (outside lock would be better but we need to ensure
	// it happens before returning)
	if err := fw.loadAllConfigs(); err != nil {
		log.Errorf("traefik-externals: initial load failed: %v", err)
	}

	// Start watch loop
	fw.wg.Add(1)
	go fw.watchLoop()

	log.Infof("traefik-externals: watching directory %s", fw.directory)
	return nil
}

// Stop halts the file watcher.
func (fw *FileWatcher) Stop() {
	fw.mu.Lock()
	if !fw.running {
		fw.mu.Unlock()
		return
	}
	fw.running = false
	fw.mu.Unlock()

	// Cancel any pending debounce timer
	fw.debounceMu.Lock()
	if fw.debounceTimer != nil {
		fw.debounceTimer.Stop()
		fw.debounceTimer = nil
	}
	fw.debounceMu.Unlock()

	if fw.cancel != nil {
		fw.cancel()
	}
	if fw.watcher != nil {
		fw.watcher.Close()
	}
	fw.wg.Wait()
}

// watchLoop handles file system events.
func (fw *FileWatcher) watchLoop() {
	defer fw.wg.Done()

	for {
		select {
		case <-fw.ctx.Done():
			return

		case event, ok := <-fw.watcher.Events:
			if !ok {
				return
			}
			fw.handleEvent(event)

		case err, ok := <-fw.watcher.Errors:
			if !ok {
				return
			}
			log.Errorf("traefik-externals: watcher error: %v", err)
		}
	}
}

// handleEvent processes a file system event.
func (fw *FileWatcher) handleEvent(event fsnotify.Event) {
	// Only care about YAML files
	if filepath.Ext(event.Name) != ".yml" && filepath.Ext(event.Name) != ".yaml" {
		return
	}

	// Skip middleware-only files
	if fw.parser.IsMiddlewareOnly(filepath.Base(event.Name)) {
		return
	}

	log.Debugf("traefik-externals: file event %s: %s", event.Op, event.Name)

	// Debounce rapid changes (e.g., editor save patterns)
	fw.debounceMu.Lock()
	if fw.debounceTimer != nil {
		fw.debounceTimer.Stop()
	}
	fw.debounceTimer = time.AfterFunc(500*time.Millisecond, func() {
		if err := fw.loadAllConfigs(); err != nil {
			log.Errorf("traefik-externals: reload failed: %v", err)
		}
	})
	fw.debounceMu.Unlock()
}

// loadAllConfigs reads all YAML files from the directory and updates records.
func (fw *FileWatcher) loadAllConfigs() error {
	entries, err := os.ReadDir(fw.directory)
	if err != nil {
		return err
	}

	allHosts := make(map[string]string)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Phase 2.1: Skip symlinks and non-regular files (security)
		info, err := entry.Info()
		if err != nil {
			log.Debugf("traefik-externals: skipping %s: %v", entry.Name(), err)
			continue
		}
		if !info.Mode().IsRegular() {
			log.Debugf("traefik-externals: skipping non-regular file %s", entry.Name())
			continue
		}

		name := entry.Name()
		ext := filepath.Ext(name)
		if ext != ".yml" && ext != ".yaml" {
			continue
		}

		// Skip middleware-only files
		if fw.parser.IsMiddlewareOnly(name) {
			log.Debugf("traefik-externals: skipping middleware file %s", name)
			continue
		}

		filePath := filepath.Join(fw.directory, name)
		hosts, err := fw.parser.ParseFile(filePath)
		if err != nil {
			log.Warningf("traefik-externals: failed to parse %s: %v", name, err)
			parseErrorsTotal.Inc()
			continue
		}

		for _, host := range hosts {
			allHosts[host] = fw.hostIP
		}
	}

	// Atomically replace all records
	fw.records.ReplaceAll(allHosts)

	// Update Prometheus metrics
	recordsTotal.Set(float64(len(allHosts)))
	reloadsTotal.Inc()

	if len(allHosts) > 0 {
		hostnames := make([]string, 0, len(allHosts))
		for h := range allHosts {
			hostnames = append(hostnames, h)
		}
		log.Infof("traefik-externals: loaded %d hostnames: %v", len(hostnames), hostnames)
	} else {
		log.Info("traefik-externals: no hostnames found in external configs")
	}

	return nil
}

// GetDirectory returns the watched directory path.
func (fw *FileWatcher) GetDirectory() string {
	return fw.directory
}
