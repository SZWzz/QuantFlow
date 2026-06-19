package research

import (
	"context"
	"log/slog"

	"quantflow/internal/market/adapters"
)

// AnnouncementService provides stock exchange announcement data from 巨潮资讯网.
// Degrades gracefully when the adapter is nil or API calls fail.
type AnnouncementService struct {
	adapter *adapters.CninfoAdapter
}

// NewAnnouncementService creates a new AnnouncementService. adapter may be nil for mock mode.
func NewAnnouncementService(adapter *adapters.CninfoAdapter) *AnnouncementService {
	return &AnnouncementService{adapter: adapter}
}

// GetAnnouncements returns recent stock exchange announcements for a symbol.
func (s *AnnouncementService) GetAnnouncements(ctx context.Context, symbol string, pageSize int) ([]adapters.Announcement, error) {
	if s.adapter == nil {
		return nil, nil
	}
	data, err := s.adapter.FetchAnnouncements(ctx, symbol, pageSize)
	if err != nil {
		slog.Warn("announcement: fetch failed", "symbol", symbol, "error", err)
		return nil, nil
	}
	return data, nil
}
