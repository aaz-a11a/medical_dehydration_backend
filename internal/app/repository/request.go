package repository

import (
	"dehydrotationlab2/internal/app/ds"

	"gorm.io/gorm"
)

func (r *Repository) GetOrCreateDraftRequest(userID uint) (*ds.DehydrationRequest, error) {
	var request ds.DehydrationRequest
	err := r.db.Where("user_id = ? AND status = ?", userID, "черновик").First(&request).Error
	if err == nil {
		return &request, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	request = ds.DehydrationRequest{
		UserID: userID,
		Status: "черновик",
	}
	err = r.db.Create(&request).Error
	if err != nil {
		return nil, err
	}
	return &request, nil
}

func (r *Repository) AddSymptomToRequest(requestID, symptomID uint) error {
	requestSymptom := ds.RequestSymptom{
		RequestID: requestID,
		SymptomID: symptomID,
	}
	err := r.db.Create(&requestSymptom).Error
	if err != nil {
		// Ignore duplicate key errors
		return nil
	}
	return nil
}

func (r *Repository) DeleteRequest(requestID uint) error {

	result := r.db.Exec("UPDATE dehydration_requests SET status = 'удалён' WHERE id = ?", requestID)
	return result.Error
}

func (r *Repository) GetDraftRequest(userID uint) (*ds.DehydrationRequest, error) {
	var request ds.DehydrationRequest
	err := r.db.Where("user_id = ? AND status = ?", userID, "черновик").First(&request).Error
	if err != nil {
		return nil, err
	}
	return &request, nil
}

func (r *Repository) GetDraftSymptoms(userID uint) ([]ds.Symptom, *ds.DehydrationRequest, error) {
	request, err := r.GetDraftRequest(userID)
	if err != nil {
		return nil, nil, err
	}

	var symptoms []ds.Symptom
	err = r.db.Table("symptoms").
		Joins("JOIN request_symptoms ON request_symptoms.symptom_id = symptoms.id").
		Where("request_symptoms.request_id = ? AND symptoms.is_active = ?", request.ID, true).
		Find(&symptoms).Error
	if err != nil {
		return nil, nil, err
	}

	return symptoms, request, nil
}

func (r *Repository) CountDraftItems(userID uint) (int64, error) {
	request, err := r.GetDraftRequest(userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil
		}
		return 0, err
	}

	var count int64
	err = r.db.Model(&ds.RequestSymptom{}).Where("request_id = ?", request.ID).Count(&count).Error
	return count, err
}

func (r *Repository) ListRequestsByUser(userID uint) ([]ds.DehydrationRequest, error) {
	var requests []ds.DehydrationRequest
	err := r.db.Where("user_id = ?", userID).Order("id DESC").Find(&requests).Error
	return requests, err
}

func (r *Repository) GetRequestByID(id uint) (*ds.DehydrationRequest, error) {
	var request ds.DehydrationRequest
	err := r.db.Where("id = ?", id).First(&request).Error
	if err != nil {
		return nil, err
	}
	return &request, nil
}

func (r *Repository) GetRequestSymptoms(requestID uint) ([]ds.Symptom, error) {
	var symptoms []ds.Symptom
	err := r.db.Table("symptoms").
		Joins("JOIN request_symptoms ON request_symptoms.symptom_id = symptoms.id").
		Where("request_symptoms.request_id = ?", requestID).
		Find(&symptoms).Error
	if err != nil {
		return nil, err
	}
	return symptoms, nil
}
