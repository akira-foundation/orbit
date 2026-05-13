package projects

// Status represents the runtime state of a project.
type Status string

const (
	StatusStopped   Status = "stopped"
	StatusStarting  Status = "starting"
	StatusRunning   Status = "running"
	StatusIdle      Status = "idle"
	StatusSuspended Status = "suspended"
	StatusError     Status = "error"
)

// Project is the domain model exposed via the Wails RPC and stored in SQLite.
// Timestamps are ISO-8601 strings so the Wails binding generator emits `string`
// instead of `any` (which happens with time.Time).
type Project struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	Path              string   `json:"path"`
	Slug              string   `json:"slug"`
	LocalDomain       string   `json:"localDomain"`
	DetectedFramework string   `json:"detectedFramework"`
	PackageManager    string   `json:"packageManager"`
	DevCommand        string   `json:"devCommand"`
	DevPort           int      `json:"devPort"`
	Status            Status   `json:"status"`
	InstalledHash     string   `json:"installedHash"`
	CreatedAt         string   `json:"createdAt"`
	UpdatedAt         string   `json:"updatedAt"`
	Scripts           []Script `json:"scripts,omitempty"`
}

// Script is a named npm/yarn/pnpm/bun script detected in package.json.
type Script struct {
	ID        string `json:"id"`
	ProjectID string `json:"projectId"`
	Name      string `json:"name"`
	Command   string `json:"command"`
	CreatedAt string `json:"createdAt"`
}

type Domain struct {
	ID         string `json:"id"`
	ProjectID  string `json:"projectId"`
	Domain     string `json:"domain"`
	TargetPort int    `json:"targetPort"`
	Enabled    bool   `json:"enabled"`
	CreatedAt  string `json:"createdAt"`
	UpdatedAt  string `json:"updatedAt"`
}
