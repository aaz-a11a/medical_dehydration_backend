package repository

import (
	"dehydrotationlab3/internal/app/ds"
	"strings"
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
	q := strings.ToLower(query)
	err := r.db.Where("is_active = ? AND (LOWER(title) LIKE ? OR LOWER(description) LIKE ?)",
		true, "%"+q+"%", "%"+q+"%").Find(&symptoms).Error
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

func (r *Repository) GetSymptomAny(id uint) (ds.Symptom, error) {
	var symptom ds.Symptom
	err := r.db.Where("id = ?", id).First(&symptom).Error
	if err != nil {
		return ds.Symptom{}, err
	}
	return symptom, nil
}

// FilterSymptoms фильтрует по названию (LIKE по title/description) и активности (если задана)
func (r *Repository) FilterSymptoms(title string, active *bool) ([]ds.Symptom, error) {
	var symptoms []ds.Symptom
	tx := r.db.Model(&ds.Symptom{})
	if active != nil {
		tx = tx.Where("is_active = ?", *active)
	}
	if title != "" {
		q := strings.ToLower(title)
		tx = tx.Where("LOWER(title) LIKE ? OR LOWER(description) LIKE ?", "%"+q+"%", "%"+q+"%")
	}
	if err := tx.Find(&symptoms).Error; err != nil {
		return nil, err
	}
	return symptoms, nil
}

func (r *Repository) CreateSymptom(s *ds.Symptom) error {
	return r.db.Create(s).Error
}

func (r *Repository) UpdateSymptom(id uint, upd ds.Symptom) (ds.Symptom, error) {
	var s ds.Symptom
	if err := r.db.First(&s, id).Error; err != nil {
		return s, err
	}
	// системные поля не трогаем: ID, ImageURL меняется отдельным методом
	s.Title = upd.Title
	s.Category = upd.Category
	s.Description = upd.Description
	s.Severity = upd.Severity
	s.WeightLoss = upd.WeightLoss
	s.FluidNeed = upd.FluidNeed
	s.RecoveryTime = upd.RecoveryTime
	s.IsActive = upd.IsActive
	if err := r.db.Save(&s).Error; err != nil {
		return s, err
	}
	return s, nil
}

func (r *Repository) UpdateSymptomImage(id uint, key string) error {
	return r.db.Model(&ds.Symptom{}).Where("id = ?", id).Update("image_url", key).Error
}

func (r *Repository) DeleteSymptom(id uint) error {
	// Мягкое удаление: помечаем симптом как неактивный, чтобы не нарушать FK в request_symptoms
	return r.db.Model(&ds.Symptom{}).Where("id = ?", id).Update("is_active", false).Error
}
