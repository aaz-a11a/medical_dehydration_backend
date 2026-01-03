package repository

import (
	"dehydrotationlab2/internal/app/ds"
)

func (r *Repository) GetActiveSymptoms() ([]ds.Symptom, error) {
	var symptoms []ds.Symptom
	err := r.db.Where("is_active = ?", true).Find(&symptoms).Error
	if err != nil {
		return nil, err
	}
	return symptoms, nil
}

func (r *Repository) SearchSymptoms(query string) ([]ds.Symptom, error) {
	var symptoms []ds.Symptom
	err := r.db.Where("is_active = ? AND (LOWER(title) LIKE ? OR LOWER(description) LIKE ?)",
		true, "%"+query+"%", "%"+query+"%").Find(&symptoms).Error
	if err != nil {
		return nil, err
	}
	return symptoms, nil
}

func (r *Repository) GetSymptom(id uint) (ds.Symptom, error) {
	var symptom ds.Symptom
	err := r.db.Where("id = ? AND is_active = ?", id, true).First(&symptom).Error
	if err != nil {
		return ds.Symptom{}, err
	}
	return symptom, nil
}
