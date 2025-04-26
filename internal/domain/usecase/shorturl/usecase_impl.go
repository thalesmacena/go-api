package shorturl

import (
	"errors"
	"go-api/internal/domain/entity"
	"go-api/internal/domain/gateway/db"
	"go-api/internal/domain/model"
	"go-api/pkg/msg"
	"math/rand/v2"
	"net/url"
)

const infiniteExpiration = "9999-12-31 23:59:59"

type shortUrlUseCase struct {
	gateway db.ShortUrlGateway
}

func NewShortUrlUseCase(gateway db.ShortUrlGateway) UseCase {
	return &shortUrlUseCase{
		gateway: gateway,
	}
}

func (uc *shortUrlUseCase) FindAll(offset int, limit int) ([]entity.ShortUrl, error) {
	return uc.gateway.FindAll(offset, limit)
}

func (uc *shortUrlUseCase) FindByURLPart(urlPart string, offset int, limit int) ([]entity.ShortUrl, error) {
	return uc.gateway.FindByURLPart(urlPart, offset, limit)
}

func (uc *shortUrlUseCase) FindByID(id string) (*entity.ShortUrl, error) {
	return uc.gateway.FindByID(id)
}

func (uc *shortUrlUseCase) FindByHash(hash string) (*entity.ShortUrl, error) {
	return uc.gateway.FindByHash(hash)
}

func (uc *shortUrlUseCase) Create(dto model.CreateShortUrlDTO) (*entity.ShortUrl, error) {
	if dto.Url == "" {
		return nil, errors.New(msg.GetMessage("short-url.error.empty-url"))
	}
	if !isValidURL(dto.Url) {
		return nil, errors.New(msg.GetMessage("short-url.error.invalid-url"))
	}

	if dto.Expiration == "" {
		dto.Expiration = infiniteExpiration
	}

	hash := generateUniqueHash()

	shortURL := entity.ShortUrl{
		Hash:       hash,
		Url:        dto.Url,
		Expiration: dto.Expiration,
	}

	createdShortUrl, err := uc.gateway.Create(shortURL)
	if err != nil {
		return nil, err
	}

	return createdShortUrl, nil
}

func (uc *shortUrlUseCase) UpdateByHash(hash string, dto model.UpdateShortUrlDTO) (*entity.ShortUrl, error) {
	if dto.Url == "" {
		return nil, errors.New(msg.GetMessage("short-url.error.empty-url"))
	}
	if !isValidURL(dto.Url) {
		return nil, errors.New(msg.GetMessage("short-url.error.invalid-url"))
	}
	if dto.Hash == "" {
		return nil, errors.New(msg.GetMessage("short-url.error.empty-hash"))
	}

	existing, err := uc.gateway.FindByHash(hash)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, errors.New(msg.GetMessage("short-url.error.not-found"))
	}

	other, err := uc.gateway.FindByHash(dto.Hash)
	if err != nil {
		return nil, err
	}
	if other != nil && other.ID != existing.ID {
		return nil, errors.New(msg.GetMessage("short-url.error.existent-hash"))
	}

	existing.Url = dto.Url
	existing.Hash = dto.Hash
	if dto.Expiration == "" {
		dto.Expiration = existing.Expiration
	}
	existing.Expiration = dto.Expiration

	updatedShortUrl, err := uc.gateway.UpdateByHash(hash, *existing)
	if err != nil {
		return nil, err
	}

	return updatedShortUrl, nil
}

func (uc *shortUrlUseCase) UpdateByID(id string, dto model.UpdateShortUrlDTO) (*entity.ShortUrl, error) {
	if dto.Url == "" {
		return nil, errors.New(msg.GetMessage("short-url.error.empty-url"))
	}
	if !isValidURL(dto.Url) {
		return nil, errors.New(msg.GetMessage("short-url.error.invalid-url"))
	}

	existing, err := uc.gateway.FindByID(id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, errors.New(msg.GetMessage("short-url.error.not-found"))
	}

	if dto.Hash != "" {
		other, err := uc.gateway.FindByHash(dto.Hash)
		if err != nil {
			return nil, err
		}
		if other != nil && other.Hash != dto.Hash {
			return nil, errors.New(msg.GetMessage("short-url.error.existent-hash"))
		}
	}

	existing.Url = dto.Url
	existing.Expiration = dto.Expiration

	updatedShortUrl, err := uc.gateway.UpdateByID(id, *existing)
	if err != nil {
		return nil, err
	}

	return updatedShortUrl, nil
}

func (uc *shortUrlUseCase) ClearAllByExpiration() error {
	return uc.gateway.ClearAllByExpiration()
}

func (uc *shortUrlUseCase) DeleteByID(id string) error {
	return uc.gateway.DeleteByID(id)
}

func (uc *shortUrlUseCase) DeleteByHash(hash string) error {
	return uc.gateway.DeleteByHash(hash)
}

func isValidURL(u string) bool {
	_, err := url.ParseRequestURI(u)
	return err == nil
}

func generateUniqueHash() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	hash := make([]byte, 8)
	for i := range hash {
		hash[i] = charset[rand.IntN(len(charset))]
	}
	return string(hash)
}
