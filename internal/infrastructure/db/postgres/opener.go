package postgres

import (
	"context"

	appports "github.com/EdOoO21/metadata-parser/internal/application/ports"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CatalogConnection struct {
	pool *pgxpool.Pool
	repo appports.CatalogRepository
}

func (c *CatalogConnection) Repository() appports.CatalogRepository {
	return c.repo
}

func (c *CatalogConnection) Close() {
	c.pool.Close()
}

type RepositoryOpener struct{}

func NewRepositoryOpener() *RepositoryOpener {
	return &RepositoryOpener{}
}

func (o *RepositoryOpener) Open(ctx context.Context, dsnEnv string) (appports.CatalogConnection, error) {
	pool, err := NewPoolFromEnv(ctx, dsnEnv)
	if err != nil {
		return nil, err
	}

	return &CatalogConnection{
		pool: pool,
		repo: NewRepository(pool),
	}, nil
}
