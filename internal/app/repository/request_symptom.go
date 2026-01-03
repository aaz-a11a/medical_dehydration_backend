package repository

import "dehydrotationlab4/internal/app/ds"

func (r *Repository) DeleteRequestSymptom(requestID, symptomID uint) error {
	return r.db.Where("request_id = ? AND symptom_id = ?", requestID, symptomID).Delete(&ds.RequestSymptom{}).Error
}

func (r *Repository) UpdateRequestSymptom(requestID, symptomID uint, intensity *int, comment *string, isMain *bool) error {
	updates := map[string]interface{}{}
	if intensity != nil {
		updates["intensity"] = *intensity
	}
	if comment != nil {
		updates["comment"] = *comment
	}
	if isMain != nil {
		updates["is_main"] = *isMain
	}
	if len(updates) == 0 {
		return nil
	}
	return r.db.Model(&ds.RequestSymptom{}).Where("request_id = ? AND symptom_id = ?", requestID, symptomID).Updates(updates).Error
}
