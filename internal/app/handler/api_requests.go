package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"dehydrotationlab3/internal/app/ds"

	"github.com/gin-gonic/gin"
)

// GET /api/requests/cart
func (h *Handler) ApiGetCart(ctx *gin.Context) {
	count, _ := h.Repository.CountDraftItems(CurrentUserID())
	draft, _ := h.Repository.GetDraftRequest(CurrentUserID())
	var draftID *uint
	if draft != nil {
		draftID = &draft.ID
	}
	jsonResponse(ctx, gin.H{"draft_id": draftID, "count": count}, 1, gin.H{})
}

// GET /api/requests
func (h *Handler) ApiListRequests(ctx *gin.Context) {
	status := ctx.Query("status")
	from := ctx.Query("from")
	to := ctx.Query("to")
	var statusPtr *string
	if status != "" {
		statusPtr = &status
	}
	list, err := h.Repository.ListRequestsWithFilters(CurrentUserID(), statusPtr, &from, &to)
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	// Плоская структура без вложенных объектов
	type requestItem struct {
		ID                 uint       `json:"id"`
		UserID             uint       `json:"user_id"`
		UserLogin          string     `json:"user_login"`
		Status             string     `json:"status"`
		CreatedAt          time.Time  `json:"created_at"`
		FormedAt           *time.Time `json:"formed_at"`
		CompletedAt        *time.Time `json:"completed_at"`
		ModeratorID        *uint      `json:"moderator_id"`
		ModeratorLogin     *string    `json:"moderator_login,omitempty"`
		PatientWeight      *float64   `json:"patient_weight"`
		DehydrationPercent *float64   `json:"dehydration_percent"`
		FluidDeficit       *float64   `json:"fluid_deficit"`
		DoctorComment      string     `json:"doctor_comment,omitempty"`
		CommentsCount      int64      `json:"comments_count"`
		Result             *float64   `json:"result"`
	}

	resp := make([]requestItem, 0, len(list))
	for _, r := range list {
		cnt, _ := h.Repository.CountRequestSymptomsWithComment(r.ID)
		var res *float64
		if r.PatientWeight != nil && r.DehydrationPercent != nil {
			v := (*r.PatientWeight) * (*r.DehydrationPercent) * 0.01
			res = &v
		} else if r.FluidDeficit != nil {
			res = r.FluidDeficit
		}

		doctorComment := ""
		if r.DoctorComment.Valid {
			doctorComment = r.DoctorComment.String
		}

		var moderatorLogin *string
		if r.Moderator != nil {
			moderatorLogin = &r.Moderator.Login
		}

		resp = append(resp, requestItem{
			ID:                 r.ID,
			UserID:             r.UserID,
			UserLogin:          r.User.Login,
			Status:             r.Status,
			CreatedAt:          r.CreatedAt,
			FormedAt:           r.FormedAt,
			CompletedAt:        r.CompletedAt,
			ModeratorID:        r.ModeratorID,
			ModeratorLogin:     moderatorLogin,
			PatientWeight:      r.PatientWeight,
			DehydrationPercent: r.DehydrationPercent,
			FluidDeficit:       r.FluidDeficit,
			DoctorComment:      doctorComment,
			CommentsCount:      cnt,
			Result:             res,
		})
	}
	jsonResponse(ctx, resp, int64(len(resp)), gin.H{"status": status, "from": from, "to": to})
}

// GET /api/requests/:id
func (h *Handler) ApiGetRequest(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}
	dr, err := h.Repository.GetRequestWithSymptoms(uint(id))
	if err != nil || dr == nil {
		h.errorHandler(ctx, http.StatusNotFound, err)
		return
	}
	// скрываем удаленные
	if dr.Request.Status == "удален" || dr.Request.Status == "удалён" {
		ctx.Status(http.StatusNotFound)
		return
	}

	// Формируем ответ с результатом
	type responseT struct {
		ID                 uint         `json:"id"`
		UserID             uint         `json:"user_id"`
		User               ds.User      `json:"user"`
		Status             string       `json:"status"`
		CreatedAt          time.Time    `json:"created_at"`
		FormedAt           *time.Time   `json:"formed_at"`
		CompletedAt        *time.Time   `json:"completed_at"`
		PatientWeight      *float64     `json:"patient_weight"`
		DehydrationPercent *float64     `json:"dehydration_percent"`
		FluidDeficit       *float64     `json:"fluid_deficit"`
		DoctorComment      string       `json:"doctor_comment,omitempty"`
		Symptoms           []ds.Symptom `json:"symptoms"`
	}

	doctorComment := ""
	if dr.Request.DoctorComment.Valid {
		doctorComment = dr.Request.DoctorComment.String
	}

	resp := responseT{
		ID:                 dr.Request.ID,
		UserID:             dr.Request.UserID,
		User:               dr.Request.User,
		Status:             dr.Request.Status,
		CreatedAt:          dr.Request.CreatedAt,
		FormedAt:           dr.Request.FormedAt,
		CompletedAt:        dr.Request.CompletedAt,
		PatientWeight:      dr.Request.PatientWeight,
		DehydrationPercent: dr.Request.DehydrationPercent,
		FluidDeficit:       dr.Request.FluidDeficit,
		DoctorComment:      doctorComment,
		Symptoms:           dr.Symptoms,
	}

	jsonResponse(ctx, resp, 1, gin.H{"id": id})
}

