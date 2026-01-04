package dockercluster

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/coredns/coredns/plugin/pkg/log"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

// RecordChangeCallback is called when DNS records are added or removed.
// timestamp is Unix nanoseconds, used for cluster LWW conflict resolution.
type RecordChangeCallback func(hostname, ip string, added bool, timestamp int64)

// DockerWatcher monitors Docker container events and updates DNS records.
type DockerWatcher struct {
	dockerSocket string
	hostIP       string
	labels       []string
	records      *Records
	callback     RecordChangeCallback

	client     *client.Client
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	mu         sync.RWMutex
	running    bool
	containers map[string][]string // containerID -> hostnames
}

// NewDockerWatcher creates a new Docker watcher.
func NewDockerWatcher(dockerSocket, hostIP string, labels []string, records *Records) *DockerWatcher {
	return &DockerWatcher{
		dockerSocket: dockerSocket,
		hostIP:       hostIP,
		labels:       labels,
		records:      records,
		containers:   make(map[string][]string),
	}
}

// SetCallback sets the callback for record changes.
// If set, the callback is invoked for each record add/remove.
func (dw *DockerWatcher) SetCallback(cb RecordChangeCallback) {
	dw.mu.Lock()
	defer dw.mu.Unlock()
	dw.callback = cb
}

// Start begins watching Docker for container events.
func (dw *DockerWatcher) Start(ctx context.Context) error {
	dw.mu.Lock()
	if dw.running {
		dw.mu.Unlock()
		return nil
	}
	dw.running = true
	dw.ctx, dw.cancel = context.WithCancel(ctx)
	dw.mu.Unlock()

	dw.wg.Add(1)
	go dw.watchLoop()

	return nil
}

// Stop halts the Docker watcher and waits for goroutines to finish.
func (dw *DockerWatcher) Stop() {
	dw.mu.Lock()
	if !dw.running {
		dw.mu.Unlock()
		return
	}
	dw.running = false
	dw.mu.Unlock()

	if dw.cancel != nil {
		dw.cancel()
	}
	dw.wg.Wait()

	dw.mu.Lock()
	if dw.client != nil {
		dw.client.Close()
		dw.client = nil
	}
	dw.mu.Unlock()
}

// watchLoop handles connection, syncing, and event watching with exponential backoff.
func (dw *DockerWatcher) watchLoop() {
	defer dw.wg.Done()

	backoff := time.Second
	maxBackoff := 5 * time.Minute

	for {
		select {
		case <-dw.ctx.Done():
			return
		default:
		}

		// Connect to Docker
		if err := dw.connect(); err != nil {
			log.Errorf("docker-cluster: failed to connect to Docker: %v", err)
			dw.sleep(backoff)
			backoff = dw.nextBackoff(backoff, maxBackoff)
			continue
		}

		// Reset backoff on successful connection
		backoff = time.Second

		// Sync existing containers
		if err := dw.syncContainers(); err != nil {
			log.Errorf("docker-cluster: failed to sync containers: %v", err)
			dw.closeClient()
			dw.sleep(backoff)
			backoff = dw.nextBackoff(backoff, maxBackoff)
			continue
		}

		// Watch for events
		if err := dw.watchEvents(); err != nil {
			log.Errorf("docker-cluster: event stream error: %v", err)
			dw.closeClient()
			dw.sleep(backoff)
			backoff = dw.nextBackoff(backoff, maxBackoff)
			continue
		}
	}
}

// connect establishes a connection to the Docker daemon.
func (dw *DockerWatcher) connect() error {
	opts := []client.Opt{
		client.WithAPIVersionNegotiation(),
	}

	if dw.dockerSocket != "" {
		opts = append(opts, client.WithHost(dw.dockerSocket))
	}

	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return err
	}

	// Verify connection by pinging the daemon
	_, err = cli.Ping(dw.ctx)
	if err != nil {
		cli.Close()
		return err
	}

	dw.mu.Lock()
	dw.client = cli
	dw.mu.Unlock()

	log.Info("docker-cluster: connected to Docker daemon")
	return nil
}

// closeClient closes the Docker client connection.
func (dw *DockerWatcher) closeClient() {
	dw.mu.Lock()
	defer dw.mu.Unlock()

	if dw.client != nil {
		dw.client.Close()
		dw.client = nil
	}
}

// syncContainers fetches all running containers and syncs their DNS records.
func (dw *DockerWatcher) syncContainers() error {
	dw.mu.RLock()
	cli := dw.client
	dw.mu.RUnlock()

	if cli == nil {
		return nil
	}

	// Get all running containers
	containers, err := cli.ContainerList(dw.ctx, container.ListOptions{
		Filters: filters.NewArgs(filters.Arg("status", "running")),
	})
	if err != nil {
		return err
	}

	// Track which containers we've seen
	seen := make(map[string]bool)

	// Process each container
	for _, c := range containers {
		hostnames := dw.extractHostnames(c.Labels)
		if len(hostnames) > 0 {
			seen[c.ID] = true
			dw.updateContainer(c.ID, hostnames)
		}
	}

	// Remove containers that no longer exist
	var removedHostnames []string
	dw.mu.Lock()
	for id, hostnames := range dw.containers {
		if !seen[id] {
			for _, hostname := range hostnames {
				dw.records.Remove(hostname)
				removedHostnames = append(removedHostnames, hostname)
			}
			delete(dw.containers, id)
		}
	}
	dw.mu.Unlock()

	// Invoke callback for removed hostnames outside of lock
	if len(removedHostnames) > 0 {
		dw.mu.RLock()
		cb := dw.callback
		dw.mu.RUnlock()
		if cb != nil {
			ts := time.Now().UnixNano()
			for _, hostname := range removedHostnames {
				cb(hostname, "", false, ts)
			}
		}
	}

	log.Infof("docker-cluster: synced %d containers with DNS records", len(seen))
	dw.logCurrentState()
	return nil
}

