package analyzer

import (
	"os"
	"path/filepath"
	"strings"

	"orbit-app/internal/semverlite"
	"orbit-app/internal/services"
)

var pythonMarkers = []string{
	"pyproject.toml",
	"requirements.txt",
	"Pipfile",
	"manage.py",
	"setup.py",
	"asgi.py",
	"wsgi.py",
}

func isPythonProject(dir string) bool {
	for _, name := range pythonMarkers {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return true
		}
	}
	return false
}

func analyzePython(dir string) *Analysis {
	deps := pythonDependencies(dir)
	framework := detectPythonFramework(dir, deps)
	pm := detectPythonPackageManager(dir)
	cmd, port := pythonDevCommand(dir, framework)

	return &Analysis{
		Name:           filepath.Base(dir),
		Path:           dir,
		Framework:      framework,
		PackageManager: pm,
		RuntimeKind:    "python",
		PythonVersion:  resolvePythonVersion(dir),
		DevCommand:     cmd,
		DevPort:        port,
		Scripts:        map[string]string{},
	}
}

func resolvePythonVersion(dir string) string {
	available := services.PythonVersions()
	if len(available) == 0 {
		return ""
	}
	constraint := readVersionFile(dir, ".python-version")
	if constraint == "" {
		constraint = pyprojectRequiresPython(dir)
	}
	if constraint != "" {
		if resolved, ok := semverlite.Resolve(constraint, available); ok {
			return resolved
		}
		if major := matchPythonMajor(constraint, available); major != "" {
			return major
		}
	}
	return available[0]
}

func matchPythonMajor(constraint string, available []string) string {
	digits := strings.TrimSpace(constraint)
	for _, v := range available {
		major := v[:strings.LastIndex(v, ".")]
		if strings.Contains(digits, major) {
			return v
		}
	}
	return ""
}

func pyprojectRequiresPython(dir string) string {
	raw, err := os.ReadFile(filepath.Join(dir, "pyproject.toml"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "requires-python") || strings.HasPrefix(line, "python ") || strings.HasPrefix(line, "python=") {
			if i := strings.Index(line, "="); i >= 0 {
				return strings.Trim(strings.TrimSpace(line[i+1:]), "\"'")
			}
		}
	}
	return ""
}

func pythonDependencies(dir string) map[string]bool {
	out := map[string]bool{}
	if raw, err := os.ReadFile(filepath.Join(dir, "requirements.txt")); err == nil {
		for _, line := range strings.Split(string(raw), "\n") {
			if name := requirementName(line); name != "" {
				out[name] = true
			}
		}
	}
	if raw, err := os.ReadFile(filepath.Join(dir, "pyproject.toml")); err == nil {
		lower := strings.ToLower(string(raw))
		for _, pkg := range []string{"django", "fastapi", "flask", "starlette", "uvicorn", "gunicorn"} {
			if strings.Contains(lower, pkg) {
				out[pkg] = true
			}
		}
	}
	return out
}

func requirementName(line string) string {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "-") {
		return ""
	}
	for i, ch := range line {
		if strings.ContainsRune("=<>!~[ ;", ch) {
			return strings.ToLower(strings.TrimSpace(line[:i]))
		}
	}
	return strings.ToLower(line)
}

func detectPythonFramework(dir string, deps map[string]bool) string {
	if _, err := os.Stat(filepath.Join(dir, "manage.py")); err == nil {
		return "django"
	}
	switch {
	case deps["django"]:
		return "django"
	case deps["fastapi"]:
		return "fastapi"
	case deps["starlette"]:
		return "starlette"
	case deps["flask"]:
		return "flask"
	}
	return "python"
}

func detectPythonPackageManager(dir string) string {
	checks := []struct {
		file string
		name string
	}{
		{"uv.lock", "uv"},
		{"poetry.lock", "poetry"},
		{"Pipfile.lock", "pipenv"},
		{"Pipfile", "pipenv"},
	}
	for _, c := range checks {
		if _, err := os.Stat(filepath.Join(dir, c.file)); err == nil {
			return c.name
		}
	}
	if raw, err := os.ReadFile(filepath.Join(dir, "pyproject.toml")); err == nil {
		body := string(raw)
		if strings.Contains(body, "[tool.uv") {
			return "uv"
		}
		if strings.Contains(body, "[tool.poetry") {
			return "poetry"
		}
	}
	return "pip"
}

func pythonDevCommand(dir, framework string) (string, int) {
	switch framework {
	case "django":
		return "python manage.py runserver 127.0.0.1:$PORT", 8000
	case "flask":
		return "flask run --host 127.0.0.1 --port $PORT", 5000
	case "fastapi", "starlette":
		return "uvicorn " + asgiTarget(dir) + " --host 127.0.0.1 --port $PORT", 8000
	}
	if entry := firstExisting(dir, "main.py", "app.py", "run.py"); entry != "" {
		return "python " + entry, 8000
	}
	return "python main.py", 8000
}

func asgiTarget(dir string) string {
	for _, mod := range []string{"main", "app", "asgi", "server"} {
		if _, err := os.Stat(filepath.Join(dir, mod+".py")); err == nil {
			return mod + ":app"
		}
	}
	return "main:app"
}

func firstExisting(dir string, names ...string) string {
	for _, name := range names {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return name
		}
	}
	return ""
}
