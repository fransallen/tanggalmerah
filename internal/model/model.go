package model

// HolidayType classifies each entry in a year's data file.
//
//   - "holiday" – National public holiday (mandatory day off)
//   - "leave"   – Collective leave / bridge day (cuti bersama)
type HolidayType string

const (
	TypePublicHoliday   HolidayType = "holiday"
	TypeCollectiveLeave HolidayType = "leave"
)

// Holiday represents a single entry in a YEAR.json data file.
type Holiday struct {
	Date string      `json:"date"` // ISO 8601: YYYY-MM-DD
	Day  string      `json:"day"`  // Indonesian day name, e.g. "Senin"
	Name string      `json:"name"` // Holiday name in Indonesian
	Type HolidayType `json:"type"` // "holiday" or "leave"
}

// YearData is the top-level structure stored in each YEAR.json file.
type YearData struct {
	Year     int       `json:"year"`
	Holidays []Holiday `json:"holidays"`
}

// APIResponse is the standard success envelope returned by every endpoint.
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Meta    *Meta       `json:"meta,omitempty"`
}

// Meta carries informational metadata attached to a successful response.
type Meta struct {
	Total                int    `json:"total,omitempty"`
	TotalPublicHolidays  int    `json:"total_holidays,omitempty"`
	TotalCollectiveLeave int    `json:"total_leave,omitempty"`
	Year                 int    `json:"year,omitempty"`
	Month                int    `json:"month,omitempty"`
	Source               string `json:"source,omitempty"`
}

// ErrorResponse is the standard error envelope.
type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	Code    string `json:"code"`
}

// DateCheckResponse is returned by GET /api/holidays/check.
type DateCheckResponse struct {
	Date              string    `json:"date"`
	IsPublicHoliday   bool      `json:"is_holiday"`
	IsCollectiveLeave bool      `json:"is_leave"`
	IsOffDay          bool      `json:"is_off_day"`
	Holidays          []Holiday `json:"holidays,omitempty"`
}
