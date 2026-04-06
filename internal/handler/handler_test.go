package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/fransallen/tanggalmerah/internal/handler"
	"github.com/fransallen/tanggalmerah/internal/model"
	"github.com/fransallen/tanggalmerah/internal/repository"
)

// --------------------------------------------------------------------------
// Test fixture
// --------------------------------------------------------------------------

const fixtureJSON = `{
  "year": 2026,
  "holidays": [
    {"date":"2026-01-01","day":"Kamis","name":"New Year's Day","type":"holiday"},
    {"date":"2026-03-19","day":"Kamis","name":"Nyepi","type":"holiday"},
    {"date":"2026-03-18","day":"Rabu","name":"Nyepi","type":"leave"},
    {"date":"2026-03-21","day":"Sabtu","name":"Eid al-Fitr","type":"holiday"},
    {"date":"2026-03-22","day":"Minggu","name":"Eid al-Fitr","type":"holiday"},
    {"date":"2026-03-20","day":"Jumat","name":"Eid al-Fitr","type":"leave"},
    {"date":"2026-03-23","day":"Senin","name":"Eid al-Fitr","type":"leave"},
    {"date":"2026-12-25","day":"Jumat","name":"Christmas Day","type":"holiday"},
    {"date":"2026-12-24","day":"Kamis","name":"Christmas Day","type":"leave"}
  ]
}`

func newTestHandler(t *testing.T) *handler.Handler {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "2026.json"), []byte(fixtureJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	return handler.New(repository.New(dir), "test")
}

func decode(t *testing.T, body []byte) model.APIResponse {
	t.Helper()
	var resp model.APIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("unmarshal: %v\nbody: %s", err, body)
	}
	return resp
}

func dataBytes(t *testing.T, resp model.APIResponse) []byte {
	t.Helper()
	b, err := json.Marshal(resp.Data)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func get(t *testing.T, h http.Handler, target string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func newMux(h *handler.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.Health)
	mux.HandleFunc("GET /api/years", h.ListYears)
	mux.HandleFunc("GET /api/check", h.CheckDate)
	mux.HandleFunc("GET /api/holidays", h.ListHolidays)
	return mux
}

// --------------------------------------------------------------------------
// Health
// --------------------------------------------------------------------------

func TestHealth(t *testing.T) {
	rr := get(t, newMux(newTestHandler(t)), "/health")
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "ok" {
		t.Errorf("want status=ok, got %q", body["status"])
	}
}

// --------------------------------------------------------------------------
// ListYears
// --------------------------------------------------------------------------

func TestListYears(t *testing.T) {
	rr := get(t, newMux(newTestHandler(t)), "/api/years")
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
	var years []int
	if err := json.Unmarshal(dataBytes(t, decode(t, rr.Body.Bytes())), &years); err != nil {
		t.Fatal(err)
	}
	if len(years) != 1 || years[0] != 2026 {
		t.Errorf("want [2026], got %v", years)
	}
}

// --------------------------------------------------------------------------
// ListHolidays
// --------------------------------------------------------------------------

func TestListHolidays_All(t *testing.T) {
	rr := get(t, newMux(newTestHandler(t)), "/api/holidays?year=2026")
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
	var holidays []model.Holiday
	if err := json.Unmarshal(dataBytes(t, decode(t, rr.Body.Bytes())), &holidays); err != nil {
		t.Fatal(err)
	}
	if len(holidays) != 9 {
		t.Errorf("want 9 entries, got %d", len(holidays))
	}
}

func TestListHolidays_MonthFilter(t *testing.T) {
	rr := get(t, newMux(newTestHandler(t)), "/api/holidays?year=2026&month=3")
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
	var holidays []model.Holiday
	if err := json.Unmarshal(dataBytes(t, decode(t, rr.Body.Bytes())), &holidays); err != nil {
		t.Fatal(err)
	}
	// March: Nyepi (1 public + 1 collective) + Eid (2 public + 2 collective) = 6
	if len(holidays) != 6 {
		t.Errorf("want 6 March entries, got %d", len(holidays))
	}
}

func TestListHolidays_TypePublicHoliday(t *testing.T) {
	rr := get(t, newMux(newTestHandler(t)), "/api/holidays?year=2026&type=holiday")
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
	var holidays []model.Holiday
	if err := json.Unmarshal(dataBytes(t, decode(t, rr.Body.Bytes())), &holidays); err != nil {
		t.Fatal(err)
	}
	for _, hol := range holidays {
		if hol.Type != model.TypePublicHoliday {
			t.Errorf("expected only holiday, got %s on %s", hol.Type, hol.Date)
		}
	}
}

func TestListHolidays_TypeCollectiveLeave(t *testing.T) {
	rr := get(t, newMux(newTestHandler(t)), "/api/holidays?year=2026&type=leave")
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
	var holidays []model.Holiday
	if err := json.Unmarshal(dataBytes(t, decode(t, rr.Body.Bytes())), &holidays); err != nil {
		t.Fatal(err)
	}
	// fixture has 4 leave entries
	if len(holidays) != 4 {
		t.Errorf("want 4 leave entries, got %d", len(holidays))
	}
	for _, hol := range holidays {
		if hol.Type != model.TypeCollectiveLeave {
			t.Errorf("expected only leave, got %s on %s", hol.Type, hol.Date)
		}
	}
}

func TestListHolidays_MissingYear(t *testing.T) {
	rr := get(t, newMux(newTestHandler(t)), "/api/holidays")
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rr.Code)
	}
}

