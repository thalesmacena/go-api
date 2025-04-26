package entity

type Todo struct {
	ID          uint   `gorm:"primaryKey"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Tasks       []Task `json:"tasks" gorm:"foreignKey:TodoID"`
	CreatedAt   string `json:"creationDate"`
	UpdatedAt   string `json:"updateDate"`
}
