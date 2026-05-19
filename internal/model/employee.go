package model

import "time"

type Employee struct {
	ID           uint       `gorm:"primaryKey"        json:"id"`
	DepartmentID uint       `gorm:"not null;index"    json:"department_id"`
	Fullname     string     `gorm:"not null;size:200" json:"full_name"`
	Position     string     `gorm:"not null;size:200" json:"position"`
	HiredAt      *time.Time `gorm:"type:date"         json:"hired_at"`
	CreatedAt    time.Time  `                         json:"created_at"`

	Department *Department `gorm:"foreignKey:DepartmentID;constraint:OnDelete:CASCADE" json:"department,omitempty"`
}
