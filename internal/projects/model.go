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

type RuntimeKind string

const (
	RuntimeKindCommand RuntimeKind = "command"
	RuntimeKindPHPFPM  RuntimeKind = "php-fpm"
)

// Project is the domain model exposed via the Wails RPC and stored in SQLite.
// Timestamps are ISO-8601 strings so the Wails binding generator emits `string`
// instead of `any` (which happens with time.Time).
type Project struct {
	ID                string        `json:"id"`
	Name              string        `json:"name"`
	Path              string        `json:"path"`
	Slug              string        `json:"slug"`
	LocalDomain       string        `json:"localDomain"`
	DetectedFramework string        `json:"detectedFramework"`
	PackageManager    string        `json:"packageManager"`
	DevCommand        string        `json:"devCommand"`
	DevPort           int           `json:"devPort"`
	NodeVersion       string        `json:"nodeVersion"`
	RuntimeKind       RuntimeKind   `json:"runtimeKind"`
	PHPVersion        string        `json:"phpVersion"`
	Status            Status        `json:"status"`
	Secure            bool          `json:"secure"`
	InstalledHash     string        `json:"installedHash"`
	CreatedAt         string        `json:"createdAt"`
	UpdatedAt         string        `json:"updatedAt"`
	Scripts           []Script      `json:"scripts,omitempty"`
	Processes         []ProcessSpec `json:"processes,omitempty"`
}

// Script is a named npm/yarn/pnpm/bun script detected in package.json.
type Script struct {
	ID        string `json:"id"`
	ProjectID string `json:"projectId"`
	Name      string `json:"name"`
	Command   string `json:"command"`
	CreatedAt string `json:"createdAt"`
}

type ProcessRole string

const (
	ProcessRoleBackend  ProcessRole = "backend"
	ProcessRoleFrontend ProcessRole = "frontend"
)

// ProcessSpec is one runnable process belonging to a project (e.g. the
// php-fpm backend and a companion Vite dev server for Inertia/monorepo apps).
type ProcessSpec struct {
	ID         string      `json:"id"`
	ProjectID  string      `json:"projectId"`
	Role       ProcessRole `json:"role"`
	Kind       RuntimeKind `json:"kind"`
	WorkDir    string      `json:"workDir"`
	Command    string      `json:"command"`
	Port       int         `json:"port"`
	PHPVersion string      `json:"phpVersion"`
	Order      int         `json:"order"`
	CreatedAt  string      `json:"createdAt"`
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
