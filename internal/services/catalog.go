package services

import (
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
	SMTPPort     int
	WebPort      int
	APIPort      int
	Bind         string
	WebDomain    string
	ReadyMarkers []string
	Platforms    map[string]Platform
	Setup        SetupInfo
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
		SMTPPort:    1025,
		WebPort:     8025,
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
			"--smtp", mp.Bind + ":1025",
			"--listen", mp.Bind + ":8025",
		}
	}
	mp.Setup = SetupInfo{
		Fields: []SetupField{
			{Label: "SMTP host", Value: "127.0.0.2"},
			{Label: "SMTP port", Value: "1025"},
			{Label: "Auth", Value: "none"},
			{Label: "Encryption", Value: "none"},
			{Label: "Web inbox", Value: "mail.orbit.test"},
		},
		Snippets: []SetupSnippet{
			{
				Label:    ".env",
				Language: "ini",
				Code:     "MAIL_HOST=127.0.0.2\nMAIL_PORT=1025",
			},
			{
				Label:    "Laravel .env",
				Language: "ini",
				Code: "MAIL_MAILER=smtp\nMAIL_HOST=127.0.0.2\nMAIL_PORT=1025\n" +
					"MAIL_USERNAME=null\nMAIL_PASSWORD=null\nMAIL_ENCRYPTION=null",
			},
			{
				Label:    "Node / Nodemailer",
				Language: "javascript",
				Code: "const transport = nodemailer.createTransport({\n" +
					"  host: \"127.0.0.2\",\n  port: 1025,\n  secure: false,\n});",
			},
		},
	}
	return []Engine{mp}
}

func ResolveEngine(id string) (Engine, bool) {
	for _, e := range Catalog() {
		if e.ID == id {
			return e, true
		}
	}
	return Engine{}, false
}
