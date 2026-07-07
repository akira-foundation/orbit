package services

import (
	"context"
	"fmt"
	"runtime"
)

type Platform struct {
	URL               string
	SHA256            string
	ArchiveBinaryPath string
}

type SetupField struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type SetupSnippet struct {
	Label    string `json:"label"`
	Language string `json:"language"`
	Code     string `json:"code"`
}

type SetupInfo struct {
	Fields   []SetupField   `json:"fields"`
	Snippets []SetupSnippet `json:"snippets"`
}

type Engine struct {
	ID           string
	Version      string
	DisplayName  string
	Description  string
	Family       string
	Port         int
	SMTPPort     int
	WebPort      int
	APIPort      int
	Bind         string
	WebDomain    string
	ExtractTree  bool
	RawBinary    bool
	Zip          bool
	Env          []string
	ReadyMarkers []string
	Platforms    map[string]Platform
	Setup        SetupInfo
	Init         func(binDir, dataDir string) error
	Provision    func(ctx context.Context, port int, slug string) (map[string]string, error)
	argsTemplate func(p Platform, dataDir string) []string
}

func (e Engine) PlatformKey() string {
	return runtime.GOOS + "/" + runtime.GOARCH
}

func (e Engine) CurrentPlatform() (Platform, bool) {
	p, ok := e.Platforms[e.PlatformKey()]
	return p, ok
}

func (e Engine) Args(p Platform, dataDir string) []string {
	if e.argsTemplate == nil {
		return nil
	}
	return e.argsTemplate(p, dataDir)
}

func mailpitURL(version, goos, goarch string) string {
	return fmt.Sprintf(
		"https://github.com/axllent/mailpit/releases/download/%s/mailpit-%s-%s.tar.gz",
		version, goos, goarch)
}

func Catalog() []Engine {
	const mpVer = "v1.30.3"
	mp := Engine{
		ID:          "mailpit",
		Version:     mpVer,
		DisplayName: "Mailpit",
		Description: "Captures outgoing mail from your apps in a local inbox.",
		Port:        40321,
		SMTPPort:    40322,
		WebPort:     40321,
		Bind:        "127.0.0.2",
		WebDomain:   "mail",
		ReadyMarkers: []string{
			"accessible via http",
			"[http] starting on",
		},
		Platforms: map[string]Platform{
			"darwin/arm64": {
				URL:               mailpitURL(mpVer, "darwin", "arm64"),
				SHA256:            "46b68e5701c32f2137e97d325605f7e8f0fbb6518e567b7589147c3534bd943e",
				ArchiveBinaryPath: "mailpit",
			},
			"darwin/amd64": {
				URL:               mailpitURL(mpVer, "darwin", "amd64"),
				SHA256:            "ea8c2f5ac717ece100b453de282474b46e8f4c327d3e61bee6348f60989eade3",
				ArchiveBinaryPath: "mailpit",
			},
			"linux/arm64": {
				URL:               mailpitURL(mpVer, "linux", "arm64"),
				SHA256:            "4211e158fcf46862b9b15bacd1fb10253a1865617ed5f38f27cbec230f89ec84",
				ArchiveBinaryPath: "mailpit",
			},
			"linux/amd64": {
				URL:               mailpitURL(mpVer, "linux", "amd64"),
				SHA256:            "6c7af993fb4054def4adfc7c85b40f9570fd6172eaccd0221c4969f2cc7a6294",
				ArchiveBinaryPath: "mailpit",
			},
		},
	}
	mp.argsTemplate = func(p Platform, dataDir string) []string {
		return []string{
			"--database", dataDir + "/mailpit.db",
			"--smtp", fmt.Sprintf("%s:%d", mp.Bind, mp.SMTPPort),
			"--listen", fmt.Sprintf("%s:%d", mp.Bind, mp.WebPort),
		}
	}
	mp.Setup = SetupInfo{
		Fields: []SetupField{
			{Label: "SMTP host", Value: mp.Bind},
			{Label: "SMTP port", Value: fmt.Sprintf("%d", mp.SMTPPort)},
			{Label: "Auth", Value: "none"},
			{Label: "Encryption", Value: "none"},
			{Label: "Web inbox", Value: mp.WebDomain + ".orbit.test"},
		},
		Snippets: []SetupSnippet{
			{
				Label:    ".env",
				Language: "ini",
				Code:     fmt.Sprintf("MAIL_HOST=%s\nMAIL_PORT=%d", mp.Bind, mp.SMTPPort),
			},
			{
				Label:    "Laravel .env",
				Language: "ini",
				Code: fmt.Sprintf("MAIL_MAILER=smtp\nMAIL_HOST=%s\nMAIL_PORT=%d\n", mp.Bind, mp.SMTPPort) +
					"MAIL_USERNAME=null\nMAIL_PASSWORD=null\nMAIL_ENCRYPTION=null",
			},
			{
				Label:    "Node / Nodemailer",
				Language: "javascript",
				Code: "const transport = nodemailer.createTransport({\n" +
					fmt.Sprintf("  host: %q,\n  port: %d,\n  secure: false,\n});", mp.Bind, mp.SMTPPort),
			},
		},
	}
	out := append([]Engine{mp}, postgresEngines()...)
	out = append(out, minioEngine())
	return append(out, redisEngine())
}

func ResolveEngine(id string) (Engine, bool) {
	for _, e := range Catalog() {
		if e.ID == id {
			return e, true
		}
	}
	return Engine{}, false
}
