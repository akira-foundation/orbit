package bindings

import (
	"context"
	"fmt"

	"orbit-app/internal/services"
	"orbit-app/internal/services/miniostore"
	"orbit-app/internal/services/pgbrowse"
)

const queryRowLimit = 500

type Data struct {
	ctx      context.Context
	services *services.Manager
}

func NewData() *Data { return &Data{} }

func (d *Data) Attach(dep Deps) {
	d.ctx = dep.Ctx
	d.services = dep.Services
}

func (d *Data) dsn(db string) (string, error) {
	host, port, ok := d.services.RunningEndpoint("postgres")
	if !ok {
		return "", fmt.Errorf("postgres service is not running")
	}
	return fmt.Sprintf("postgres://orbit@%s:%d/%s?sslmode=disable", host, port, db), nil
}

func (d *Data) DBDatabases() ([]string, error) {
	dsn, err := d.dsn("postgres")
	if err != nil {
		return nil, err
	}
	return pgbrowse.Databases(d.ctx, dsn)
}

func (d *Data) DBSchema(database string) (map[string][]string, error) {
	dsn, err := d.dsn(database)
	if err != nil {
		return nil, err
	}
	return pgbrowse.Schema(d.ctx, dsn)
}

func (d *Data) DBTables(database string) ([]string, error) {
	dsn, err := d.dsn(database)
	if err != nil {
		return nil, err
	}
	return pgbrowse.Tables(d.ctx, dsn)
}

func (d *Data) DBQuery(database, sql string, allowWrites bool) (*pgbrowse.QueryResult, error) {
	if !allowWrites {
		if err := pgbrowse.ReadOnly(sql); err != nil {
			return nil, err
		}
	}
	dsn, err := d.dsn(database)
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
