package analyzer

import (
	"orbit-app/internal/semverlite"
	"orbit-app/internal/services"
)

func resolvePHPVersion(constraint string) string {
	available := services.PHPVersions()
	if len(available) == 0 {
		return ""
	}
	if constraint != "" {
		if resolved, ok := semverlite.Resolve(constraint, available); ok {
			return resolved
		}
	}
	return available[0]
}