// PUT /api/requests/:id
func (h *Handler) ApiUpdateRequest(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id64, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}
	id := uint(id64)
	// только владелец
	if owner, err := h.Repository.IsRequestOwner(CurrentUserID(), id); err != nil || !owner {
		h.errorHandler(ctx, http.StatusForbidden, err)
		return
	}
	rq, err := h.Repository.GetRequestByID(id)
	if err != nil {
		h.errorHandler(ctx, http.StatusNotFound, err)
		return
	}
	if rq.Status != "черновик" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "only draft can be updated"})
		return
	}
	type bodyT struct {
		PatientWeight *float64 `json:"patient_weight"`
		DoctorComment *string  `json:"doctor_comment"`
	}
	var body bodyT
	if err := ctx.ShouldBindJSON(&body); err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}
	fields := map[string]interface{}{}
	if body.PatientWeight != nil {
		fields["patient_weight"] = *body.PatientWeight
	}
	if body.DoctorComment != nil {
		fields["doctor_comment"] = *body.DoctorComment
	}
	if err := h.Repository.UpdateRequestFields(id, fields); err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}
	rq, _ = h.Repository.GetRequestByID(id)
	jsonResponse(ctx, rq, 1, gin.H{"id": id})
}

// PUT /api/requests/:id/form — сформировать заявку
func (h *Handler) ApiFormRequest(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id64, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}
	id := uint(id64)
	// только владелец может формировать черновик
	if owner, err := h.Repository.IsRequestOwner(CurrentUserID(), id); err != nil || !owner {
		h.errorHandler(ctx, http.StatusForbidden, err)
		return
	}
	rq, err := h.Repository.GetRequestByID(id)
	if err != nil {
		h.errorHandler(ctx, http.StatusNotFound, err)
		return
	}
	if rq.Status != "черновик" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "only draft can be formed"})
		return
	}
	// обязательные поля: есть хотя бы один симптом
	cnt, err := h.Repository.CountRequestSymptoms(id)
	if err != nil || cnt == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "empty request"})
		return
	}
	if err := h.Repository.SetFormed(id, time.Now()); err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}
	rq, _ = h.Repository.GetRequestByID(id)
	jsonResponse(ctx, rq, 1, gin.H{"id": id})
}

// PUT /api/requests/:id/complete — завершить/отклонить
func (h *Handler) ApiCompleteRequest(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id64, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}
	id := uint(id64)
	type bodyT struct {
		Status             string   `json:"status" binding:"required"` // завершен | отклонен
		PatientWeight      *float64 `json:"patient_weight"`
		DehydrationPercent *float64 `json:"dehydration_percent"`
		DoctorComment      *string  `json:"doctor_comment"`
	}
	var body bodyT
	if err := ctx.ShouldBindJSON(&body); err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}
	if body.Status != "завершен" && body.Status != "отклонен" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid status"})
		return
	}
	rq, err := h.Repository.GetRequestByID(id)
	if err != nil {
		h.errorHandler(ctx, http.StatusNotFound, err)
		return
	}
	// проверка прав модератора
	u, err := h.Repository.GetUserByID(CurrentUserID())
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}
	if !u.IsModerator {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "only moderator can complete/reject"})
		return
	}
	if rq.Status != "сформирован" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "only formed can be completed"})
		return
	}
	var fluidDeficit *float64
	var percentPtr *float64
	if body.Status == "завершен" {
		if body.PatientWeight == nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "weight required"})
			return
		}
		// Если процент не передан — оцениваем по симптомам автоматически
		if body.DehydrationPercent == nil {
			syms, err := h.Repository.GetRequestSymptoms(id)
			if err != nil {
				h.errorHandler(ctx, http.StatusInternalServerError, err)
				return
			}
			p := estimatePercentFromSymptoms(syms)
			percentPtr = &p
		} else {
			percentPtr = body.DehydrationPercent
		}
		v := (*body.PatientWeight) * (*percentPtr) * 0.01
		fluidDeficit = &v
	}
	if err := h.Repository.SetCompleted(id, CurrentUserID(), body.Status, time.Now(), body.PatientWeight, percentPtr, fluidDeficit, body.DoctorComment); err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}
	rq, _ = h.Repository.GetRequestByID(id)
	jsonResponse(ctx, rq, 1, gin.H{"id": id})
}

// estimatePercentFromSymptoms пытается оценить процент обезвоживания по полю Severity
func estimatePercentFromSymptoms(syms []ds.Symptom) float64 {
	if len(syms) == 0 {
		return 0
	}
	total := 0.0
	count := 0.0
	for _, s := range syms {
		sev := strings.ToLower(s.Severity)
		val := 0.0
		if strings.Contains(sev, "тяжел") || strings.Contains(sev, "тяжёл") {
			val = 8.0
		} else if strings.Contains(sev, "средн") {
			val = 4.5
		} else if strings.Contains(sev, "легк") || strings.Contains(sev, "лёгк") {
			val = 1.5
		}
		if val > 0 {
			total += val
			count += 1.0
		}
	}
	if count == 0 {
		return 0
	}
	return total / count
}

// DELETE /api/requests/:id
func (h *Handler) ApiDeleteRequest(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}
	// только создатель
	if owner, err := h.Repository.IsRequestOwner(CurrentUserID(), uint(id)); err != nil || !owner {
		h.errorHandler(ctx, http.StatusForbidden, err)
		return
	}
	if err := h.Repository.DeleteRequest(uint(id)); err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}
	jsonResponse(ctx, gin.H{"deleted": id}, 1, gin.H{})
}
