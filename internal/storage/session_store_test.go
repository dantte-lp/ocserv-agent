package storage

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSessionStore(t *testing.T) {
	t.Run("creates store without TTL", func(t *testing.T) {
		store := NewSessionStore(0)
		assert.NotNil(t, store)
		assert.Equal(t, time.Duration(0), store.ttl)
		assert.NotNil(t, store.sessions)
	})

	t.Run("creates store with TTL", func(t *testing.T) {
		store := NewSessionStore(5 * time.Minute)
		assert.NotNil(t, store)
		assert.Equal(t, 5*time.Minute, store.ttl)
	})
}

func TestSessionStoreAdd(t *testing.T) {
	store := NewSessionStore(0)

	t.Run("adds session successfully", func(t *testing.T) {
		session := &VPNSession{
			SessionID: "session-001",
			Username:  "john.doe",
			ClientIP:  "203.0.113.1",
			VpnIP:     "10.8.0.2",
		}

		err := store.Add(session)
		require.NoError(t, err)

		// Verify session was added
		retrieved, err := store.Get("session-001")
		require.NoError(t, err)
		assert.Equal(t, session.SessionID, retrieved.SessionID)
		assert.Equal(t, session.Username, retrieved.Username)
	})

	t.Run("sets timestamps automatically", func(t *testing.T) {
		session := &VPNSession{
			SessionID: "session-002",
			Username:  "jane.smith",
		}

		err := store.Add(session)
		require.NoError(t, err)

		retrieved, err := store.Get("session-002")
		require.NoError(t, err)
		assert.False(t, retrieved.ConnectedAt.IsZero())
		assert.False(t, retrieved.LastActivity.IsZero())
	})

	t.Run("initializes metadata map", func(t *testing.T) {
		session := &VPNSession{
			SessionID: "session-003",
			Username:  "test.user",
		}

		err := store.Add(session)
		require.NoError(t, err)

		retrieved, err := store.Get("session-003")
		require.NoError(t, err)
		assert.NotNil(t, retrieved.Metadata)
	})

	t.Run("returns error for nil session", func(t *testing.T) {
		err := store.Add(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "session cannot be nil")
	})

	t.Run("returns error for empty session ID", func(t *testing.T) {
		session := &VPNSession{
			SessionID: "",
			Username:  "test.user",
		}

		err := store.Add(session)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "session ID cannot be empty")
	})

	t.Run("returns error for empty username", func(t *testing.T) {
		session := &VPNSession{
			SessionID: "session-004",
			Username:  "",
		}

		err := store.Add(session)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "username cannot be empty")
	})
}

