package services

import (
	"context"
	"fmt"

	"orbit-app/internal/services/miniostore"
)

const minioRelease = "RELEASE.2025-09-07T16-13-09Z"

var minioChecksums = map[string]string{
	"darwin-arm64": "7c3b3039b76e55a1b80935848ed83998d5e8d317374f87851f46a019ff5c0aa4",
	"darwin-amd64": "4759080aeef7385aaceaac1131c30aaeb99605921553967dc6d3ef4e16ac64f9",
	"linux-arm64":  "5c83cd2cf151717ba0243f73e1c7802ff36e272b67144bdd7f1f7d684fd6f03d",
	"linux-amd64":  "7c5bd8512c6e966455b1d198209358b2d191c77a83ab377c4073281065fb855f",
}

var minioPlatformKeys = map[string]string{
	"darwin/arm64": "darwin-arm64",
	"darwin/amd64": "darwin-amd64",
	"linux/arm64":  "linux-arm64",
	"linux/amd64":  "linux-amd64",
}

func MinioEnv(host string, port int, slug string) map[string]string {
	endpoint := fmt.Sprintf("http://%s:%d", host, port)
	return map[string]string{
		"AWS_ACCESS_KEY_ID":           miniostore.RootUser,
		"AWS_SECRET_ACCESS_KEY":       miniostore.RootPassword,
		"AWS_DEFAULT_REGION":          "us-east-1",
		"AWS_BUCKET":                  slug,
		"AWS_ENDPOINT":                endpoint,
		"AWS_URL":                     endpoint + "/" + slug,
		"AWS_USE_PATH_STYLE_ENDPOINT": "true",
	}
}

func minioEngine() Engine {
	platforms := map[string]Platform{}
	for key, plat := range minioPlatformKeys {
		platforms[key] = Platform{
			URL: fmt.Sprintf(
				"https://github.com/minio/minio/releases/download/%s/minio.%s.%s",
				minioRelease, plat, minioRelease),
			SHA256:            minioChecksums[plat],
			ArchiveBinaryPath: "minio",
		}
	}
	e := Engine{
		ID:          "minio",
		Version:     minioRelease,
		DisplayName: "MinIO",
		Description: "S3-compatible object storage. One bucket per project, created automatically.",
		Family:      "minio",
		Port:        40340,
		APIPort:     40340,
		WebPort:     40341,
		Bind:        "127.0.0.2",
		WebDomain:   "s3",
		RawBinary:   true,
		Env: []string{
			"MINIO_ROOT_USER=" + miniostore.RootUser,
			"MINIO_ROOT_PASSWORD=" + miniostore.RootPassword,
		},
		ReadyMarkers: []string{
			"api: http",
			"s3-api:",
		},
		Platforms: platforms,
	}
	e.argsTemplate = func(p Platform, dataDir string) []string {
		return []string{
			"server", dataDir,
			"--address", fmt.Sprintf("%s:40340", e.Bind),
			"--console-address", fmt.Sprintf("%s:40341", e.Bind),
		}
	}
	e.Provision = func(ctx context.Context, port int, slug string) (map[string]string, error) {
		if err := miniostore.EnsureBucket(ctx, e.Bind, port, slug); err != nil {
			return nil, err
		}
		return MinioEnv(e.Bind, port, slug), nil
	}
	e.Setup = SetupInfo{
		Fields: []SetupField{
			{Label: "Endpoint", Value: "http://127.0.0.2:40340"},
			{Label: "Access key", Value: miniostore.RootUser},
			{Label: "Secret key", Value: miniostore.RootPassword},
			{Label: "Region", Value: "us-east-1"},
			{Label: "Bucket", Value: "<project-slug>"},
			{Label: "Console", Value: "s3.orbit.test"},
		},
		Snippets: []SetupSnippet{
			{Label: ".env", Language: "ini",
				Code: "AWS_ACCESS_KEY_ID=orbit\nAWS_SECRET_ACCESS_KEY=orbitsecret\n" +
					"AWS_DEFAULT_REGION=us-east-1\nAWS_BUCKET=<project-slug>\n" +
					"AWS_ENDPOINT=http://127.0.0.2:40340\nAWS_USE_PATH_STYLE_ENDPOINT=true"},
			{Label: "Laravel filesystems", Language: "ini",
				Code: "FILESYSTEM_DISK=s3\nAWS_ACCESS_KEY_ID=orbit\nAWS_SECRET_ACCESS_KEY=orbitsecret\n" +
					"AWS_DEFAULT_REGION=us-east-1\nAWS_BUCKET=<project-slug>\n" +
					"AWS_ENDPOINT=http://127.0.0.2:40340\nAWS_USE_PATH_STYLE_ENDPOINT=true"},
			{Label: "Node / aws-sdk", Language: "javascript",
				Code: "const s3 = new S3Client({\n  endpoint: \"http://127.0.0.2:40340\",\n" +
					"  region: \"us-east-1\",\n  forcePathStyle: true,\n" +
					"  credentials: { accessKeyId: \"orbit\", secretAccessKey: \"orbitsecret\" },\n});"},
		},
	}
	return e
}
