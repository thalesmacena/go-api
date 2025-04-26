package entity

type Task struct {
	ID        uint   `gorm:"primaryKey"`
	TodoID    uint   `json:"-"`
	Name      string `json:"name"`
	Done      bool   `json:"done"`
	DoneDate  string `json:"doneDate"`
	CreatedAt string `json:"createdDate"`
	UpdatedAt string `json:"updatedDate"`
}
