package models

// Category groups products in the catalog.
// Code is the human-readable identifier exposed by the API; ID stays internal.
type Category struct {
	ID   uint   `gorm:"primaryKey"`
	Code string `gorm:"uniqueIndex;not null"`
	Name string `gorm:"not null"`
}

func (c *Category) TableName() string {
	return "categories"
}