// watchEvents subscribes to Docker events and processes container start/stop events.
func (dw *DockerWatcher) watchEvents() error {
	dw.mu.RLock()
	cli := dw.client
	dw.mu.RUnlock()

	if cli == nil {
		return nil
	}

	// Subscribe to container events
	eventFilter := filters.NewArgs()
	eventFilter.Add("type", "container")
	eventFilter.Add("event", "start")
	eventFilter.Add("event", "stop")
	eventFilter.Add("event", "die")

	eventChan, errChan := cli.Events(dw.ctx, events.ListOptions{
		Filters: eventFilter,
	})

	log.Info("docker-cluster: watching for container events")

	for {
		select {
		case <-dw.ctx.Done():
			return nil

		case err := <-errChan:
			return err

		case event := <-eventChan:
			dw.handleEvent(event)
		}
	}
}

// handleEvent processes a single Docker event.
func (dw *DockerWatcher) handleEvent(event events.Message) {
	switch event.Action {
	case "start":
		dw.handleContainerStart(event.Actor.ID)
	case "stop", "die":
		dw.handleContainerStop(event.Actor.ID)
	}
}

// handleContainerStart processes a container start event.
func (dw *DockerWatcher) handleContainerStart(containerID string) {
	dw.mu.RLock()
	cli := dw.client
	dw.mu.RUnlock()

	if cli == nil {
		return
	}

	// Inspect container to get labels
	info, err := cli.ContainerInspect(dw.ctx, containerID)
	if err != nil {
		log.Errorf("docker-cluster: failed to inspect container %s: %v", containerID[:12], err)
		return
	}

	hostnames := dw.extractHostnames(info.Config.Labels)
	if len(hostnames) > 0 {
		dw.updateContainer(containerID, hostnames)
		log.Infof("docker-cluster: container %s started with hostnames: %v", containerID[:12], hostnames)
		dw.logCurrentState()
	}
}

// handleContainerStop processes a container stop or die event.
func (dw *DockerWatcher) handleContainerStop(containerID string) {
	dw.mu.Lock()
	hostnames, exists := dw.containers[containerID]
	if exists {
		delete(dw.containers, containerID)
	}
	dw.mu.Unlock()

	if exists {
		for _, hostname := range hostnames {
			dw.records.Remove(hostname)
		}

		// Invoke callback outside of lock
		dw.mu.RLock()
		cb := dw.callback
		dw.mu.RUnlock()
		if cb != nil {
			ts := time.Now().UnixNano()
			for _, hostname := range hostnames {
				cb(hostname, "", false, ts)
			}
		}

		log.Infof("docker-cluster: container %s stopped, removed hostnames: %v", containerID[:12], hostnames)
		dw.logCurrentState()
	}
}

// updateContainer updates the DNS records for a container.
func (dw *DockerWatcher) updateContainer(containerID string, newHostnames []string) {
	dw.mu.Lock()
	oldHostnames := dw.containers[containerID]
	dw.containers[containerID] = newHostnames
	dw.mu.Unlock()

	// Build sets for comparison
	oldSet := make(map[string]bool)
	for _, h := range oldHostnames {
		oldSet[h] = true
	}

	newSet := make(map[string]bool)
	for _, h := range newHostnames {
		newSet[h] = true
	}

	// Track removed and added hostnames for callback
	var removed []string
	var added []string

	// Remove old hostnames not in new set
	for _, h := range oldHostnames {
		if !newSet[h] {
			dw.records.Remove(h)
			removed = append(removed, h)
		}
	}

	// Add new hostnames
	for _, h := range newHostnames {
		if !oldSet[h] {
			dw.records.Add(h, dw.hostIP)
			added = append(added, h)
		}
	}

	// Invoke callback outside of lock
	dw.mu.RLock()
	cb := dw.callback
	dw.mu.RUnlock()
	if cb != nil {
		ts := time.Now().UnixNano()
		for _, h := range removed {
			cb(h, "", false, ts)
		}
		for _, h := range added {
			cb(h, dw.hostIP, true, ts)
		}
	}
}

// extractHostnames extracts hostnames from container labels.
func (dw *DockerWatcher) extractHostnames(labels map[string]string) []string {
	var hostnames []string
	seen := make(map[string]bool)

	for _, labelName := range dw.labels {
		if value, ok := labels[labelName]; ok {
			parts := strings.Split(value, ",")
			for _, part := range parts {
				hostname := strings.TrimSpace(part)
				if hostname != "" && !seen[hostname] {
					seen[hostname] = true
					hostnames = append(hostnames, hostname)
				}
			}
		}
	}

	return hostnames
}

// sleep waits for the specified duration or until context is cancelled.
func (dw *DockerWatcher) sleep(d time.Duration) {
	select {
	case <-time.After(d):
	case <-dw.ctx.Done():
	}
}

// nextBackoff calculates the next backoff duration.
func (dw *DockerWatcher) nextBackoff(current, max time.Duration) time.Duration {
	next := current * 2
	if next > max {
		return max
	}
	return next
}

// logCurrentState logs all currently registered hostnames.
func (dw *DockerWatcher) logCurrentState() {
	allRecords := dw.records.GetAll()
	if len(allRecords) == 0 {
		log.Info("docker-cluster: no hostnames registered")
		return
	}

	hostnames := make([]string, 0, len(allRecords))
	for hostname := range allRecords {
		hostnames = append(hostnames, hostname)
	}
	log.Infof("docker-cluster: registered hostnames: %v", hostnames)
}
