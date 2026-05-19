package model

import "time"

type Department struct {
	ID        uint      `gorm:"primaryKey"         json:"id"`
	Name      string    `gorm:"not null;size:200"  json:"name"`
	ParentID  *uint     `gorm:"index"              json:"parent_id"`
	Depth     int       `gorm:"not null;default:1" json:"depth"`
	CreatedAt time.Time `                          json:"created_at"`

	Parent    *Department  `gorm:"foreignKey:ParentID;constraint:OnDelete:CASCADE" json:"parent,omitempty"`
	Children  []Department `gorm:"foreignKey:ParentID;constraint:OnDelete:CASCADE" json:"children,omitempty"`
	Employees []Employee   `gorm:"foreignKey:DepartmentID;constraint:OnDelete:CASCADE" json:"employees,omitempty"`
}
