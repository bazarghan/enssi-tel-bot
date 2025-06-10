package word

import (
	"math"
	"time"
)

// InitialInterval is the first interval after learning a word.
const InitialInterval = 1 * 24 * time.Hour // 1 day

// MaxInterval is the maximum duration between reviews.
const MaxInterval = 128 * 24 * time.Hour // ~4 months

// CalculateNextReview determines the next review time and interval based on performance.
// This is a pure function containing the core SRS logic, modifying the passed-in struct.
func (sw *StudiedWord) CalculateNextReview(wasCorrect bool) {
	sw.LastReviewedAt = time.Now()

	if wasCorrect {
		// If correct, double the interval, up to the maximum.
		newIntervalDays := sw.ReviewIntervalDays * 2
		if newIntervalDays == 0 { // Handle the very first time a word is studied.
			newIntervalDays = uint(InitialInterval.Hours() / 24)
		}
		sw.ReviewIntervalDays = uint(math.Min(float64(newIntervalDays), float64(MaxInterval/(24*time.Hour))))
	} else {
		// If incorrect, reset the interval.
		sw.ReviewIntervalDays = uint(InitialInterval.Hours() / 24)
	}

	sw.NextReviewAt = sw.LastReviewedAt.Add(time.Duration(sw.ReviewIntervalDays) * 24 * time.Hour)
}
