package handler

import (
	"net/http"
	"strconv"
	"time"

	"dehydrotationlab4/internal/app/ds"
	"dehydrotationlab4/internal/app/middleware"

	"github.com/gin-gonic/gin"
)

// ApiGetCart получить корзину (черновик заявки)
// @Summary Получить корзину
// @Description Требуется авторизация. Возвращает ID черновика и количество симптомов в нем.
// @Tags requests
// @Security BearerAuth
// @Security CookieAuth
// @Accept json
// @Produce json
// @Success 200 {object} object{data=object{draft_id=int,count=int},total=int,filters=object}
// @Failure 401 {object} object{error=string}
// @Router /api/requests/cart [get]
func (h *Handler) ApiGetCart(ctx *gin.Context) {
	userID, ok := middleware.GetCurrentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	count, _ := h.Repository.CountDraftItems(userID)
	draft, _ := h.Repository.GetDraftRequest(userID)
	var draftID *uint
	if draft != nil {
		draftID = &draft.ID
	}
	jsonResponse(ctx, gin.H{"draft_id": draftID, "count": count}, 1, gin.H{})
}

// ApiDraftStatus получить сведения о текущем черновике
// @Summary Черновик текущего пользователя
// @Description Возвращает id черновика и статус, если он есть. Если нет — draft_id=null.
// @Tags requests
// @Security BearerAuth
// @Security CookieAuth
// @Produce json
// @Success 200 {object} object{data=object{draft_id=int,status=string},total=int,filters=object}
// @Failure 401 {object} object{error=string}
// @Router /api/requests/draft-status [get]
func (h *Handler) ApiDraftStatus(ctx *gin.Context) {
	userID, ok := middleware.GetCurrentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	draft, err := h.Repository.GetDraftRequest(userID)
	if err != nil {
		// Если записи нет — возвращаем пустой
		jsonResponse(ctx, gin.H{"draft_id": nil, "status": nil}, 1, gin.H{})
		return
	}
	id := draft.ID
	status := draft.Status
	jsonResponse(ctx, gin.H{"draft_id": &id, "status": &status}, 1, gin.H{})
}

// ApiListRequests получить список заявок
// @Summary Получить список заявок
// @Description Требуется авторизация. Обычный пользователь видит только свои заявки. Модератор видит все заявки.
// @Tags requests
// @Security BearerAuth
// @Security CookieAuth
// @Accept json
// @Produce json
// @Param status query string false "Фильтр по статусу"
// @Param from query string false "Дата начала (formed_at)"
// @Param to query string false "Дата окончания (formed_at)"
// @Success 200 {object} object{data=[]object,total=int,filters=object}
// @Failure 401 {object} object{error=string}
// @Router /api/requests [get]
func (h *Handler) ApiListRequests(ctx *gin.Context) {
	userID, ok := middleware.GetCurrentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Проверяем, является ли пользователь модератором
	isModerator := middleware.IsCurrentUserModerator(ctx)

	status := ctx.Query("status")
	from := ctx.Query("from")
	to := ctx.Query("to")
	var statusPtr *string
	if status != "" {
		statusPtr = &status
	}

	var list []ds.DehydrationRequest
	var err error

	if isModerator {
		// Модератор видит все заявки
		list, err = h.Repository.ListAllRequestsWithFilters(statusPtr, &from, &to)
	} else {
		// Обычный пользователь видит только свои заявки
		list, err = h.Repository.ListRequestsWithFilters(userID, statusPtr, &from, &to)
	}

	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	// Добавляем вычисляемое поле: количество симптомов с непустым комментарием в заявке
	type requestItem struct {
		ds.DehydrationRequest `json:",inline"`
		CommentsCount         int64 `json:"comments_count"`
	}
	resp := make([]requestItem, 0, len(list))
	for _, r := range list {
		cnt, _ := h.Repository.CountRequestSymptomsWithComment(r.ID)
		resp = append(resp, requestItem{DehydrationRequest: r, CommentsCount: cnt})
	}
	// flat=true -> вернуть просто массив без обертки
	if ctx.Query("flat") == "true" {
		ctx.JSON(200, resp)
		return
	}
	jsonResponse(ctx, resp, int64(len(resp)), gin.H{"status": status, "from": from, "to": to})
}