func TestListHolidays_InvalidYear(t *testing.T) {
	rr := get(t, newMux(newTestHandler(t)), "/api/holidays?year=abc")
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rr.Code)
	}
}

func TestListHolidays_InvalidMonth(t *testing.T) {
	rr := get(t, newMux(newTestHandler(t)), "/api/holidays?year=2026&month=13")
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rr.Code)
	}
}

func TestListHolidays_InvalidType(t *testing.T) {
	rr := get(t, newMux(newTestHandler(t)), "/api/holidays?year=2026&type=weekend")
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rr.Code)
	}
}

func TestListHolidays_YearNotFound(t *testing.T) {
	rr := get(t, newMux(newTestHandler(t)), "/api/holidays?year=2000")
	if rr.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", rr.Code)
	}
}

func TestListHolidays_WithMonthPublicHolidayOnly(t *testing.T) {
	rr := get(t, newMux(newTestHandler(t)), "/api/holidays?year=2026&month=3&type=holiday")
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
	var holidays []model.Holiday
	if err := json.Unmarshal(dataBytes(t, decode(t, rr.Body.Bytes())), &holidays); err != nil {
		t.Fatal(err)
	}
	// March public: Nyepi(1) + Eid(2) = 3
	if len(holidays) != 3 {
		t.Errorf("want 3 holiday entries in March, got %d", len(holidays))
	}
}

func TestListHolidays_WithMonthCollectiveLeaveOnly(t *testing.T) {
	rr := get(t, newMux(newTestHandler(t)), "/api/holidays?year=2026&month=3&type=leave")
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
	var holidays []model.Holiday
	if err := json.Unmarshal(dataBytes(t, decode(t, rr.Body.Bytes())), &holidays); err != nil {
		t.Fatal(err)
	}
	// March leave: Nyepi bridge(1) + Eid bridges(2) = 3
	if len(holidays) != 3 {
		t.Errorf("want 3 leave entries in March, got %d", len(holidays))
	}
}

func TestListHolidays_InvalidMonthFormat(t *testing.T) {
	rr := get(t, newMux(newTestHandler(t)), "/api/holidays?year=2026&month=0")
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rr.Code)
	}
}

// --------------------------------------------------------------------------
// CheckDate
// --------------------------------------------------------------------------

func TestCheckDate_PublicHoliday(t *testing.T) {
	rr := get(t, newMux(newTestHandler(t)), "/api/check?date=2026-01-01")
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
	var check model.DateCheckResponse
	if err := json.Unmarshal(dataBytes(t, decode(t, rr.Body.Bytes())), &check); err != nil {
		t.Fatal(err)
	}
	if !check.IsPublicHoliday {
		t.Error("want is_holiday=true")
	}
	if check.IsCollectiveLeave {
		t.Error("want is_leave=false")
	}
	if !check.IsOffDay {
		t.Error("want is_off_day=true")
	}
}

func TestCheckDate_CollectiveLeave(t *testing.T) {
	rr := get(t, newMux(newTestHandler(t)), "/api/check?date=2026-03-20")
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
	var check model.DateCheckResponse
	if err := json.Unmarshal(dataBytes(t, decode(t, rr.Body.Bytes())), &check); err != nil {
		t.Fatal(err)
	}
	if check.IsPublicHoliday {
		t.Error("want is_holiday=false")
	}
	if !check.IsCollectiveLeave {
		t.Error("want is_leave=true")
	}
	if !check.IsOffDay {
		t.Error("want is_off_day=true")
	}
}

func TestCheckDate_NormalWorkday(t *testing.T) {
	rr := get(t, newMux(newTestHandler(t)), "/api/check?date=2026-03-25")
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
	var check model.DateCheckResponse
	if err := json.Unmarshal(dataBytes(t, decode(t, rr.Body.Bytes())), &check); err != nil {
		t.Fatal(err)
	}
	if check.IsPublicHoliday || check.IsCollectiveLeave || check.IsOffDay {
		t.Error("want all flags=false for a normal workday")
	}
	if len(check.Holidays) != 0 {
		t.Errorf("want empty holidays slice, got %d", len(check.Holidays))
	}
}

func TestCheckDate_MissingDate(t *testing.T) {
	rr := get(t, newMux(newTestHandler(t)), "/api/check")
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rr.Code)
	}
}

func TestCheckDate_InvalidDate(t *testing.T) {
	rr := get(t, newMux(newTestHandler(t)), "/api/check?date=not-a-date")
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rr.Code)
	}
}

func TestCheckDate_UnknownYear(t *testing.T) {
	rr := get(t, newMux(newTestHandler(t)), "/api/check?date=2000-01-01")
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
	var check model.DateCheckResponse
	if err := json.Unmarshal(dataBytes(t, decode(t, rr.Body.Bytes())), &check); err != nil {
		t.Fatal(err)
	}
	if check.IsOffDay {
		t.Error("want is_off_day=false for unknown year")
	}
}

// --------------------------------------------------------------------------
// Meta counts
// --------------------------------------------------------------------------

func TestMeta_SplitCounts(t *testing.T) {
	rr := get(t, newMux(newTestHandler(t)), "/api/holidays?year=2026")
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
	var resp struct {
		Meta model.Meta `json:"meta"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	// fixture: 5 holiday, 4 leave
	if resp.Meta.TotalPublicHolidays != 5 {
		t.Errorf("want total_holidays=5, got %d", resp.Meta.TotalPublicHolidays)
	}
	if resp.Meta.TotalCollectiveLeave != 4 {
		t.Errorf("want total_leave=4, got %d", resp.Meta.TotalCollectiveLeave)
	}
}
