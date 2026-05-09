package projects

import "time"

type Status string

const (
	StatusStopped   Status = "stopped"
	StatusStarting  Status = "starting"
	StatusRunning   Status = "running"
	StatusIdle      Status = "idle"
	StatusSuspended Status = "suspended"
	StatusError     Status = "error"
)

type Project struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	Path              string    `json:"path"`
	Slug              string    `json:"slug"`
	LocalDomain       string    `json:"localDomain"`
	DetectedFramework string    `json:"detectedFramework"`
	PackageManager    string    `json:"packageManager"`
	DevCommand        string    `json:"devCommand"`
	DevPort           int       `json:"devPort"`
	Status            Status    `json:"status"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
	Scripts           []Script  `json:"scripts,omitempty"`
}

type Script struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"projectId"`
	Name      string    `json:"name"`
	Command   string    `json:"command"`
	CreatedAt time.Time `json:"createdAt"`
}