// ApiGetRequest получить заявку по ID
// @Summary Получить заявку по ID
// @Description Требуется авторизация. Возвращает заявку с симптомами.
// @Tags requests
// @Security BearerAuth
// @Security CookieAuth
// @Accept json
// @Produce json
// @Param id path int true "ID заявки"
// @Success 200 {object} object{data=object,total=int,filters=object}
// @Failure 404 {object} object{error=string}
// @Failure 401 {object} object{error=string}
// @Router /api/requests/{id} [get]
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
	// По умолчанию возвращаем плоский объект
	cnt, _ := h.Repository.CountRequestSymptomsWithComment(dr.Request.ID)
	type symOut struct {
		ds.Symptom     `json:",inline"`
		PublicImageURL string `json:"public_image_url"`
	}
	syms := make([]symOut, 0, len(dr.Symptoms))
	for _, s := range dr.Symptoms {
		syms = append(syms, symOut{Symptom: s, PublicImageURL: h.BuildPublicImageURL(s.ImageURL)})
	}
	out := gin.H{
		"id":                  dr.Request.ID,
		"user_id":             dr.Request.UserID,
		"status":              dr.Request.Status,
		"created_at":          dr.Request.CreatedAt,
		"formed_at":           dr.Request.FormedAt,
		"completed_at":        dr.Request.CompletedAt,
		"moderator_id":        dr.Request.ModeratorID,
		"patient_weight":      dr.Request.PatientWeight,
		"dehydration_percent": dr.Request.DehydrationPercent,
		"fluid_deficit":       dr.Request.FluidDeficit,
		"doctor_comment":      dr.Request.DoctorComment,
		"comments_count":      cnt,
		"symptoms":            syms,
	}
	ctx.JSON(200, out)
}

// ApiUpdateRequest обновить заявку
// @Summary Обновить заявку
// @Description Требуется авторизация. Доступно только владельцу черновика.
// @Tags requests
// @Security BearerAuth
// @Security CookieAuth
// @Accept json
// @Produce json
// @Param id path int true "ID заявки"
// @Param request body object{patient_weight=float64,doctor_comment=string} true "Обновляемые поля"
// @Success 200 {object} object{data=ds.DehydrationRequest,total=int,filters=object}
// @Failure 400 {object} object{error=string}
// @Failure 401 {object} object{error=string}
// @Failure 403 {object} object{error=string}
// @Router /api/requests/{id} [put]
func (h *Handler) ApiUpdateRequest(ctx *gin.Context) {
	userID, ok := middleware.GetCurrentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	idStr := ctx.Param("id")
	id64, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}
	id := uint(id64)
	// только владелец
	if owner, err := h.Repository.IsRequestOwner(userID, id); err != nil || !owner {
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

// ApiFormRequest сформировать заявку
// @Summary Сформировать заявку
// @Description Требуется авторизация. Переводит черновик в статус "сформирован". Доступно только владельцу.
// @Tags requests
// @Security BearerAuth
// @Security CookieAuth
// @Accept json
// @Produce json
// @Param id path int true "ID заявки"
// @Success 200 {object} object{data=ds.DehydrationRequest,total=int,filters=object}
// @Failure 400 {object} object{error=string}
// @Failure 401 {object} object{error=string}
// @Failure 403 {object} object{error=string}
// @Router /api/requests/{id}/form [put]
func (h *Handler) ApiFormRequest(ctx *gin.Context) {
	userID, ok := middleware.GetCurrentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	idStr := ctx.Param("id")
	id64, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}
	id := uint(id64)
	// только владелец может формировать черновик
	if owner, err := h.Repository.IsRequestOwner(userID, id); err != nil || !owner {
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

// ApiCompleteRequest завершить/отклонить заявку
// @Summary Завершить или отклонить заявку
// @Description Требуется авторизация модератора. Переводит заявку в статус "завершен" или "отклонен".
// @Tags requests
// @Security BearerAuth
// @Security CookieAuth
// @Accept json
// @Produce json
// @Param id path int true "ID заявки"
// @Param request body object{status=string,patient_weight=float64,dehydration_percent=float64,doctor_comment=string} true "Данные для завершения"
// @Success 200 {object} object{data=ds.DehydrationRequest,total=int,filters=object}
// @Failure 400 {object} object{error=string}
// @Failure 401 {object} object{error=string}
// @Failure 403 {object} object{error=string}
// @Router /api/requests/{id}/complete [put]
func (h *Handler) ApiCompleteRequest(ctx *gin.Context) {
	userID, ok := middleware.GetCurrentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Проверка прав модератора (дополнительная проверка)
	if !middleware.IsCurrentUserModerator(ctx) {
		debugVal, _ := ctx.Get("_moderator_debug")
		ctx.JSON(http.StatusForbidden, gin.H{"error": "only moderator can complete/reject", "debug": debugVal})
		return
	}

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
	if rq.Status != "сформирован" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "only formed can be completed"})
		return
	}
	var fluidDeficit *float64
	if body.Status == "завершен" {
		if body.PatientWeight == nil || body.DehydrationPercent == nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "weight and percent required"})
			return
		}
		v := (*body.PatientWeight) * (*body.DehydrationPercent) * 0.01
		fluidDeficit = &v
	}
	if err := h.Repository.SetCompleted(id, userID, body.Status, time.Now(), body.PatientWeight, body.DehydrationPercent, fluidDeficit, body.DoctorComment); err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}
	rq, _ = h.Repository.GetRequestByID(id)
	jsonResponse(ctx, rq, 1, gin.H{"id": id})
}

