package model

type CreateShortUrlDTO struct {
	Url        string `json:"url"`
	Expiration string `json:"expiration"`
}

type UpdateShortUrlDTO struct {
	Url        string `json:"url"`
	Expiration string `json:"expiration"`
	Hash       string `json:"hash"`
}
