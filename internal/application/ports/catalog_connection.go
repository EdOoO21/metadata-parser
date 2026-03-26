package ports

import "context"

type CatalogConnection interface {
	// Repository возвращает репозиторий каталога для выполнения операций.
	Repository() CatalogRepository
	// Close закрывает подключение к каталогу.
	Close()
}

type CatalogRepositoryOpener interface {
	// Open открывает подключение к каталогу по имени env-переменной с DSN.
	Open(ctx context.Context, dsnEnv string) (CatalogConnection, error)
}
