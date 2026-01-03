package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func (h *Handler) DeleteRequest(ctx *gin.Context) {
	userID := uint(1)

	// Получаем существующий черновик (не создаем новый)
	request, err := h.Repository.GetDraftRequest(userID)
	if err != nil {
		// Если черновика нет - просто редирект
		ctx.Redirect(http.StatusFound, "/")
		return
	}

	if err := h.Repository.DeleteRequest(request.ID); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete request"})
		return
	}
	ctx.Redirect(http.StatusFound, "/")
}

func (h *Handler) ListRequests(ctx *gin.Context) {
	userID := uint(1)
	list, err := h.Repository.ListRequestsByUser(userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load requests"})
		return
	}
	ctx.JSON(http.StatusOK, list)
}

func (h *Handler) ViewRequest(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	dr, err := h.Repository.GetRequestByID(uint(id))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed"})
		return
	}

	if dr == nil || dr.Status == "удален" || dr.Status == "удалён" {
		ctx.Status(http.StatusNotFound)
		return
	}

	// Загружаем симптомы для этой заявки
	symptoms, err := h.Repository.GetRequestSymptoms(uint(id))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load symptoms"})
		return
	}

	// Отображаем HTML-страницу
	ctx.HTML(http.StatusOK, "request_detail.html", gin.H{
		"Request":  dr,
		"Symptoms": symptoms,
	})
}
