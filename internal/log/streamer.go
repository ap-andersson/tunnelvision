package log

import (
	"bufio"
	"os/exec"
	"sync"
)

const ringSize = 500

// Streamer tails journalctl for NetworkManager and WireGuard kernel logs.
type Streamer struct {
	mu       sync.Mutex
	ring     []string
	pos      int // next write position in ring buffer
	count    int // total lines written (for knowing if ring has wrapped)
	cmd      *exec.Cmd
	listener func(line string)
	stopCh   chan struct{}
}

// NewStreamer creates a new log Streamer.
func NewStreamer() *Streamer {
	return &Streamer{
		ring:   make([]string, ringSize),
		stopCh: make(chan struct{}),
	}
}

// SetListener sets a callback invoked for each new log line.
func (s *Streamer) SetListener(fn func(line string)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listener = fn
}

// Lines returns the buffered log lines in chronological order.
func (s *Streamer) Lines() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.count == 0 {
		return nil
	}

	n := s.count
	if n > ringSize {
		n = ringSize
	}
	result := make([]string, 0, n)

	if s.count <= ringSize {
		// Ring hasn't wrapped yet
		for i := 0; i < s.pos; i++ {
			result = append(result, s.ring[i])
		}
	} else {
		// Ring has wrapped: read from pos..end, then 0..pos
		for i := s.pos; i < ringSize; i++ {
			result = append(result, s.ring[i])
		}
		for i := 0; i < s.pos; i++ {
			result = append(result, s.ring[i])
		}
	}
	return result
}

// Start begins tailing journalctl. It runs until Stop is called.
func (s *Streamer) Start() error {
	s.cmd = exec.Command("journalctl", "-f", "-n", "200", "--no-pager", "-o", "short-precise",
		"_SYSTEMD_UNIT=NetworkManager.service",
	)

	stdout, err := s.cmd.StdoutPipe()
	if err != nil {
		return err
	}

	if err := s.cmd.Start(); err != nil {
		return err
	}

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			s.mu.Lock()
			s.ring[s.pos] = line
			s.pos = (s.pos + 1) % ringSize
			s.count++
			fn := s.listener
			s.mu.Unlock()

			if fn != nil {
				fn(line)
			}
		}
	}()

	return nil
}

// Stop kills the journalctl subprocess.
func (s *Streamer) Stop() {
	if s.cmd != nil && s.cmd.Process != nil {
		s.cmd.Process.Kill()
		s.cmd.Wait()
	}
}
