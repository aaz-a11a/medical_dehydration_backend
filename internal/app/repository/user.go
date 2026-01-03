package repository

import (
	"dehydrotationlab4/internal/app/ds"
)

func (r *Repository) GetUserByID(id uint) (*ds.User, error) {
	var u ds.User
	if err := r.db.First(&u, id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repository) GetUserByLogin(login string) (*ds.User, error) {
	var u ds.User
	if err := r.db.Where("login = ?", login).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repository) CreateUser(u *ds.User) error {
	return r.db.Create(u).Error
}

func (r *Repository) UpdateUser(id uint, fields map[string]interface{}) error {
	return r.db.Model(&ds.User{}).Where("id = ?", id).Updates(fields).Error
}
