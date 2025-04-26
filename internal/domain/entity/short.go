package entity

type ShortUrl struct {
	ID         string `json:"id"`
	Hash       string `json:"hash"`
	Url        string `json:"url"`
	Expiration string `json:"expiration"`
	CreatedAt  string `json:"createdDate"`
	UpdatedAt  string `json:"updatedDate"`
}
