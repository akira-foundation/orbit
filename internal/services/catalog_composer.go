package services

const (
	composerVersion = "2.10.2"
	composerSHA256  = "5ee7125f8a30a34d246cefdc0bc85b8a783b28f2aec968994118512350d28027"
	composerURL     = "https://getcomposer.org/download/" + composerVersion + "/composer.phar"
)

// composer.phar is arch-independent: one download works on every platform.
func ComposerEngine() Engine {
	platform := Platform{URL: composerURL, SHA256: composerSHA256, ArchiveBinaryPath: "composer.phar"}
	return Engine{
		ID:          "composer",
		Version:     composerVersion,
		DisplayName: "Composer",
		Description: "Bundled Composer, used to install PHP project dependencies.",
		Family:      "composer",
		RawBinary:   true,
		Platforms: map[string]Platform{
			"darwin/arm64": platform,
			"darwin/amd64": platform,
			"linux/arm64":  platform,
			"linux/amd64":  platform,
		},
	}
}
