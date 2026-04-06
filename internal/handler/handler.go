package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/fransallen/tanggalmerah/internal/model"
	"github.com/fransallen/tanggalmerah/internal/repository"
)

// Handler holds shared dependencies for all HTTP handlers.
type Handler struct {
	repo    *repository.Repository
	version string
}

// New creates a Handler.
func New(repo *repository.Repository, version string) *Handler {
	return &Handler{repo: repo, version: version}
}

// --------------------------------------------------------------------------
// Response helpers
// --------------------------------------------------------------------------

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, model.ErrorResponse{
		Success: false,
		Error:   message,
		Code:    code,
	})
}

func success(data interface{}, meta *model.Meta) model.APIResponse {
	return model.APIResponse{Success: true, Data: data, Meta: meta}
}

// --------------------------------------------------------------------------
// GET /health
// --------------------------------------------------------------------------

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"version": h.version,
		"time":    time.Now().UTC().Format(time.RFC3339),
	})
}

// --------------------------------------------------------------------------
// GET /api/years
// --------------------------------------------------------------------------

func (h *Handler) ListYears(w http.ResponseWriter, r *http.Request) {
	years, err := h.repo.AvailableYears()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list available years")
		return
	}
	writeJSON(w, http.StatusOK, success(years, &model.Meta{Total: len(years)}))
}

// --------------------------------------------------------------------------
// GET /api/holidays
//
// Query params:
//   ?year=YYYY         (required)
//   ?month=1..12       (optional)
//   ?type=public_holiday|collective_leave|all  (default: all)
// --------------------------------------------------------------------------

func (h *Handler) ListHolidays(w http.ResponseWriter, r *http.Request) {
	yearStr := r.URL.Query().Get("year")
	if yearStr == "" {
		writeError(w, http.StatusBadRequest, "MISSING_YEAR", "Query parameter 'year' is required (YYYY)")
		return
	}

	year, ok := parseYear(w, yearStr)
	if !ok {
		return
	}

	yd, err := h.repo.GetYear(year)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to load holiday data")
		return
	}
	if yd == nil {
		writeError(w, http.StatusNotFound, "YEAR_NOT_FOUND", "No holiday data available for the requested year")
		return
	}

	holidays := yd.Holidays

	var month int
	if m := r.URL.Query().Get("month"); m != "" {
		var ok2 bool
		month, ok2 = parseMonth(w, m)
		if !ok2 {
			return
		}
		holidays = repository.FilterByMonth(holidays, month)
	}

	holidays, ok = applyTypeFilter(w, r, holidays)
	if !ok {
		return
	}

	ph, cl := repository.SplitCounts(holidays)
	meta := &model.Meta{
		Total:                len(holidays),
		TotalPublicHolidays:  ph,
		TotalCollectiveLeave: cl,
		Year:                 year,
	}
	if month > 0 {
		meta.Month = month
	}

	writeJSON(w, http.StatusOK, success(holidays, meta))
}

// --------------------------------------------------------------------------
// GET /api/holidays/check?date=YYYY-MM-DD
// --------------------------------------------------------------------------

func (h *Handler) CheckDate(w http.ResponseWriter, r *http.Request) {
	raw := r.URL.Query().Get("date")
	if raw == "" {
		writeError(w, http.StatusBadRequest, "MISSING_DATE", "Query parameter 'date' is required (YYYY-MM-DD)")
		return
	}

	t, err := time.Parse("2006-01-02", raw)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_DATE", "date must be in YYYY-MM-DD format")
		return
	}

	yd, err := h.repo.GetYear(t.Year())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to load holiday data")
		return
	}

	resp := model.DateCheckResponse{Date: raw}
	if yd != nil {
		for _, hol := range yd.Holidays {
			if hol.Date == raw {
				resp.Holidays = append(resp.Holidays, hol)
				switch hol.Type {
				case model.TypePublicHoliday:
					resp.IsPublicHoliday = true
				case model.TypeCollectiveLeave:
					resp.IsCollectiveLeave = true
				}
			}
		}
	}
	resp.IsOffDay = resp.IsPublicHoliday || resp.IsCollectiveLeave

	writeJSON(w, http.StatusOK, success(resp, nil))
}

// --------------------------------------------------------------------------
// Helpers
// --------------------------------------------------------------------------

func parseYear(w http.ResponseWriter, s string) (int, bool) {
	year, err := strconv.Atoi(s)
	if err != nil || year < 1900 || year > 2100 {
		writeError(w, http.StatusBadRequest, "INVALID_YEAR", "year must be a 4-digit integer between 1900 and 2100")
		return 0, false
	}
	return year, true
}

func parseMonth(w http.ResponseWriter, s string) (int, bool) {
	month, err := strconv.Atoi(s)
	if err != nil || month < 1 || month > 12 {
		writeError(w, http.StatusBadRequest, "INVALID_MONTH", "month must be an integer between 1 and 12")
		return 0, false
	}
	return month, true
}

// applyTypeFilter reads the optional ?type= query param and filters accordingly.
// Accepted: "public_holiday", "collective_leave", "all" (default).
func applyTypeFilter(w http.ResponseWriter, r *http.Request, holidays []model.Holiday) ([]model.Holiday, bool) {
	switch r.URL.Query().Get("type") {
	case "", "all":
		return holidays, true
	case string(model.TypePublicHoliday):
		return repository.FilterByType(holidays, model.TypePublicHoliday), true
	case string(model.TypeCollectiveLeave):
		return repository.FilterByType(holidays, model.TypeCollectiveLeave), true
	default:
		writeError(w, http.StatusBadRequest, "INVALID_TYPE",
			"type must be one of: holiday, leave, all")
		return nil, false
	}
}
