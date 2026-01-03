package ds

type RequestSymptom struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	RequestID uint   `gorm:"not null;uniqueIndex:idx_request_symptom" json:"request_id"`
	SymptomID uint   `gorm:"not null;uniqueIndex:idx_request_symptom" json:"symptom_id"`
	Intensity *int   `gorm:"type:integer;check:intensity BETWEEN 1 AND 10" json:"intensity"`
	IsMain    bool   `gorm:"type:boolean;default:false" json:"is_main"`
	Comment   string `gorm:"type:text" json:"comment"`

	Request DehydrationRequest `gorm:"foreignKey:RequestID" json:"request"`
	Symptom Symptom            `gorm:"foreignKey:SymptomID" json:"symptom"`
}
