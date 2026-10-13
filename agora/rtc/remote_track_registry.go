package agoraservice

import (
	"sync"
	"unsafe"
)

type remoteTrackKind uint8

const (
	remoteAudioTrack remoteTrackKind = iota
	remoteVideoTrack
)

type remoteTrackKey struct {
	kind   remoteTrackKind
	handle unsafe.Pointer
}

// remoteTrackRegistry associates opaque native track handles with public user
// identities. The numeric UID embedded in native statistics may be an internal
// ID in string-user mode, so it must never be used as a public identity fallback.
// Handles are only keys: they are never dereferenced or retained as native tracks.
// All callbacks also hold the connection's callback gate during lookup/update.
type remoteTrackRegistry struct {
	mu    sync.RWMutex
	users map[remoteTrackKey]string
}

func (r *remoteTrackRegistry) store(kind remoteTrackKind, handle unsafe.Pointer, uid string) {
	if handle == nil || uid == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.users == nil {
		r.users = make(map[remoteTrackKey]string)
	}
	r.users[remoteTrackKey{kind, handle}] = uid
}

func (r *remoteTrackRegistry) lookup(kind remoteTrackKind, handle unsafe.Pointer) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	uid, ok := r.users[remoteTrackKey{kind, handle}]
	return uid, ok
}

func (r *remoteTrackRegistry) stateChanged(kind remoteTrackKind, handle unsafe.Pointer, uid string, state int) {
	if state != 0 { // Native REMOTE_AUDIO/VIDEO_STATE_STOPPED is zero.
		r.store(kind, handle, uid)
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	key := remoteTrackKey{kind, handle}
	if r.users[key] == uid {
		delete(r.users, key)
	}
}

func (r *remoteTrackRegistry) remove(kind remoteTrackKind, uid string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for key, value := range r.users {
		if key.kind == kind && (uid == "" || value == uid) {
			delete(r.users, key)
		}
	}
}

func (r *remoteTrackRegistry) removeUser(uid string) {
	if uid == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for key, value := range r.users {
		if value == uid {
			delete(r.users, key)
		}
	}
}

func (r *remoteTrackRegistry) clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.users = nil
}
