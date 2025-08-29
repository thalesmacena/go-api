package db

import (
	"database/sql"
	"errors"
	"go-api/internal/domain/entity"
	"time"

	"github.com/google/uuid"
)

const timeLayout = "2006-01-02 15:04:05"

type SQLCShortUrlGateway struct {
	DB *sql.DB
}

func NewSQLCShortUrlGateway(db *sql.DB) *SQLCShortUrlGateway {
	return &SQLCShortUrlGateway{DB: db}
}

func (gateway *SQLCShortUrlGateway) FindAll(offset int, limit int) ([]entity.ShortUrl, error) {
	rows, err := gateway.DB.Query(`
		SELECT id, hash, url, expiration, created_at, updated_at
		FROM short_urls
		ORDER BY created_at DESC
		OFFSET $1 LIMIT $2`, offset, limit)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			err = closeErr
		}
	}()

	results := make([]entity.ShortUrl, 0)
	for rows.Next() {
		var s entity.ShortUrl
		if err := rows.Scan(&s.ID, &s.Hash, &s.Url, &s.Expiration, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		results = append(results, s)
	}
	return results, nil
}

func (gateway *SQLCShortUrlGateway) FindByURLPart(urlPart string, offset int, limit int) ([]entity.ShortUrl, error) {
	rows, err := gateway.DB.Query(`
		SELECT id, hash, url, expiration, created_at, updated_at
		FROM short_urls
		WHERE url ILIKE '%' || $1 || '%'
		ORDER BY created_at DESC
		OFFSET $2 LIMIT $3`, urlPart, offset, limit)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			err = closeErr
		}
	}()

	results := make([]entity.ShortUrl, 0)
	for rows.Next() {
		var s entity.ShortUrl
		if err := rows.Scan(&s.ID, &s.Hash, &s.Url, &s.Expiration, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		results = append(results, s)
	}
	return results, nil
}

func (gateway *SQLCShortUrlGateway) FindByID(id string) (*entity.ShortUrl, error) {
	var s entity.ShortUrl
	err := gateway.DB.QueryRow(`
		SELECT id, hash, url, expiration, created_at, updated_at
		FROM short_urls
		WHERE id = $1`, id).
		Scan(&s.ID, &s.Hash, &s.Url, &s.Expiration, &s.CreatedAt, &s.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (gateway *SQLCShortUrlGateway) FindByHash(hash string) (*entity.ShortUrl, error) {
	var s entity.ShortUrl
	err := gateway.DB.QueryRow(`
		SELECT id, hash, url, expiration, created_at, updated_at
		FROM short_urls
		WHERE hash = $1`, hash).
		Scan(&s.ID, &s.Hash, &s.Url, &s.Expiration, &s.CreatedAt, &s.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (gateway *SQLCShortUrlGateway) Create(shortURL entity.ShortUrl) (*entity.ShortUrl, error) {
	shortURL.ID = uuid.New().String()
	now := time.Now().UTC().Format(timeLayout)
	shortURL.CreatedAt = now
	shortURL.UpdatedAt = now

	_, err := gateway.DB.Exec(`
		INSERT INTO short_urls (id, hash, url, expiration, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		shortURL.ID, shortURL.Hash, shortURL.Url, shortURL.Expiration,
		shortURL.CreatedAt, shortURL.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &shortURL, nil
}

func (gateway *SQLCShortUrlGateway) UpdateByHash(hash string, updated entity.ShortUrl) (*entity.ShortUrl, error) {
	updated.UpdatedAt = time.Now().UTC().Format(timeLayout)

	_, err := gateway.DB.Exec(`
		UPDATE short_urls
		SET url = $1, expiration = $2, hash = $3, updated_at = $4
		WHERE hash = $5`,
		updated.Url, updated.Expiration, updated.Hash, updated.UpdatedAt, hash)
	if err != nil {
		return nil, err
	}

	updated.Hash = hash
	return &updated, nil
}

func (gateway *SQLCShortUrlGateway) UpdateByID(id string, updated entity.ShortUrl) (*entity.ShortUrl, error) {
	updated.UpdatedAt = time.Now().UTC().Format(timeLayout)

	_, err := gateway.DB.Exec(`
		UPDATE short_urls
		SET url = $1, expiration = $2, hash = $3, updated_at = $4
		WHERE id = $5`,
		updated.Url, updated.Expiration, updated.Hash, updated.UpdatedAt, id)
	if err != nil {
		return nil, err
	}

	updated.ID = id
	return &updated, nil
}

func (gateway *SQLCShortUrlGateway) ClearAllByExpiration() error {
	_, err := gateway.DB.Exec(`
		DELETE FROM short_urls
		WHERE expiration < NOW()`)
	return err
}

func (gateway *SQLCShortUrlGateway) DeleteByID(id string) error {
	_, err := gateway.DB.Exec(`DELETE FROM short_urls WHERE id = $1`, id)
	return err
}

func (gateway *SQLCShortUrlGateway) DeleteByHash(hash string) error {
	_, err := gateway.DB.Exec(`DELETE FROM short_urls WHERE hash = $1`, hash)
	return err
}

// CountAll returns the total count of short URLs
func (gateway *SQLCShortUrlGateway) CountAll() (int64, error) {
	var count int64
	err := gateway.DB.QueryRow(`SELECT COUNT(*) FROM short_urls`).Scan(&count)
	return count, err
}

// CountByURLPart returns the count of short URLs that contain the specified URL part
func (gateway *SQLCShortUrlGateway) CountByURLPart(urlPart string) (int64, error) {
	var count int64
	err := gateway.DB.QueryRow(`
		SELECT COUNT(*) 
		FROM short_urls 
		WHERE url ILIKE '%' || $1 || '%'`, urlPart).Scan(&count)
	return count, err
}
