package services

type InstalledBinary struct {
	Engine      string `json:"engine"`
	Version     string `json:"version"`
	Path        string `json:"path"`
	Hash        string `json:"hash"`
	InstalledAt string `json:"installedAt"`
}
