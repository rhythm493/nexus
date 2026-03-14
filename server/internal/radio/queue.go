package radio

import (
	"errors"
	"sync"
	"time"

	"github.com/rhythm493/pocket-assistant/server/internal/library"
)

// Common queue errors
var (
	ErrQueueEmpty         = errors.New("queue is empty")
	ErrInvalidPosition    = errors.New("invalid queue position")
	ErrPositionOutOfRange = errors.New("position out of range")
)

// QueueItem represents a track in the queue.
type QueueItem struct {
	Track      *library.Track `json:"track"`
	Position   int            `json:"position"`
	AddedAt    time.Time      `json:"added_at"`
	RequestID  string         `json:"request_id,omitempty"` // Liquidsoap request ID
	IsPlaying  bool           `json:"is_playing"`
}

// Queue manages the in-memory queue that syncs with Liquidsoap.
type Queue struct {
	items   []*QueueItem
	current int // Index of currently playing item (-1 if nothing playing)
	mu      sync.RWMutex
}

// NewQueue creates a new empty queue.
func NewQueue() *Queue {
	return &Queue{
		items:   make([]*QueueItem, 0),
		current: -1,
	}
}

// Add adds a track to the end of the queue and returns its position.
func (q *Queue) Add(track *library.Track) int {
	q.mu.Lock()
	defer q.mu.Unlock()

	position := len(q.items)
	item := &QueueItem{
		Track:    track,
		Position: position,
		AddedAt:  time.Now(),
	}
	q.items = append(q.items, item)

	// If nothing is playing, mark this as current
	if q.current < 0 {
		q.current = position
		item.IsPlaying = true
	}

	return position
}

// AddWithRequestID adds a track with a Liquidsoap request ID.
func (q *Queue) AddWithRequestID(track *library.Track, requestID string) int {
	q.mu.Lock()
	defer q.mu.Unlock()

	position := len(q.items)
	item := &QueueItem{
		Track:     track,
		Position:  position,
		AddedAt:   time.Now(),
		RequestID: requestID,
	}
	q.items = append(q.items, item)

	// If nothing is playing, mark this as current
	if q.current < 0 {
		q.current = position
		item.IsPlaying = true
	}

	return position
}

// Current returns the currently playing item, or nil if nothing is playing.
func (q *Queue) Current() *QueueItem {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if q.current < 0 || q.current >= len(q.items) {
		return nil
	}
	return q.items[q.current]
}

// Next returns the next item in the queue without advancing.
// Returns nil if there is no next item.
func (q *Queue) Next() *QueueItem {
	q.mu.RLock()
	defer q.mu.RUnlock()

	nextIdx := q.current + 1
	if nextIdx >= len(q.items) {
		return nil
	}
	return q.items[nextIdx]
}

// Advance moves to the next track in the queue.
// Returns the new current item, or nil if queue is exhausted.
func (q *Queue) Advance() *QueueItem {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Mark current as not playing
	if q.current >= 0 && q.current < len(q.items) {
		q.items[q.current].IsPlaying = false
	}

	// Move to next
	q.current++

	// Check if we're past the end
	if q.current >= len(q.items) {
		q.current = -1 // Reset to "nothing playing"
		return nil
	}

	q.items[q.current].IsPlaying = true
	return q.items[q.current]
}

// List returns a copy of all items in the queue.
func (q *Queue) List() []*QueueItem {
	q.mu.RLock()
	defer q.mu.RUnlock()

	result := make([]*QueueItem, len(q.items))
	copy(result, q.items)
	return result
}

// Upcoming returns items that haven't been played yet (including current).
func (q *Queue) Upcoming() []*QueueItem {
	q.mu.RLock()
	defer q.mu.RUnlock()

	startIdx := q.current
	if startIdx < 0 {
		startIdx = 0
	}

	if startIdx >= len(q.items) {
		return nil
	}

	result := make([]*QueueItem, len(q.items)-startIdx)
	copy(result, q.items[startIdx:])
	return result
}

// Clear removes all items from the queue.
func (q *Queue) Clear() {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.items = make([]*QueueItem, 0)
	q.current = -1
}

// Remove removes an item at the given position.
// Returns an error if the position is invalid or the item is currently playing.
func (q *Queue) Remove(position int) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if position < 0 || position >= len(q.items) {
		return ErrPositionOutOfRange
	}

	// Don't allow removing currently playing track
	if position == q.current {
		return errors.New("cannot remove currently playing track, use skip instead")
	}

	// Remove the item
	q.items = append(q.items[:position], q.items[position+1:]...)

	// Adjust current index if necessary
	if position < q.current {
		q.current--
	}

	// Reindex remaining items
	for i := position; i < len(q.items); i++ {
		q.items[i].Position = i
	}

	return nil
}

// Get returns the item at the given position.
func (q *Queue) Get(position int) (*QueueItem, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if position < 0 || position >= len(q.items) {
		return nil, ErrPositionOutOfRange
	}

	return q.items[position], nil
}

// Len returns the total number of items in the queue.
func (q *Queue) Len() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.items)
}

// RemainingLen returns the number of items remaining (including current).
func (q *Queue) RemainingLen() int {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if q.current < 0 {
		return len(q.items)
	}

	remaining := len(q.items) - q.current
	if remaining < 0 {
		return 0
	}
	return remaining
}

// CurrentIndex returns the index of the currently playing item.
// Returns -1 if nothing is playing.
func (q *Queue) CurrentIndex() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.current
}

// SetPlaying marks a specific position as currently playing.
// This is used to sync with Liquidsoap's actual state.
func (q *Queue) SetPlaying(position int) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if position < 0 || position >= len(q.items) {
		return ErrPositionOutOfRange
	}

	// Clear previous playing state
	for i := range q.items {
		q.items[i].IsPlaying = false
	}

	q.current = position
	q.items[position].IsPlaying = true

	return nil
}

// FindByRequestID finds a queue item by its Liquidsoap request ID.
func (q *Queue) FindByRequestID(requestID string) *QueueItem {
	q.mu.RLock()
	defer q.mu.RUnlock()

	for _, item := range q.items {
		if item.RequestID == requestID {
			return item
		}
	}
	return nil
}

// FindByTrackID finds a queue item by its track ID.
func (q *Queue) FindByTrackID(trackID string) *QueueItem {
	q.mu.RLock()
	defer q.mu.RUnlock()

	for _, item := range q.items {
		if item.Track != nil && item.Track.ID == trackID {
			return item
		}
	}
	return nil
}
