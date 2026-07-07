package bindings

import (
	"context"
	"fmt"

	"orbit-app/internal/projects"
	"orbit-app/internal/services"
	"orbit-app/internal/services/miniostore"
	"orbit-app/internal/services/pgbrowse"
)

const queryRowLimit = 500

type Data struct {
	ctx      context.Context
	service  *projects.Service
	services *services.Manager
}

func NewData() *Data { return &Data{} }

func (d *Data) Attach(dep Deps) {
	d.ctx = dep.Ctx
	d.service = dep.Service
	d.services = dep.Services
}

func (d *Data) dsn(projectID string) (string, error) {
	p, err := d.service.Get(d.ctx, projectID)
	if err != nil {
		return "", err
	}
	host, port, ok := d.services.RunningEndpoint("postgres")
	if !ok {
		return "", fmt.Errorf("postgres service is not running")
	}
	return fmt.Sprintf("postgres://orbit@%s:%d/%s", host, port, p.Slug), nil
}

func (d *Data) DBTables(projectID string) ([]string, error) {
	dsn, err := d.dsn(projectID)
	if err != nil {
		return nil, err
	}
	return pgbrowse.Tables(d.ctx, dsn)
}

func (d *Data) DBQuery(projectID, sql string, allowWrites bool) (*pgbrowse.QueryResult, error) {
	if !allowWrites {
		if err := pgbrowse.ReadOnly(sql); err != nil {
			return nil, err
		}
	}
	dsn, err := d.dsn(projectID)
	if err != nil {
		return nil, err
	}
	return pgbrowse.Query(d.ctx, dsn, sql, queryRowLimit)
}

func (d *Data) minioEndpoint() (string, int, error) {
	host, port, ok := d.services.RunningEndpoint("minio")
	if !ok {
		return "", 0, fmt.Errorf("minio service is not running")
	}
	return host, port, nil
}

func (d *Data) BucketList() ([]string, error) {
	host, port, err := d.minioEndpoint()
	if err != nil {
		return nil, err
	}
	return miniostore.ListBuckets(d.ctx, host, port)
}

func (d *Data) BucketObjects(bucket, prefix string) ([]miniostore.Object, error) {
	host, port, err := d.minioEndpoint()
	if err != nil {
		return nil, err
	}
	return miniostore.ListObjects(d.ctx, host, port, bucket, prefix)
}

func (d *Data) ObjectURL(bucket, key string) (string, error) {
	host, port, err := d.minioEndpoint()
	if err != nil {
		return "", err
	}
	return miniostore.PresignedURL(d.ctx, host, port, bucket, key)
}
