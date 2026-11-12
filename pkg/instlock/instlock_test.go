package instlock

import (
	"testing"
	"time"
)

func TestTryLockFailsWhileLockedIgnoringCase(t *testing.T) {
	Lock("AbC")
	if TryLock("abc") {
		t.Fatal("instance is locked")
	}

	if !TryLock("def") {
		t.Fatal("other instances must not be affected")
	}
	Unlock("def")

	Unlock("abc")
	if !TryLock("abc") {
		t.Fatal("instance has been unlocked")
	}
	Unlock("abc")
}

func TestLockWaitsForUnlock(t *testing.T) {
	Lock("x")

	acquired := make(chan struct{})
	go func() {
		Lock("x")
		close(acquired)
		Unlock("x")
	}()

	select {
	case <-acquired:
		t.Fatal("lock acquired twice")
	case <-time.After(50 * time.Millisecond):
	}

	Unlock("x")

	select {
	case <-acquired:
	case <-time.After(time.Second):
		t.Fatal("waiting lock not acquired after unlock")
	}
}
