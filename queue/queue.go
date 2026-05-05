package queue

import (
	_ "fmt"
	"sync"
	
	"github.com/sebatt90/discord-soundboard/models"
)

type Queue struct {
	mu sync.Mutex
	tracks []models.Track
}

func (q *Queue) Enqueue(track models.Track) {
	q.mu.Lock()
	q.tracks = append(q.tracks, track)
	q.mu.Unlock()
}

func (q *Queue) Dequeue() (track *models.Track) {
	q.mu.Lock()
	if len(q.tracks)==0 {
		return nil
	}
	var t *models.Track = &q.tracks[0]
	q.tracks = q.tracks[1:]
	q.mu.Unlock()
	return t
}

func (q *Queue) IsEmpty() (bool) {
	return len(q.tracks) == 0
}

func (q *Queue) Clear() {
	q.mu.Lock()
	q.tracks = nil
	q.mu.Unlock()
}




