package db

import (
	"go-api/internal/domain/entity"
)

type ShortUrlGateway interface {
	FindAll(offset int, limit int) ([]entity.ShortUrl, error)
	FindByURLPart(urlPart string, offset int, limit int) ([]entity.ShortUrl, error)
	FindByID(id string) (*entity.ShortUrl, error)
	FindByHash(hash string) (*entity.ShortUrl, error)

	CountAll() (int64, error)
	CountByURLPart(urlPart string) (int64, error)

	Create(shortURL entity.ShortUrl) (*entity.ShortUrl, error)
	UpdateByHash(hash string, updated entity.ShortUrl) (*entity.ShortUrl, error)
	UpdateByID(id string, updated entity.ShortUrl) (*entity.ShortUrl, error)

	ClearAllByExpiration() error
	DeleteByID(id string) error
	DeleteByHash(hash string) error
}
