package bindings

import (
	"context"

	"orbit-app/internal/config"
	"orbit-app/internal/projects"
	"orbit-app/internal/proxy"
	"orbit-app/internal/share"
)

type ShareInfo struct {
	Enabled bool   `json:"enabled"`
	URL     string `json:"url"`
	QR      string `json:"qr"`
}

type Share struct {
	ctx     context.Context
	cfg     *config.Config
	service *projects.Service
	proxy   *proxy.Server
	state   *share.State
}

func NewShare() *Share { return &Share{} }

func (s *Share) Attach(d Deps) {
	s.ctx = d.Ctx
	s.cfg = d.Cfg
	s.service = d.Service
	s.proxy = d.ProxyServer
	s.state = d.Share
}

func (s *Share) info(projectID string, enabled bool) (*ShareInfo, error) {
	if !enabled {
		return &ShareInfo{Enabled: false}, nil
	}
	p, err := s.service.Get(s.ctx, projectID)
	if err != nil {
		return nil, err
	}
	ip, err := share.PrimaryLANIP()
	if err != nil {
		return nil, err
	}
	url := share.ShareURL(ip, s.cfg.InternalPort(), p.Slug)
	qr, err := share.QRDataURI(url)
	if err != nil {
		return nil, err
	}
	return &ShareInfo{Enabled: true, URL: url, QR: qr}, nil
}

func (s *Share) ShareInfo(projectID string) (*ShareInfo, error) {
	return s.info(projectID, s.state.Enabled(projectID))
}

func (s *Share) EnableLANShare(projectID string) (*ShareInfo, error) {
	ip, err := share.PrimaryLANIP()
	if err != nil {
		return nil, err
	}
	if s.state.Count() == 0 {
		if err := s.proxy.EnableLAN(ip + ":" + s.cfg.InternalPort()); err != nil {
			return nil, err
		}
	}
	s.state.Enable(projectID)
	return s.info(projectID, true)
}

func (s *Share) DisableLANShare(projectID string) error {
	s.state.Disable(projectID)
	if s.state.Count() == 0 {
		return s.proxy.DisableLAN()
	}
	return nil
}
