package repository

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/fransallen/tanggalmerah/internal/model"
)

// Repository handles data access for holiday data files.
// Each YEAR.json file is loaded lazily and cached in memory.
type Repository struct {
	dataDir string
	cache   map[int]*model.YearData
	mu      sync.RWMutex
}

// New creates a new Repository pointing at the given data directory.
func New(dataDir string) *Repository {
	return &Repository{
		dataDir: dataDir,
		cache:   make(map[int]*model.YearData),
	}
}

// GetYear loads (and caches) holiday data for the given year.
// Returns (nil, nil) when no data file exists for that year.
func (r *Repository) GetYear(year int) (*model.YearData, error) {
	r.mu.RLock()
	if data, ok := r.cache[year]; ok {
		r.mu.RUnlock()
		return data, nil
	}
	r.mu.RUnlock()

	path := filepath.Join(r.dataDir, fmt.Sprintf("%d.json", year))
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	var yd model.YearData
	if err := json.NewDecoder(f).Decode(&yd); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}

	r.mu.Lock()
	r.cache[year] = &yd
	r.mu.Unlock()

	return &yd, nil
}

// AvailableYears scans the data directory and returns every year that has a
// corresponding JSON file, in ascending order.
func (r *Repository) AvailableYears() ([]int, error) {
	entries, err := os.ReadDir(r.dataDir)
	if err != nil {
		return nil, fmt.Errorf("read data dir: %w", err)
	}

	var years []int
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		var year int
		if _, err := fmt.Sscanf(e.Name(), "%d.json", &year); err == nil {
			years = append(years, year)
		}
	}
	return years, nil
}

// --------------------------------------------------------------------------
// Filter helpers
// --------------------------------------------------------------------------

// FilterByMonth returns only holidays whose date falls in the given month (1-12).
func FilterByMonth(holidays []model.Holiday, month int) []model.Holiday {
	prefix := monthPrefix(month)
	out := make([]model.Holiday, 0)
	for _, h := range holidays {
		if len(h.Date) >= 8 && h.Date[4:8] == prefix {
			out = append(out, h)
		}
	}
	return out
}

// FilterByType returns only holidays of the given HolidayType.
func FilterByType(holidays []model.Holiday, t model.HolidayType) []model.Holiday {
	out := make([]model.Holiday, 0)
	for _, h := range holidays {
		if h.Type == t {
			out = append(out, h)
		}
	}
	return out
}

// SplitCounts returns the count of public holidays and collective leave days.
func SplitCounts(holidays []model.Holiday) (publicHolidays, collectiveLeave int) {
	for _, h := range holidays {
		switch h.Type {
		case model.TypePublicHoliday:
			publicHolidays++
		case model.TypeCollectiveLeave:
			collectiveLeave++
		}
	}
	return
}

func monthPrefix(month int) string {
	if month < 10 {
		return fmt.Sprintf("-0%d-", month)
	}
	return fmt.Sprintf("-%d-", month)
}
