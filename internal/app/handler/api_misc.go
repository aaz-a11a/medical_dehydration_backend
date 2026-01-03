package handler

import (
	"net/http"

	"dehydrotationlab3/internal/app/ds"

	"github.com/gin-gonic/gin"
)

type reqSymptomKey struct {
	// RequestID опционален для добавления: если не указан, используем/создаём черновик
	RequestID uint `json:"request_id"`
	SymptomID uint `json:"symptom_id" binding:"required"`
}

// POST /api/request-symptoms — добавить симптом в заявку (черновик)
func (h *Handler) ApiAddRequestSymptom(ctx *gin.Context) {
	var body reqSymptomKey
	if err := ctx.ShouldBindJSON(&body); err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}
	var requestID uint
	if body.RequestID == 0 {
		// Автосоздание/получение черновика
		draft, err := h.Repository.GetOrCreateDraftRequest(CurrentUserID())
		if err != nil {
			h.errorHandler(ctx, http.StatusInternalServerError, err)
			return
		}
		requestID = draft.ID
	} else {
		requestID = body.RequestID
		// только владелец черновика
		if owner, err := h.Repository.IsRequestOwner(CurrentUserID(), requestID); err != nil || !owner {
			h.errorHandler(ctx, http.StatusForbidden, err)
			return
		}
		rq, err := h.Repository.GetRequestByID(requestID)
		if err != nil || rq.Status != "черновик" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "only draft can be modified"})
			return
		}
	}

	if err := h.Repository.AddSymptomToRequest(requestID, body.SymptomID); err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}
	jsonResponse(ctx, gin.H{"added": body.SymptomID}, 1, gin.H{"request_id": requestID})
}

// DELETE /api/request-symptoms
func (h *Handler) ApiDeleteRequestSymptom(ctx *gin.Context) {
	var body reqSymptomKey
	if err := ctx.ShouldBindJSON(&body); err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}
	// только владелец черновика может менять
	if owner, err := h.Repository.IsRequestOwner(CurrentUserID(), body.RequestID); err != nil || !owner {
		h.errorHandler(ctx, http.StatusForbidden, err)
		return
	}
	rq, err := h.Repository.GetRequestByID(body.RequestID)
	if err != nil || rq.Status != "черновик" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "only draft can be modified"})
		return
	}
	if err := h.Repository.DeleteRequestSymptom(body.RequestID, body.SymptomID); err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}
	jsonResponse(ctx, gin.H{"deleted": body}, 1, gin.H{})
}

// PUT /api/request-symptoms
func (h *Handler) ApiUpdateRequestSymptom(ctx *gin.Context) {
	type bodyT struct {
		RequestID uint    `json:"request_id" binding:"required"`
		SymptomID uint    `json:"symptom_id" binding:"required"`
		Intensity *int    `json:"intensity"`
		Comment   *string `json:"comment"`
		IsMain    *bool   `json:"is_main"`
	}
	var body bodyT
	if err := ctx.ShouldBindJSON(&body); err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}
	if owner, err := h.Repository.IsRequestOwner(CurrentUserID(), body.RequestID); err != nil || !owner {
		h.errorHandler(ctx, http.StatusForbidden, err)
		return
	}
	rq, err := h.Repository.GetRequestByID(body.RequestID)
	if err != nil || rq.Status != "черновик" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "only draft can be modified"})
		return
	}
	if err := h.Repository.UpdateRequestSymptom(body.RequestID, body.SymptomID, body.Intensity, body.Comment, body.IsMain); err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}
	jsonResponse(ctx, gin.H{"updated": body.SymptomID}, 1, gin.H{"request_id": body.RequestID})
}

// Users
func (h *Handler) ApiRegisterUser(ctx *gin.Context) {
	type bodyT struct {
		Login    string `json:"login" binding:"required"`
		Password string `json:"password" binding:"required"`
		// Разрешим флаг модератора только для простоты тестов (в реальной жизни это через роли)
		IsModerator *bool `json:"is_moderator"`
	}
	var body bodyT
	if err := ctx.ShouldBindJSON(&body); err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}
	// Проверка уникальности логина
	if u, _ := h.Repository.GetUserByLogin(body.Login); u != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "login already exists"})
		return
	}
	// Хеширование пароля (упрощенно: без внешних зависимостей можно оставить как есть, но лучше захешировать)
	// Здесь оставим пароль как есть в password_hash для простоты, авторизации всё равно нет
	newUser := ds.User{Login: body.Login, Password: body.Password}
	if body.IsModerator != nil {
		newUser.IsModerator = *body.IsModerator
	}
	if err := h.Repository.CreateUser(&newUser); err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}
	jsonResponse(ctx, gin.H{"id": newUser.ID, "login": newUser.Login, "is_moderator": newUser.IsModerator}, 1, gin.H{})
}
func (h *Handler) ApiGetProfile(ctx *gin.Context) {
	u, err := h.Repository.GetUserByID(CurrentUserID())
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}
	// Добавим поле result по последней завершенной заявке
	var result *float64
	if last, err := h.Repository.GetLastCompletedRequest(CurrentUserID()); err == nil && last != nil {
		if last.FluidDeficit != nil {
			result = last.FluidDeficit
		} else if last.PatientWeight != nil && last.DehydrationPercent != nil {
			v := (*last.PatientWeight) * (*last.DehydrationPercent) * 0.01
			result = &v
		}
	}
	type respUser struct {
		ds.User `json:",inline"`
		Result  *float64 `json:"result"`
	}
	jsonResponse(ctx, respUser{User: *u, Result: result}, 1, gin.H{})
}
func (h *Handler) ApiUpdateProfile(ctx *gin.Context) {
	type bodyT struct {
		Login       *string `json:"login"`
		IsModerator *bool   `json:"is_moderator"`
	}
	var body bodyT
	if err := ctx.ShouldBindJSON(&body); err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}
	fields := map[string]interface{}{}
	if body.Login != nil {
		fields["login"] = *body.Login
	}
	if body.IsModerator != nil {
		fields["is_moderator"] = *body.IsModerator
	}
	if err := h.Repository.UpdateUser(CurrentUserID(), fields); err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}
	jsonResponse(ctx, gin.H{"updated": true}, 1, gin.H{})
}
func (h *Handler) ApiLogin(ctx *gin.Context) {
	jsonResponse(ctx, gin.H{"user_id": CurrentUserID()}, 1, gin.H{})
}
func (h *Handler) ApiLogout(ctx *gin.Context) { jsonResponse(ctx, gin.H{"logout": true}, 1, gin.H{}) }
