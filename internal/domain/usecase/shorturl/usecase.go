package shorturl

import (
	"go-api/internal/domain/entity"
	"go-api/internal/domain/model"
)

type UseCase interface {
	FindAll(offset int, limit int) ([]entity.ShortUrl, error)
	FindByURLPart(urlPart string, offset int, limit int) ([]entity.ShortUrl, error)
	FindByID(id string) (*entity.ShortUrl, error)
	FindByHash(hash string) (*entity.ShortUrl, error)
	Create(dto model.CreateShortUrlDTO) (*entity.ShortUrl, error)
	UpdateByHash(hash string, dto model.UpdateShortUrlDTO) (*entity.ShortUrl, error)
	UpdateByID(id string, dto model.UpdateShortUrlDTO) (*entity.ShortUrl, error)
	ClearAllByExpiration() error
	DeleteByID(id string) error
	DeleteByHash(hash string) error
}
