package ports

import "catalog-tool/internal/application/dto"

type ConfigLoader interface {
	Load(path string) (*dto.AppConfig, error)
}
