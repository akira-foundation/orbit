package projects

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"orbit-app/internal/analyzer"
)

type Service struct {
	repo     *Repository
	analyzer analyzer.Analyzer
	tld      string
}

func NewService(repo *Repository, a analyzer.Analyzer, tld string) *Service {
	return &Service{repo: repo, analyzer: a, tld: tld}
}

func (s *Service) AnalyzePath(path string) (*analyzer.Analysis, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("path is required")
	}
	return s.analyzer.Analyze(path)
}

func (s *Service) Create(ctx context.Context, path string) (*Project, error) {
	a, err := s.AnalyzePath(path)
	if err != nil {
		return nil, err
	}
	slug := Slugify(a.Name)
	p := &Project{
		Name:              a.Name,
		Path:              a.Path,
		Slug:              slug,
		LocalDomain:       slug + "." + s.tld,
		DetectedFramework: a.Framework,
		PackageManager:    a.PackageManager,
		DevCommand:        a.DevCommand,
		DevPort:           a.DevPort,
		Status:            StatusStopped,
	}
	for name, cmd := range a.Scripts {
		p.Scripts = append(p.Scripts, Script{Name: name, Command: cmd})
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Service) List(ctx context.Context) ([]Project, error) {
	return s.repo.List(ctx)
}

func (s *Service) Get(ctx context.Context, id string) (*Project, error) {
	return s.repo.Get(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *Service) UpdateStatus(ctx context.Context, id string, status Status) error {
	return s.repo.UpdateStatus(ctx, id, status)
}

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

func Slugify(in string) string {
	in = strings.ToLower(strings.TrimSpace(in))
	in = strings.TrimPrefix(in, "@")
	in = strings.ReplaceAll(in, "/", "-")
	in = slugRe.ReplaceAllString(in, "-")
	in = strings.Trim(in, "-")
	if in == "" {
		in = "project"
	}
	return in
}
