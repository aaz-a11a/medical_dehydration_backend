package repository

import (
	"dehydrotationlab3/internal/app/ds"
	"time"

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
	err := r.db.Where("user_id = ? AND status NOT IN ?", userID, []string{"удален", "удалён", "черновик"}).Order("id DESC").Find(&requests).Error
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
		Where("request_symptoms.request_id = ? AND symptoms.is_active = ?", requestID, true).
		Find(&symptoms).Error
	return symptoms, err
}

type RequestWithSymptoms struct {
	Request  ds.DehydrationRequest
	Symptoms []ds.Symptom
}

func (r *Repository) GetRequestWithSymptoms(id uint) (*RequestWithSymptoms, error) {
	req, err := r.GetRequestByID(id)
	if err != nil {
		return nil, err
	}
	syms, err := r.GetRequestSymptoms(id)
	if err != nil {
		return nil, err
	}
	return &RequestWithSymptoms{Request: *req, Symptoms: syms}, nil
}

func (r *Repository) CountRequestSymptoms(requestID uint) (int64, error) {
	var count int64
	if err := r.db.Model(&ds.RequestSymptom{}).Where("request_id = ?", requestID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountRequestSymptomsWithComment считает M-M записи с непустым комментарием
func (r *Repository) CountRequestSymptomsWithComment(requestID uint) (int64, error) {
	var count int64
	if err := r.db.Model(&ds.RequestSymptom{}).
		Where("request_id = ? AND TRIM(COALESCE(comment, '')) <> ''", requestID).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *Repository) SetFormed(id uint, when time.Time) error {
	return r.db.Model(&ds.DehydrationRequest{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":    "сформирован",
		"formed_at": when,
	}).Error
}

func (r *Repository) SetCompleted(id uint, moderatorID uint, status string, when time.Time, patientWeight *float64, dehydrationPercent *float64, fluidDeficit *float64, doctorComment *string) error {
	updates := map[string]interface{}{
		"status":       status,
		"completed_at": when,
		"moderator_id": moderatorID,
	}
	if patientWeight != nil {
		updates["patient_weight"] = *patientWeight
	}
	if dehydrationPercent != nil {
		updates["dehydration_percent"] = *dehydrationPercent
	}
	if fluidDeficit != nil {
		updates["fluid_deficit"] = *fluidDeficit
	}
	if doctorComment != nil {
		updates["doctor_comment"] = *doctorComment
	}
	return r.db.Model(&ds.DehydrationRequest{}).Where("id = ?", id).Updates(updates).Error
}

func (r *Repository) IsRequestOwner(userID, requestID uint) (bool, error) {
	var cnt int64
	if err := r.db.Model(&ds.DehydrationRequest{}).Where("id = ? AND user_id = ?", requestID, userID).Count(&cnt).Error; err != nil {
		return false, err
	}
	return cnt > 0, nil
}

// ListRequestsWithFilters фильтрует по статусу и диапазону дат формирования (formed_at)
func (r *Repository) ListRequestsWithFilters(userID uint, status *string, from, to *string) ([]ds.DehydrationRequest, error) {
	var list []ds.DehydrationRequest
	tx := r.db.Model(&ds.DehydrationRequest{}).
		Where("user_id = ? AND status NOT IN ?", userID, []string{"удален", "удалён", "черновик"}).
		Preload("User").
		Preload("Moderator")
	if status != nil && *status != "" {
		tx = tx.Where("status = ?", *status)
	}
	if from != nil && *from != "" {
		tx = tx.Where("formed_at >= ?", *from)
	}
	if to != nil && *to != "" {
		tx = tx.Where("formed_at <= ?", *to)
	}
	if err := tx.Order("id DESC").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// UpdateRequestFields изменяет разрешенные поля темы (например, patient_weight, doctor_comment)
func (r *Repository) UpdateRequestFields(id uint, fields map[string]interface{}) error {
	return r.db.Model(&ds.DehydrationRequest{}).Where("id = ?", id).Updates(fields).Error
}

// ChangeRequestStatus меняет статус с проверкой допустимых переходов вне этого слоя (проверять в handler)
func (r *Repository) ChangeRequestStatus(id uint, status string) error {
	return r.db.Model(&ds.DehydrationRequest{}).Where("id = ?", id).Update("status", status).Error
}

// GetLastCompletedRequest возвращает последнюю завершённую заявку пользователя
func (r *Repository) GetLastCompletedRequest(userID uint) (*ds.DehydrationRequest, error) {
	var req ds.DehydrationRequest
	if err := r.db.Where("user_id = ? AND status = ?", userID, "завершен").
		Order("completed_at DESC").Limit(1).Find(&req).Error; err != nil {
		return nil, err
	}
	if req.ID == 0 {
		return nil, nil
	}
	return &req, nil
}
