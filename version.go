package game

var (
	// Commit (git rev-parse --short HEAD).
	Commit = "dev"
	// BuildDate (RFC3339; BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)).
	BuildDate = "2026-07-02T15:47:00Z"
)

// Version ...
type Version struct {
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
}
