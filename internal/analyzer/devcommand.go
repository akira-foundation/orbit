package analyzer

import "strings"

// runCmd returns the correct "run script" invocation for the given PM.
func runCmd(pm, script string) string {
	switch pm {
	case "yarn":
		return "yarn " + script
	case "bun":
		return "bun run " + script
	default:
		return pm + " run " + script
	}
}

func suggestDevCommand(scripts map[string]string, pm string) string {
	preferred := []string{"dev", "start:dev", "develop", "serve", "start"}
	for _, name := range preferred {
		if _, ok := scripts[name]; ok {
			return runCmd(pm, name)
		}
	}
	return runCmd(pm, "dev")
}

var defaultPorts = map[string]int{
	"nextjs":       3000,
	"nuxt":         3000,
	"remix":        3000,
	"sveltekit":    5173,
	"solid-start":  3000,
	"astro":        4321,
	"gatsby":       8000,
	"expo":         8081,
	"angular":      4200,
	"svelte":       5173,
	"solid":        3000,
	"react-native": 8081,
	"cra":          3000,
	"vite":         5173,
	"nestjs":       3000,
	"fastify":      3000,
	"express":      3000,
	"koa":          3000,
	"hapi":         3000,
	"strapi":       1337,
	"payload":      3000,
	"electron":     0,
}

func suggestPort(framework string, scripts map[string]string, devCmd string) int {
	if port := extractPortFromScripts(scripts); port > 0 {
		return port
	}
	if p, ok := defaultPorts[framework]; ok {
		return p
	}
	return 0
}

func extractPortFromScripts(scripts map[string]string) int {
	preferred := []string{"dev", "start:dev", "develop", "serve", "start"}
	for _, name := range preferred {
		cmd, ok := scripts[name]
		if !ok {
			continue
		}
		for _, flag := range []string{"--port ", "--port="} {
			if i := strings.Index(cmd, flag); i >= 0 {
				rest := cmd[i+len(flag):]
				rest = strings.TrimSpace(rest)
				var port int
				for _, ch := range rest {
					if ch >= '0' && ch <= '9' {
						port = port*10 + int(ch-'0')
					} else {
						break
					}
				}
				if port > 0 {
					return port
				}
			}
		}
	}
	return 0
}
