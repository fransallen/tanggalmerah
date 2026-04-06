package repository_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fransallen/tanggalmerah/internal/model"
	"github.com/fransallen/tanggalmerah/internal/repository"
)

const yearJSON = `{
  "year": 2026,
  "holidays": [
    {"date":"2026-01-01","day":"Kamis","name":"New Year","type":"holiday"},
    {"date":"2026-03-18","day":"Rabu","name":"Nyepi leave","type":"leave"},
    {"date":"2026-03-19","day":"Kamis","name":"Nyepi","type":"holiday"},
    {"date":"2026-12-25","day":"Jumat","name":"Christmas","type":"holiday"}
  ]
}`

func newTestRepo(t *testing.T) (*repository.Repository, string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "2026.json"), []byte(yearJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	return repository.New(dir), dir
}

// --------------------------------------------------------------------------
// GetYear
// --------------------------------------------------------------------------

func TestGetYear_Found(t *testing.T) {
	repo, _ := newTestRepo(t)
	yd, err := repo.GetYear(2026)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if yd == nil {
		t.Fatal("want year data, got nil")
	}
	if yd.Year != 2026 {
		t.Errorf("want year=2026, got %d", yd.Year)
	}
	if len(yd.Holidays) != 4 {
		t.Errorf("want 4 holidays, got %d", len(yd.Holidays))
	}
}

func TestGetYear_NotFound(t *testing.T) {
	repo, _ := newTestRepo(t)
	yd, err := repo.GetYear(1999)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if yd != nil {
		t.Errorf("want nil for missing year, got %+v", yd)
	}
}

func TestGetYear_CacheHit(t *testing.T) {
	repo, _ := newTestRepo(t)
	yd1, err := repo.GetYear(2026)
	if err != nil || yd1 == nil {
		t.Fatalf("first call failed: %v", err)
	}
	yd2, err := repo.GetYear(2026)
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}
	if yd1 != yd2 {
		t.Error("want same pointer from cache, got different")
	}
}

func TestGetYear_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "2025.json"), []byte(`not json`), 0o644); err != nil {
		t.Fatal(err)
	}
	repo := repository.New(dir)
	_, err := repo.GetYear(2025)
	if err == nil {
		t.Error("want error for invalid JSON, got nil")
	}
}

// --------------------------------------------------------------------------
// AvailableYears
// --------------------------------------------------------------------------

