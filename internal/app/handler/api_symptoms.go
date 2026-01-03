package handler

import (
	"context"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"dehydrotationlab3/internal/app/ds"

	"github.com/gin-gonic/gin"
)

// GET /api/symptoms?title=&active=true|false
func (h *Handler) ApiListSymptoms(ctx *gin.Context) {
	title := ctx.Query("title")
	active := ctx.Query("active")
	var activePtr *bool
	if active == "true" {
		v := true
		activePtr = &v
	} else if active == "false" {
		v := false
		activePtr = &v
	}
	list, err := h.Repository.FilterSymptoms(title, activePtr)
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}
	// фильтр по активности, если задан
	if active == "false" {
		// получить и неактивные отдельно (требуется метод в репо) — временно просто не фильтруем
	}
	type symptomItem struct {
		ds.Symptom     `json:",inline"`
		PublicImageURL string `json:"public_image_url"`
	}
	resp := make([]symptomItem, 0, len(list))
	for _, s := range list {
		resp = append(resp, symptomItem{Symptom: s, PublicImageURL: h.BuildPublicImageURL(s.ImageURL)})
	}
	jsonResponse(ctx, resp, int64(len(resp)), gin.H{"title": title, "active": active})
}

// GET /api/symptoms/:id
func (h *Handler) ApiGetSymptom(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}
	s, err := h.Repository.GetSymptomAny(uint(id))
	if err != nil {
		h.errorHandler(ctx, http.StatusNotFound, err)
		return
	}
	jsonResponse(ctx, gin.H{"symptom": s, "public_image_url": h.BuildPublicImageURL(s.ImageURL)}, 1, gin.H{"id": id})
}

// POST /api/symptoms
func (h *Handler) ApiCreateSymptom(ctx *gin.Context) {
	var req ds.Symptom
	if err := ctx.ShouldBindJSON(&req); err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}
	// системные поля игнорим: ID
	req.ID = 0
	if err := h.Repository.CreateSymptom(&req); err != nil {
		// повтор названия (уникальный индекс на title) — вернём 400
		if strings.Contains(err.Error(), "23505") || strings.Contains(strings.ToLower(err.Error()), "unique") {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "symptom title must be unique"})
			return
		}
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}
	jsonResponse(ctx, req, 1, gin.H{})
}

// PUT /api/symptoms/:id
func (h *Handler) ApiUpdateSymptom(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}
	var req ds.Symptom
	if err := ctx.ShouldBindJSON(&req); err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}
	updated, err := h.Repository.UpdateSymptom(uint(id), req)
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}
	jsonResponse(ctx, updated, 1, gin.H{"id": id})
}

// DELETE /api/symptoms/:id
func (h *Handler) ApiDeleteSymptom(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}
	// удалить запись и изображение
	// получить запись, чтобы знать ключ изображения
	sAny, err := h.Repository.GetSymptomAny(uint(id))
	if err != nil {
		h.errorHandler(ctx, http.StatusNotFound, err)
		return
	}
	// удаление изображения из Minio если есть
	if sAny.ImageURL != "" && h.Storage != nil {
		if st, ok := h.Storage.(interface {
			DeleteImage(context.Context, string) error
		}); ok {
			_ = st.DeleteImage(ctx, sAny.ImageURL)
		}
	}
	if err := h.Repository.DeleteSymptom(uint(id)); err != nil {
		// Если симптом используется в заявках (FK 23503) — вернём 400 с понятным сообщением
		if strings.Contains(err.Error(), "23503") || strings.Contains(strings.ToLower(err.Error()), "foreign key") {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "symptom is referenced by requests"})
			return
		}
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}
	jsonResponse(ctx, gin.H{"deleted": id}, 1, gin.H{})
}

// POST /api/symptoms/:id/image
func (h *Handler) ApiUploadSymptomImage(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}

	s, err := h.Repository.GetSymptomAny(uint(id))
	if err != nil {
		h.errorHandler(ctx, http.StatusNotFound, err)
		return
	}

	// Поддержка двух названий поля: file (основное) и image (fallback)
	file, err := ctx.FormFile("file")
	if err != nil {
		file, err = ctx.FormFile("image")
		if err != nil {
			h.errorHandler(ctx, http.StatusBadRequest, err)
			return
		}
	}

	// загрузка в MinIO
	if h.Storage == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "storage not configured"})
		return
	}
	// type assert на наш storage
	if st, ok := h.Storage.(interface {
		UploadImage(context.Context, *multipart.FileHeader, string) (string, string, error)
		DeleteImage(context.Context, string) error
	}); ok {
		// удалим старое, если было
		if s.ImageURL != "" {
			_ = st.DeleteImage(ctx, s.ImageURL)
		}
		key, publicURL, err := st.UploadImage(ctx, file, s.Title)
		if err != nil {
			h.errorHandler(ctx, http.StatusInternalServerError, err)
			return
		}
		// сохранить ключ/имя
		if err := h.Repository.UpdateSymptomImage(uint(id), key); err != nil {
			h.errorHandler(ctx, http.StatusInternalServerError, err)
			return
		}
		jsonResponse(ctx, gin.H{"image_key": key, "public_url": publicURL}, 1, gin.H{"id": id})
		return
	}
	ctx.JSON(http.StatusInternalServerError, gin.H{"error": "storage not configured"})
}
