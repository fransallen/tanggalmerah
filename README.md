# Tanggal Merah API

A lightweight, **zero-dependency** REST API for Indonesian public holidays and collective leave days, built with Go standard library only.

> **Tanggal merah** (lit. _red date_) is the Indonesian term for a public holiday — the days marked in red on a calendar.

## Features

- Query holidays by year, year + month, or specific date
- Two types: **`holiday`** (hari libur nasional) and **`leave`** (cuti bersama)
- Date check endpoint returns `is_holiday`, `is_leave`, and `is_off_day`
- `year` defaults to the current year when omitted
- In-memory cache per year — fast after first load
- Graceful shutdown, structured JSON logging, CORS-ready
- Multi-stage Docker build producing a minimal `scratch` image
- Full test suite (`go test ./...`)
- Zero external dependencies — Go 1.22 stdlib only

## Used by

Showcasing who is using the Tanggal Merah API:

- [TanggalanJawa.com](https://tanggalanjawa.com)

---

## Quick Start

```bash
git clone https://github.com/fransallen/tanggalmerah.git
cd tanggalmerah

air
# http://localhost:8080
```

### Docker

```bash
docker build -t tanggalmerah .
docker run -p 8080:8080 tanggalmerah
```

---

## Configuration

| Variable   | Default | Description                            |
| ---------- | ------- | -------------------------------------- |
| `PORT`     | `8080`  | Port the HTTP server listens on        |
| `DATA_DIR` | `data`  | Directory containing `YEAR.json` files |

---

## API Reference

**Base URL:** `http://localhost:8080`

### Response Envelope

All successful responses:

```json
{
  "success": true,
  "data": "...",
  "meta": {
    "total": 25,
    "total_holidays": 17,
    "total_leave": 8,
    "year": 2026
  }
}
```

All error responses:

```json
{ "success": false, "error": "Human-readable message", "code": "ERROR_CODE" }
```

### Holiday Types

| `type`    | Description                                           |
| --------- | ----------------------------------------------------- |
| `holiday` | Hari Libur Nasional — mandatory day off               |
| `leave`   | Cuti Bersama — government-designated bridge/leave day |

---

### `GET /`

Redirects to the web interface for the current year.

---

### `GET /health`

```bash
curl http://localhost:8080/health
```

```json
{ "status": "ok", "version": "1.0.0", "time": "2026-01-01T00:00:00Z" }
```

---

### `GET /api/years`

List all years that have available data.

```bash
curl http://localhost:8080/api/years
```

```json
{
  "success": true,
  "data": [2023, 2024, 2025, 2026, 2027],
  "meta": { "total": 5 }
}
```

---

### `GET /api/holidays`

Query holidays by year and optional month, with type filters.

| Query param | Values                        | Required | Default      |
| ----------- | ----------------------------- | -------- | ------------ |
| `year`      | `1900`–`2100`                 | No       | current year |
| `month`     | `1`–`12`                      | No       | —            |
| `type`      | `holiday` \| `leave` \| `all` | No       | `all`        |

```bash
# All holidays for the current year
curl "http://localhost:8080/api/holidays"

# All 2026 holidays
curl "http://localhost:8080/api/holidays?year=2026"

# March only
curl "http://localhost:8080/api/holidays?year=2026&month=3"

# Only collective leave days in March
curl "http://localhost:8080/api/holidays?year=2026&month=3&type=leave"
```

**Example response:**

```json
{
  "success": true,
  "data": [
    {
      "date": "2026-03-18",
      "day": "Rabu",
      "name": "Hari Suci Nyepi Tahun Baru Saka 1948",
      "type": "leave"
    }
  ],
  "meta": {
    "total": 1,
    "total_holidays": 0,
    "total_leave": 1,
    "year": 2026,
    "month": 3,
    "source": "Government of the Republic of Indonesia"
  }
}
```

**Error codes**

| Code             | HTTP | When                                    |
| ---------------- | ---- | --------------------------------------- |
| `INVALID_YEAR`   | 400  | Year is not a valid integer (1900–2100) |
| `INVALID_MONTH`  | 400  | Month is not 1–12                       |
| `INVALID_TYPE`   | 400  | Type is not one of the accepted values  |
| `YEAR_NOT_FOUND` | 404  | No data file exists for that year       |

---

### `GET /api/check?date=YYYY-MM-DD`

Check whether a specific date is a public holiday, collective leave day, or a regular workday.

```bash
curl "http://localhost:8080/api/check?date=2026-03-21"
```

```json
{
  "success": true,
  "data": {
    "date": "2026-03-21",
    "is_holiday": true,
    "is_leave": false,
    "is_off_day": true,
    "holidays": [
      {
        "date": "2026-03-21",
        "day": "Sabtu",
        "name": "Hari Raya Idul Fitri 1447 Hijriyah",
        "type": "holiday"
      }
    ]
  }
}
```

```bash
curl "http://localhost:8080/api/check?date=2026-03-20"
```

```json
{
  "success": true,
  "data": {
    "date": "2026-03-20",
    "is_holiday": false,
    "is_leave": true,
    "is_off_day": true,
    "holidays": [
      {
        "date": "2026-03-20",
        "day": "Jumat",
        "name": "Hari Raya Idul Fitri 1447 Hijriyah",
        "type": "leave"
      }
    ]
  }
}
```

```bash
curl "http://localhost:8080/api/check?date=2026-03-25"
```

```json
{
  "success": true,
  "data": {
    "date": "2026-03-25",
    "is_holiday": false,
    "is_leave": false,
    "is_off_day": false
  }
}
```

> If no data file exists for the year of the requested date, the endpoint returns `200` with all flags `false` rather than `404`.

**Error codes**

| Code           | HTTP | When                                 |
| -------------- | ---- | ------------------------------------ |
| `MISSING_DATE` | 400  | `date` query param is absent         |
| `INVALID_DATE` | 400  | `date` is not in `YYYY-MM-DD` format |

---

## Data Format

Each year's data lives in `data/YYYY.json`:

```json
{
  "year": 2026,
  "holidays": [
    {
      "date": "2026-01-01",
      "day": "Kamis",
      "name": "Tahun Baru 2026 Masehi",
      "type": "holiday"
    },
    {
      "date": "2026-12-24",
      "day": "Kamis",
      "name": "Hari Raya Natal",
      "type": "leave"
    }
  ]
}
```

To add a new year: drop a `YYYY.json` file into the `data/` folder and restart the server.

---

## Project Structure

```
tanggalmerah/
├── cmd/server/
│   ├── main.go                       # Entry point, router, middleware, graceful shutdown
│   └── main_test.go                  # Tests for middleware and helpers
├── internal/
│   ├── handler/
│   │   ├── handler.go                # HTTP handlers
│   │   └── handler_test.go           # Tests
│   ├── model/
│   │   └── model.go                  # Domain types and response envelopes
│   └── repository/
│       ├── repository.go             # JSON loading, in-memory cache, filter helpers
│       └── repository_test.go        # Tests
├── data/
│   ├── 2023.json
│   ├── 2024.json
│   ├── 2025.json
│   └── 2026.json
├── .air.toml                         # Live-reload config (air)
├── Dockerfile
├── go.mod
├── package.json
└── README.md
```

---

## Running Tests

```bash
go test ./...
go test ./... -v -race
```

---

## Contributing

Pull requests are welcome! The most valuable contribution is adding official data for years not yet covered.

1. Fork the repository
2. Create `data/YYYY.json` following the schema above
3. Verify the data against the official government decree (_Surat Keputusan Bersama_)
4. Include both `holiday` and `leave` entries where applicable
5. Open a pull request

---

## License

MIT
