package projects

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"orbit-app/internal/analyzer"
)

type Service struct {
	repo         *Repository
	analyzer     analyzer.Analyzer
	domainSuffix string
}

func NewService(repo *Repository, a analyzer.Analyzer, domainSuffix string) *Service {
	return &Service{repo: repo, analyzer: a, domainSuffix: domainSuffix}
}

func (s *Service) DomainSuffix() string { return s.domainSuffix }

func (s *Service) DomainFor(slug string) string { return slug + "." + s.domainSuffix }

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
		LocalDomain:       s.DomainFor(slug),
		DetectedFramework: a.Framework,
		PackageManager:    a.PackageManager,
		DevCommand:        a.DevCommand,
		DevPort:           a.DevPort,
		NodeVersion:       a.NodeVersion,
		RuntimeKind:       RuntimeKind(a.RuntimeKind),
		PHPVersion:        a.PHPVersion,
		Status:            StatusStopped,
	}
	for name, cmd := range a.Scripts {
		p.Scripts = append(p.Scripts, Script{Name: name, Command: cmd})
	}
	for _, ps := range a.Processes {
		p.Processes = append(p.Processes, ProcessSpec{
			Role:       ProcessRole(ps.Role),
			Kind:       RuntimeKind(ps.Kind),
			WorkDir:    ps.WorkDir,
			Command:    ps.Command,
			Port:       ps.Port,
			PHPVersion: ps.PHPVersion,
		})
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

func (s *Service) InstalledHash(ctx context.Context, id string) (string, error) {
	return s.repo.InstalledHash(ctx, id)
}

func (s *Service) SetInstalledHash(ctx context.Context, id, hash string) error {
	return s.repo.SetInstalledHash(ctx, id, hash)
}

func (s *Service) SetSecure(ctx context.Context, id string, secure bool) error {
	return s.repo.SetSecure(ctx, id, secure)
}

func (s *Service) ListDomains(ctx context.Context) ([]Domain, error) {
	return s.repo.ListDomains(ctx)
}

func (s *Service) GetDomain(ctx context.Context, host string) (*Domain, error) {
	return s.repo.GetDomain(ctx, host)
}

func (s *Service) UpdateDomainPort(ctx context.Context, id string, port int) error {
	return s.repo.UpdateDomainPort(ctx, id, port)
}

func (s *Service) CreateGroup(ctx context.Context, name string) (*Group, error) {
	return s.repo.CreateGroup(ctx, name)
}

func (s *Service) ListGroups(ctx context.Context) ([]Group, error) {
	return s.repo.ListGroups(ctx)
}

func (s *Service) DeleteGroup(ctx context.Context, id string) error {
	return s.repo.DeleteGroup(ctx, id)
}

func (s *Service) AddToGroup(ctx context.Context, groupID, projectID string) error {
	return s.repo.AddGroupMember(ctx, groupID, projectID)
}

func (s *Service) RemoveFromGroup(ctx context.Context, groupID, projectID string) error {
	return s.repo.RemoveGroupMember(ctx, groupID, projectID)
}

func (s *Service) GroupMembers(ctx context.Context, groupID string) ([]string, error) {
	return s.repo.GroupMembers(ctx, groupID)
}

var slugRe = regexp.MustCompile(`[^a-z0-9.]+`)

func Slugify(in string) string {
	in = strings.ToLower(strings.TrimSpace(in))
	in = strings.TrimPrefix(in, "@")
	in = strings.ReplaceAll(in, "/", "-")
	in = slugRe.ReplaceAllString(in, "-")
	in = strings.Trim(in, "-.")
	if in == "" {
		in = "project"
	}
	return in
}