func TestAvailableYears(t *testing.T) {
	repo, dir := newTestRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "2025.json"), []byte(`{"year":2025,"holidays":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	// Non-JSON file should be ignored.
	if err := os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("ignore"), 0o644); err != nil {
		t.Fatal(err)
	}
	years, err := repo.AvailableYears()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(years) != 2 {
		t.Errorf("want 2 years, got %d: %v", len(years), years)
	}
}

func TestAvailableYears_Empty(t *testing.T) {
	dir := t.TempDir()
	repo := repository.New(dir)
	years, err := repo.AvailableYears()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(years) != 0 {
		t.Errorf("want 0 years, got %v", years)
	}
}

func TestAvailableYears_SkipsDirectories(t *testing.T) {
	repo, dir := newTestRepo(t)
	if err := os.Mkdir(filepath.Join(dir, "subdir"), 0o755); err != nil {
		t.Fatal(err)
	}
	years, err := repo.AvailableYears()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(years) != 1 {
		t.Errorf("want 1 year (subdir ignored), got %d: %v", len(years), years)
	}
}

// --------------------------------------------------------------------------
// FilterByMonth
// --------------------------------------------------------------------------

func TestFilterByMonth(t *testing.T) {
	holidays := []model.Holiday{
		{Date: "2026-01-01", Type: model.TypePublicHoliday},
		{Date: "2026-03-18", Type: model.TypeCollectiveLeave},
		{Date: "2026-03-19", Type: model.TypePublicHoliday},
		{Date: "2026-12-25", Type: model.TypePublicHoliday},
	}
	got := repository.FilterByMonth(holidays, 3)
	if len(got) != 2 {
		t.Errorf("want 2 March holidays, got %d", len(got))
	}
	got = repository.FilterByMonth(holidays, 1)
	if len(got) != 1 {
		t.Errorf("want 1 January holiday, got %d", len(got))
	}
	got = repository.FilterByMonth(holidays, 12)
	if len(got) != 1 {
		t.Errorf("want 1 December holiday, got %d", len(got))
	}
}

func TestFilterByMonth_NoMatch(t *testing.T) {
	holidays := []model.Holiday{
		{Date: "2026-01-01", Type: model.TypePublicHoliday},
	}
	got := repository.FilterByMonth(holidays, 6)
	if len(got) != 0 {
		t.Errorf("want 0, got %d", len(got))
	}
}

func TestFilterByMonth_ShortDate(t *testing.T) {
	// Entries with date shorter than 8 chars should not match.
	holidays := []model.Holiday{
		{Date: "short"},
	}
	got := repository.FilterByMonth(holidays, 3)
	if len(got) != 0 {
		t.Errorf("want 0, got %d", len(got))
	}
}

// --------------------------------------------------------------------------
// FilterByType
// --------------------------------------------------------------------------

func TestFilterByType_PublicHoliday(t *testing.T) {
	holidays := []model.Holiday{
		{Date: "2026-01-01", Type: model.TypePublicHoliday},
		{Date: "2026-03-18", Type: model.TypeCollectiveLeave},
		{Date: "2026-03-19", Type: model.TypePublicHoliday},
	}
	got := repository.FilterByType(holidays, model.TypePublicHoliday)
	if len(got) != 2 {
		t.Errorf("want 2 public holidays, got %d", len(got))
	}
}

func TestFilterByType_CollectiveLeave(t *testing.T) {
	holidays := []model.Holiday{
		{Date: "2026-01-01", Type: model.TypePublicHoliday},
		{Date: "2026-03-18", Type: model.TypeCollectiveLeave},
	}
	got := repository.FilterByType(holidays, model.TypeCollectiveLeave)
	if len(got) != 1 {
		t.Errorf("want 1 collective leave, got %d", len(got))
	}
}

// --------------------------------------------------------------------------
// SplitCounts
// --------------------------------------------------------------------------

func TestSplitCounts(t *testing.T) {
	holidays := []model.Holiday{
		{Type: model.TypePublicHoliday},
		{Type: model.TypePublicHoliday},
		{Type: model.TypeCollectiveLeave},
	}
	ph, cl := repository.SplitCounts(holidays)
	if ph != 2 {
		t.Errorf("want 2 public holidays, got %d", ph)
	}
	if cl != 1 {
		t.Errorf("want 1 collective leave, got %d", cl)
	}
}

func TestSplitCounts_Empty(t *testing.T) {
	ph, cl := repository.SplitCounts(nil)
	if ph != 0 || cl != 0 {
		t.Errorf("want 0,0; got %d,%d", ph, cl)
	}
}

func TestGetYear_OpenError(t *testing.T) {
	dir := t.TempDir()
	// Create "2027.json" as a directory; os.Open succeeds but json.Decode fails
	// with an error that is NOT os.IsNotExist, hitting the decode error branch.
	if err := os.Mkdir(filepath.Join(dir, "2027.json"), 0o755); err != nil {
		t.Fatal(err)
	}
	repo := repository.New(dir)
	_, err := repo.GetYear(2027)
	if err == nil {
		t.Error("want error for undecodable JSON, got nil")
	}
}

func TestAvailableYears_ReadDirError(t *testing.T) {
	repo := repository.New("/nonexistent/path/that/does/not/exist")
	_, err := repo.AvailableYears()
	if err == nil {
		t.Error("want error for unreadable dir, got nil")
	}
}

func TestGetYear_PermissionDenied(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root, permission test not applicable")
	}
	dir := t.TempDir()
	// Create a subdirectory with no read permission; opening a file inside it
	// will fail with EACCES (not IsNotExist), hitting the open-error branch.
	restricted := filepath.Join(dir, "restricted")
	if err := os.Mkdir(restricted, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(restricted, 0o755) })

	repo := repository.New(restricted)
	_, err := repo.GetYear(2028)
	if err == nil {
		t.Error("want error for permission-denied open, got nil")
	}
}
