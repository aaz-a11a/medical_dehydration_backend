package ds

type User struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	Login       string `gorm:"type:varchar(100);unique;not null" json:"login"`
	Password    string `gorm:"column:password_hash;type:varchar(255)" json:"-"`
	IsModerator bool   `gorm:"type:boolean;default:false" json:"-"`
}