func TestSessionStoreGet(t *testing.T) {
	store := NewSessionStore(0)

	session := &VPNSession{
		SessionID: "session-get-001",
		Username:  "test.user",
	}
	_ = store.Add(session)

	t.Run("retrieves existing session", func(t *testing.T) {
		retrieved, err := store.Get("session-get-001")
		require.NoError(t, err)
		assert.Equal(t, "session-get-001", retrieved.SessionID)
		assert.Equal(t, "test.user", retrieved.Username)
	})

	t.Run("returns error for non-existent session", func(t *testing.T) {
		_, err := store.Get("non-existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "session not found")
	})

	t.Run("returns error for empty session ID", func(t *testing.T) {
		_, err := store.Get("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "session ID cannot be empty")
	})
}

func TestSessionStoreUpdate(t *testing.T) {
	store := NewSessionStore(0)

	session := &VPNSession{
		SessionID: "session-update-001",
		Username:  "test.user",
		BytesIn:   1000,
		BytesOut:  2000,
	}
	_ = store.Add(session)

	t.Run("updates session successfully", func(t *testing.T) {
		err := store.Update("session-update-001", func(s *VPNSession) error {
			s.BytesIn = 5000
			s.BytesOut = 10000
			return nil
		})
		require.NoError(t, err)

		retrieved, _ := store.Get("session-update-001")
		assert.Equal(t, uint64(5000), retrieved.BytesIn)
		assert.Equal(t, uint64(10000), retrieved.BytesOut)
	})

	t.Run("updates LastActivity timestamp", func(t *testing.T) {
		oldActivity := time.Now().Add(-1 * time.Hour)
		session := &VPNSession{
			SessionID:    "session-update-002",
			Username:     "test.user",
			LastActivity: oldActivity,
		}
		_ = store.Add(session)

		time.Sleep(10 * time.Millisecond)

		err := store.Update("session-update-002", func(s *VPNSession) error {
			return nil
		})
		require.NoError(t, err)

		retrieved, _ := store.Get("session-update-002")
		assert.True(t, retrieved.LastActivity.After(oldActivity))
	})

	t.Run("returns error for non-existent session", func(t *testing.T) {
		err := store.Update("non-existent", func(s *VPNSession) error {
			return nil
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "session not found")
	})

	t.Run("returns error for empty session ID", func(t *testing.T) {
		err := store.Update("", func(s *VPNSession) error {
			return nil
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "session ID cannot be empty")
	})
}

func TestSessionStoreRemove(t *testing.T) {
	store := NewSessionStore(0)

	session := &VPNSession{
		SessionID: "session-remove-001",
		Username:  "test.user",
	}
	_ = store.Add(session)

	t.Run("removes existing session", func(t *testing.T) {
		err := store.Remove("session-remove-001")
		require.NoError(t, err)

		_, err = store.Get("session-remove-001")
		assert.Error(t, err)
	})

	t.Run("returns error for non-existent session", func(t *testing.T) {
		err := store.Remove("non-existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "session not found")
	})

	t.Run("returns error for empty session ID", func(t *testing.T) {
		err := store.Remove("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "session ID cannot be empty")
	})
}

func TestSessionStoreList(t *testing.T) {
	store := NewSessionStore(0)

	sessions := []*VPNSession{
		{SessionID: "s1", Username: "user1"},
		{SessionID: "s2", Username: "user2"},
		{SessionID: "s3", Username: "user3"},
	}

	for _, s := range sessions {
		_ = store.Add(s)
	}

	t.Run("lists all sessions", func(t *testing.T) {
		list := store.List()
		assert.Len(t, list, 3)
	})

	t.Run("returns empty list for empty store", func(t *testing.T) {
		emptyStore := NewSessionStore(0)
		list := emptyStore.List()
		assert.Empty(t, list)
	})
}

func TestSessionStoreListByUsername(t *testing.T) {
	store := NewSessionStore(0)

	sessions := []*VPNSession{
		{SessionID: "s1", Username: "user1"},
		{SessionID: "s2", Username: "user2"},
		{SessionID: "s3", Username: "user1"},
		{SessionID: "s4", Username: "user2"},
	}

	for _, s := range sessions {
		_ = store.Add(s)
	}

	t.Run("lists sessions for specific user", func(t *testing.T) {
		list := store.ListByUsername("user1")
		assert.Len(t, list, 2)

		for _, s := range list {
			assert.Equal(t, "user1", s.Username)
		}
	})

	t.Run("returns empty list for non-existent user", func(t *testing.T) {
		list := store.ListByUsername("non-existent")
		assert.Empty(t, list)
	})

	t.Run("returns empty list for empty username", func(t *testing.T) {
		list := store.ListByUsername("")
		assert.Empty(t, list)
	})
}

func TestSessionStoreCount(t *testing.T) {
	store := NewSessionStore(0)

	t.Run("counts sessions correctly", func(t *testing.T) {
		assert.Equal(t, 0, store.Count())

		_ = store.Add(&VPNSession{SessionID: "s1", Username: "user1"})
		assert.Equal(t, 1, store.Count())

		_ = store.Add(&VPNSession{SessionID: "s2", Username: "user2"})
		assert.Equal(t, 2, store.Count())

		_ = store.Remove("s1")
		assert.Equal(t, 1, store.Count())
	})
}

func TestSessionStoreCountByUsername(t *testing.T) {
	store := NewSessionStore(0)

	sessions := []*VPNSession{
		{SessionID: "s1", Username: "user1"},
		{SessionID: "s2", Username: "user2"},
		{SessionID: "s3", Username: "user1"},
	}

	for _, s := range sessions {
		_ = store.Add(s)
	}

	t.Run("counts sessions by username", func(t *testing.T) {
		assert.Equal(t, 2, store.CountByUsername("user1"))
		assert.Equal(t, 1, store.CountByUsername("user2"))
		assert.Equal(t, 0, store.CountByUsername("non-existent"))
	})
}

func TestSessionStoreClear(t *testing.T) {
	store := NewSessionStore(0)

	sessions := []*VPNSession{
		{SessionID: "s1", Username: "user1"},
		{SessionID: "s2", Username: "user2"},
	}

	for _, s := range sessions {
		_ = store.Add(s)
	}

	t.Run("clears all sessions", func(t *testing.T) {
		assert.Equal(t, 2, store.Count())

		store.Clear()

		assert.Equal(t, 0, store.Count())
		assert.Empty(t, store.List())
	})
}

func TestSessionStoreRemoveByUsername(t *testing.T) {
	store := NewSessionStore(0)

	sessions := []*VPNSession{
		{SessionID: "s1", Username: "user1"},
		{SessionID: "s2", Username: "user2"},
		{SessionID: "s3", Username: "user1"},
	}

	for _, s := range sessions {
		_ = store.Add(s)
	}

	t.Run("removes all sessions for user", func(t *testing.T) {
		count := store.RemoveByUsername("user1")
		assert.Equal(t, 2, count)

		assert.Equal(t, 0, store.CountByUsername("user1"))
		assert.Equal(t, 1, store.CountByUsername("user2"))
	})

	t.Run("returns 0 for non-existent user", func(t *testing.T) {
		count := store.RemoveByUsername("non-existent")
		assert.Equal(t, 0, count)
	})

	t.Run("returns 0 for empty username", func(t *testing.T) {
		count := store.RemoveByUsername("")
		assert.Equal(t, 0, count)
	})
}

func TestSessionStoreUpdateStats(t *testing.T) {
	store := NewSessionStore(0)

	session := &VPNSession{
		SessionID: "s1",
		Username:  "user1",
		BytesIn:   1000,
		BytesOut:  2000,
	}
	_ = store.Add(session)

	t.Run("updates stats successfully", func(t *testing.T) {
		err := store.UpdateStats("s1", 5000, 10000)
		require.NoError(t, err)

		retrieved, _ := store.Get("s1")
		assert.Equal(t, uint64(5000), retrieved.BytesIn)
		assert.Equal(t, uint64(10000), retrieved.BytesOut)
	})
}

func TestSessionStoreGetStats(t *testing.T) {
	store := NewSessionStore(0)

	sessions := []*VPNSession{
		{SessionID: "s1", Username: "user1", BytesIn: 1000, BytesOut: 2000},
		{SessionID: "s2", Username: "user2", BytesIn: 3000, BytesOut: 4000},
		{SessionID: "s3", Username: "user1", BytesIn: 5000, BytesOut: 6000},
	}

	for _, s := range sessions {
		_ = store.Add(s)
	}

	t.Run("calculates stats correctly", func(t *testing.T) {
		stats := store.GetStats()

		assert.Equal(t, 3, stats.TotalSessions)
		assert.Equal(t, uint64(9000), stats.TotalBytesIn)
		assert.Equal(t, uint64(12000), stats.TotalBytesOut)
		assert.Equal(t, 2, stats.UserSessions["user1"])
		assert.Equal(t, 1, stats.UserSessions["user2"])
	})
}

func TestSessionStoreExists(t *testing.T) {
	store := NewSessionStore(0)

	session := &VPNSession{
		SessionID: "s1",
		Username:  "user1",
	}
	_ = store.Add(session)

	t.Run("returns true for existing session", func(t *testing.T) {
		assert.True(t, store.Exists("s1"))
	})

	t.Run("returns false for non-existent session", func(t *testing.T) {
		assert.False(t, store.Exists("non-existent"))
	})
}

func TestSessionStoreGetOrCreate(t *testing.T) {
	store := NewSessionStore(0)

	t.Run("creates new session if not exists", func(t *testing.T) {
		session := &VPNSession{
			SessionID: "s1",
			Username:  "user1",
		}

		retrieved, created, err := store.GetOrCreate(session)
		require.NoError(t, err)
		assert.True(t, created)
		assert.Equal(t, "s1", retrieved.SessionID)
	})

	t.Run("returns existing session if exists", func(t *testing.T) {
		session := &VPNSession{
			SessionID: "s2",
			Username:  "user2",
		}
		_ = store.Add(session)

		retrieved, created, err := store.GetOrCreate(session)
		require.NoError(t, err)
		assert.False(t, created)
		assert.Equal(t, "s2", retrieved.SessionID)
	})

	t.Run("returns error for invalid session", func(t *testing.T) {
		_, _, err := store.GetOrCreate(nil)
		assert.Error(t, err)
	})
}

func TestSessionStoreTTL(t *testing.T) {
	t.Run("expires sessions after TTL", func(t *testing.T) {
		store := NewSessionStore(100 * time.Millisecond)

		session := &VPNSession{
			SessionID: "s1",
			Username:  "user1",
		}
		_ = store.Add(session)

		// Session should exist initially
		assert.True(t, store.Exists("s1"))

		// Wait for TTL to expire
		time.Sleep(150 * time.Millisecond)

		// Session should be expired or removed by cleanup
		_, err := store.Get("s1")
		assert.Error(t, err)
		// Can be either "session expired" or "session not found" depending on cleanup timing
		assert.True(t,
			strings.Contains(err.Error(), "session expired") ||
			strings.Contains(err.Error(), "session not found"),
			"Expected expired or not found error, got: %s", err.Error())
	})

	t.Run("excludes expired sessions from list", func(t *testing.T) {
		store := NewSessionStore(100 * time.Millisecond)

		_ = store.Add(&VPNSession{SessionID: "s1", Username: "user1"})
		time.Sleep(150 * time.Millisecond)

		list := store.List()
		assert.Empty(t, list)
	})

	t.Run("updates ExpiresAt on update", func(t *testing.T) {
		store := NewSessionStore(1 * time.Second)

		session := &VPNSession{
			SessionID: "s1",
			Username:  "user1",
		}
		_ = store.Add(session)

		// Get initial expiry
		retrieved, _ := store.Get("s1")
		initialExpiry := *retrieved.ExpiresAt

		time.Sleep(100 * time.Millisecond)

		// Update session
		_ = store.Update("s1", func(s *VPNSession) error {
			return nil
		})

		// Get new expiry
		retrieved, _ = store.Get("s1")
		newExpiry := *retrieved.ExpiresAt

		// New expiry should be later
		assert.True(t, newExpiry.After(initialExpiry))
	})
}

func TestSessionStoreThreadSafety(t *testing.T) {
	store := NewSessionStore(0)

	const numGoroutines = 100
	const sessionsPerGoroutine = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < sessionsPerGoroutine; j++ {
				session := &VPNSession{
					SessionID: "session-" + string(rune(id*sessionsPerGoroutine+j)),
					Username:  "user-" + string(rune(id)),
				}
				_ = store.Add(session)
			}
		}(i)
	}

	wg.Wait()

	// Verify no race conditions occurred
	count := store.Count()
	assert.True(t, count > 0, "should have sessions after concurrent writes")
}

func TestSessionStoreCleanup(t *testing.T) {
	t.Run("cleanup goroutine removes expired sessions", func(t *testing.T) {
		store := NewSessionStore(200 * time.Millisecond)

		// Add sessions
		for i := 0; i < 5; i++ {
			session := &VPNSession{
				SessionID: "cleanup-" + string(rune(i)),
				Username:  "user",
			}
			_ = store.Add(session)
		}

		assert.Equal(t, 5, store.Count())

		// Wait for cleanup to run (TTL/2 + some buffer)
		time.Sleep(300 * time.Millisecond)

		// Expired sessions should be cleaned up
		count := store.Count()
		assert.Equal(t, 0, count, "all sessions should be cleaned up")
	})
}
