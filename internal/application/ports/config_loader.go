package ports

import "github.com/EdOoO21/metadata-parser/internal/application/dto"

type ConfigLoader interface {
	Load(path string) (*dto.AppConfig, error)
}
