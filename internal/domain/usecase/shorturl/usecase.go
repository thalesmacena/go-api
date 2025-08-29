package shorturl

import (
	"go-api/internal/domain/entity"
	"go-api/internal/domain/model"
)

type UseCase interface {
	FindAll(page int, size int) (*model.Page[entity.ShortUrl], error)
	FindByURLPart(urlPart string, page int, size int) (*model.Page[entity.ShortUrl], error)
	FindByID(id string) (*entity.ShortUrl, error)
	FindByHash(hash string) (*entity.ShortUrl, error)
	Create(dto model.CreateShortUrlDTO) (*entity.ShortUrl, error)
	UpdateByHash(hash string, dto model.UpdateShortUrlDTO) (*entity.ShortUrl, error)
	UpdateByID(id string, dto model.UpdateShortUrlDTO) (*entity.ShortUrl, error)
	ClearAllByExpiration() error
	DeleteByID(id string) error
	DeleteByHash(hash string) error
}
