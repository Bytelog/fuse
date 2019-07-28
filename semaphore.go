package fuse

import "sync"

type semaphore struct {
	avail int64
	mu    sync.Mutex
}

func (s *semaphore) tryAcquire(n int64) bool {
	s.mu.Lock()
	success := s.avail >= n
	if success {
		s.avail -= n
	}
	s.mu.Unlock()
	return success
}

func (s *semaphore) release(n int64) {
	s.mu.Lock()
	s.avail += n
	s.mu.Unlock()
}