// ApiDeleteRequest удалить заявку
// @Summary Удалить заявку
// @Description Требуется авторизация. Доступно только создателю заявки.
// @Tags requests
// @Security BearerAuth
// @Security CookieAuth
// @Accept json
// @Produce json
// @Param id path int true "ID заявки"
// @Success 200 {object} object{data=object{deleted=uint},total=int,filters=object}
// @Failure 400 {object} object{error=string}
// @Failure 401 {object} object{error=string}
// @Failure 403 {object} object{error=string}
// @Router /api/requests/{id} [delete]
func (h *Handler) ApiDeleteRequest(ctx *gin.Context) {
	userID, ok := middleware.GetCurrentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}
	// только создатель
	if owner, err := h.Repository.IsRequestOwner(userID, uint(id)); err != nil || !owner {
		h.errorHandler(ctx, http.StatusForbidden, err)
		return
	}
	if err := h.Repository.DeleteRequest(uint(id)); err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}
	jsonResponse(ctx, gin.H{"deleted": id}, 1, gin.H{})
}

// ApiListAllRequests получить все заявки (для модератора)
// @Summary Получить все заявки плоским списком
// @Description Требуется авторизация модератора. Возвращает все заявки плоским массивом.
// @Tags requests
// @Security BearerAuth
// @Security CookieAuth
// @Accept json
// @Produce json
// @Param status query string false "Фильтр по статусу"
// @Success 200 {array} object
// @Failure 401 {object} object{error=string}
// @Failure 403 {object} object{error=string}
// @Router /api/requests/all [get]
func (h *Handler) ApiListAllRequests(ctx *gin.Context) {
	// Проверка модератора
	if !middleware.IsCurrentUserModerator(ctx) {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "moderator access required"})
		return
	}

	status := ctx.Query("status")
	var statusPtr *string
	if status != "" {
		statusPtr = &status
	}

	list, err := h.Repository.ListAllRequestsWithFilters(statusPtr, nil, nil)
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	type requestItem struct {
		ds.DehydrationRequest `json:",inline"`
		CommentsCount         int64 `json:"comments_count"`
	}
	resp := make([]requestItem, 0, len(list))
	for _, r := range list {
		cnt, _ := h.Repository.CountRequestSymptomsWithComment(r.ID)
		resp = append(resp, requestItem{DehydrationRequest: r, CommentsCount: cnt})
	}

	ctx.JSON(200, resp)
}
