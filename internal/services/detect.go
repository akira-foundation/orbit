package services

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type detectSignal struct {
	Token string
	Names []string
}

var detectSignals = []detectSignal{
	{
		Token: "mailpit",
		Names: []string{
			"nodemailer", "@sendgrid/mail", "resend",
			"swiftmailer/swiftmailer", "symfony/mailer",
		},
	},
	{
		Token: "minio",
		Names: []string{
			"@aws-sdk/client-s3", "aws-sdk", "minio",
			"league/flysystem-aws-s3-v3",
		},
	},
	{
		Token: "postgres",
		Names: []string{
			"pg", "postgres", "@prisma/client", "prisma",
			"typeorm", "sequelize", "drizzle-orm", "doctrine/dbal",
		},
	},
}

func DetectTokens(projectPath string) []string {
	deps := readDependencyNames(projectPath)
	if len(deps) == 0 {
		return nil
	}
	var out []string
	for _, sig := range detectSignals {
		for _, name := range sig.Names {
			if deps[name] {
				out = append(out, sig.Token)
				break
			}
		}
	}
	return out
}

func readDependencyNames(projectPath string) map[string]bool {
	out := map[string]bool{}
	addPackageJSON(filepath.Join(projectPath, "package.json"), out)
	addComposerJSON(filepath.Join(projectPath, "composer.json"), out)
	return out
}

func addPackageJSON(path string, out map[string]bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if json.Unmarshal(data, &pkg) != nil {
		return
	}
	for k := range pkg.Dependencies {
		out[k] = true
	}
	for k := range pkg.DevDependencies {
		out[k] = true
	}
}

func addComposerJSON(path string, out map[string]bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var comp struct {
		Require    map[string]string `json:"require"`
		RequireDev map[string]string `json:"require-dev"`
	}
	if json.Unmarshal(data, &comp) != nil {
		return
	}
	for k := range comp.Require {
		out[k] = true
	}
	for k := range comp.RequireDev {
		out[k] = true
	}
}
