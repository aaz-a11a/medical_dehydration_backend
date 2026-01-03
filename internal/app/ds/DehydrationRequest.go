package ds

import (
	"database/sql"
	"time"
)

type DehydrationRequest struct {
	ID                 uint           `gorm:"primaryKey" json:"id"`
	UserID             uint           `gorm:"not null" json:"user_id"`
	Status             string         `gorm:"type:varchar(20);not null" json:"status"`
	CreatedAt          time.Time      `gorm:"not null" json:"created_at"`
	FormedAt           *time.Time     `json:"formed_at"`
	CompletedAt        *time.Time     `json:"completed_at"`
	ModeratorID        *uint          `json:"moderator_id"`
	PatientWeight      *float64       `gorm:"type:decimal(5,2)" json:"patient_weight"`
	DehydrationPercent *float64       `gorm:"type:decimal(4,2)" json:"dehydration_percent"`
	FluidDeficit       *float64       `gorm:"type:decimal(6,2)" json:"fluid_deficit"`
	DoctorComment      sql.NullString `gorm:"type:text" json:"doctor_comment"`

	User      User  `gorm:"foreignKey:UserID" json:"user"`
	Moderator *User `gorm:"foreignKey:ModeratorID" json:"moderator"`
}
