package leases

import (
	"sync"
	"time"
)

const DefaultLeaseDuration = 10 * time.Second

type LeaseManager struct {
	mu sync.RWMutex
	// taskID -> expiry time
	leases map[string]time.Time
}

func NewLeaseManager() *LeaseManager {
	return &LeaseManager{
		leases: make(map[string]time.Time),
	}
}

func (lm *LeaseManager) Acquire(taskID string, duration time.Duration) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.leases[taskID] = time.Now().Add(duration)
}

func (lm *LeaseManager) Release(taskID string) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	delete(lm.leases, taskID)
}

func (lm *LeaseManager) IsExpired(taskID string) bool {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	expiry, exists := lm.leases[taskID]
	if !exists {
		return true
	}
	return time.Now().After(expiry)
}

func (lm *LeaseManager) GetExpired() []string {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	now := time.Now()
	var expired []string
	for taskID, expiry := range lm.leases {
		if now.After(expiry) {
			expired = append(expired, taskID)
		}
	}
	return expired
}

func (lm *LeaseManager) Renew(taskID string, duration time.Duration) bool {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	if _, exists := lm.leases[taskID]; exists {
		lm.leases[taskID] = time.Now().Add(duration)
		return true
	}
	return false
}

