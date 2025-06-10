package bot

import (
	"sync"
)

// UserLockManager provides a thread-safe way to lock user interactions to prevent
// concurrent request processing for the same user.
type UserLockManager struct {
	locks map[int64]bool
	mu    sync.Mutex
}

// NewUserLockManager creates a new UserLockManager.
func NewUserLockManager() *UserLockManager {
	return &UserLockManager{
		locks: make(map[int64]bool),
	}
}

// TryLock attempts to acquire a lock for a given userID.
// It returns true if the lock was acquired, and false if the user is already locked.
// This operation is non-blocking.
func (ulm *UserLockManager) TryLock(userID int64) bool {
	ulm.mu.Lock()
	defer ulm.mu.Unlock()

	if ulm.locks[userID] {
		// User is already locked by another in-progress request.
		return false
	}

	// User is not locked, acquire the lock.
	ulm.locks[userID] = true
	return true
}

// Unlock releases the lock for a given userID.
func (ulm *UserLockManager) Unlock(userID int64) {
	ulm.mu.Lock()
	defer ulm.mu.Unlock()

	// It's safe to delete even if the key doesn't exist.
	delete(ulm.locks, userID)
}
