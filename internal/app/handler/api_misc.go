package handler

import (
	"net/http"

	"dehydrotationlab4/internal/app/middleware"

	"github.com/gin-gonic/gin"
)

type reqSymptomKey struct {
	// RequestID опционален для добавления: если не указан, используем/создаём черновик
	RequestID uint `json:"request_id"`
	SymptomID uint `json:"symptom_id" binding:"required"`
}

// ApiAddRequestSymptom добавить симптом в заявку
// @Summary Добавить симптом в заявку
// @Description Требуется авторизация. Добавляет симптом в черновик заявки. Если request_id не указан, создается/используется текущий черновик.
// @Tags request-symptoms
// @Security BearerAuth
// @Security CookieAuth
// @Accept json
// @Produce json
// @Param request_symptom body object{request_id=uint,symptom_id=uint} true "Данные для добавления"
// @Success 200 {object} object{data=object{added=uint},total=int,filters=object}
// @Failure 400 {object} object{error=string}
// @Failure 401 {object} object{error=string}
// @Failure 403 {object} object{error=string}
// @Router /api/request-symptoms [post]
func (h *Handler) ApiAddRequestSymptom(ctx *gin.Context) {
	userID, ok := middleware.GetCurrentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var body reqSymptomKey
	if err := ctx.ShouldBindJSON(&body); err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}
	var requestID uint
	if body.RequestID == 0 {
		// Автосоздание/получение черновика
		draft, err := h.Repository.GetOrCreateDraftRequest(userID)
		if err != nil {
			h.errorHandler(ctx, http.StatusInternalServerError, err)
			return
		}
		requestID = draft.ID
	} else {
		requestID = body.RequestID
		// только владелец черновика
		if owner, err := h.Repository.IsRequestOwner(userID, requestID); err != nil || !owner {
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

// ApiDeleteRequestSymptom удалить симптом из заявки
// @Summary Удалить симптом из заявки
// @Description Требуется авторизация. Удаляет симптом из черновика заявки. Доступно только владельцу.
// @Tags request-symptoms
// @Security BearerAuth
// @Security CookieAuth
// @Accept json
// @Produce json
// @Param request_symptom body object{request_id=uint,symptom_id=uint} true "Данные для удаления"
// @Success 200 {object} object{data=object{deleted=object},total=int,filters=object}
// @Failure 400 {object} object{error=string}
// @Failure 401 {object} object{error=string}
// @Failure 403 {object} object{error=string}
// @Router /api/request-symptoms [delete]
func (h *Handler) ApiDeleteRequestSymptom(ctx *gin.Context) {
	userID, ok := middleware.GetCurrentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var body reqSymptomKey
	if err := ctx.ShouldBindJSON(&body); err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}
	// только владелец черновика может менять
	if owner, err := h.Repository.IsRequestOwner(userID, body.RequestID); err != nil || !owner {
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

// ApiUpdateRequestSymptom обновить данные симптома в заявке
// @Summary Обновить данные симптома в заявке
// @Description Требуется авторизация. Обновляет данные связи заявка-симптом (интенсивность, комментарий, признак основного). Доступно только владельцу черновика.
// @Tags request-symptoms
// @Security BearerAuth
// @Security CookieAuth
// @Accept json
// @Produce json
// @Param request_symptom body object{request_id=uint,symptom_id=uint,intensity=int,comment=string,is_main=bool} true "Данные для обновления"
// @Success 200 {object} object{data=object{updated=uint},total=int,filters=object}
// @Failure 400 {object} object{error=string}
// @Failure 401 {object} object{error=string}
// @Failure 403 {object} object{error=string}
// @Router /api/request-symptoms [put]
func (h *Handler) ApiUpdateRequestSymptom(ctx *gin.Context) {
	userID, ok := middleware.GetCurrentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

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
	if owner, err := h.Repository.IsRequestOwner(userID, body.RequestID); err != nil || !owner {
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
