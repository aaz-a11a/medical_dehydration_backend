package ds

type Symptom struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	Title        string `gorm:"type:varchar(100);not null" json:"title"`
	Category     string `gorm:"type:varchar(100)" json:"category"`
	Description  string `gorm:"type:text" json:"description"`
	Severity     string `gorm:"type:varchar(50)" json:"severity"`
	WeightLoss   string `gorm:"type:varchar(50)" json:"weight_loss"`
	FluidNeed    string `gorm:"type:varchar(50)" json:"fluid_need"`
	RecoveryTime string `gorm:"type:varchar(50)" json:"recovery_time"`
	ImageURL     string `gorm:"type:varchar(200)" json:"image_url"`
	IsActive     bool   `gorm:"type:boolean;default:true" json:"is_active"`
}
