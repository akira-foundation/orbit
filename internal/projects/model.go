package projects

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
	RuntimeKindPython  RuntimeKind = "python"
)

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
	PythonVersion     string        `json:"pythonVersion"`
	Status            Status        `json:"status"`
	Secure            bool          `json:"secure"`
	InstalledHash     string        `json:"installedHash"`
	CreatedAt         string        `json:"createdAt"`
	UpdatedAt         string        `json:"updatedAt"`
	Scripts           []Script      `json:"scripts,omitempty"`
	Processes         []ProcessSpec `json:"processes,omitempty"`
}

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

type Group struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Order      int      `json:"order"`
	CreatedAt  string   `json:"createdAt"`
	ProjectIDs []string `json:"projectIds"`
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
