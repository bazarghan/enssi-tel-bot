package word

import (
	"context"
	"github.com/bazarghan/enssi-tel-bot/internal/domain/word"
)

// CacheImageCommand holds the data needed to cache image file IDs.
type CacheImageCommand struct {
	CourseWordID   uint
	ImageFileID    string
	ImageDocFileID string
}

// CacheVoiceCommand holds the data needed to cache a voice file ID.
type CacheVoiceCommand struct {
	PronunciationID uint
	VoiceFileID     string
}

// CacheMediaHandler is the use case for caching media file IDs.
type CacheMediaHandler struct {
	repo word.Repository
}

// NewCacheMediaHandler creates a new handler.
func NewCacheMediaHandler(repo word.Repository) CacheMediaHandler {
	return CacheMediaHandler{repo: repo}
}

// HandleImage caches image file IDs.
func (h CacheMediaHandler) HandleImage(ctx context.Context, cmd CacheImageCommand) error {
	return h.repo.CacheImageFileIDs(ctx, cmd.CourseWordID, cmd.ImageFileID, cmd.ImageDocFileID)
}

// HandleVoice caches a voice file ID.
func (h CacheMediaHandler) HandleVoice(ctx context.Context, cmd CacheVoiceCommand) error {
	return h.repo.CacheVoiceFileID(ctx, cmd.PronunciationID, cmd.VoiceFileID)
}
