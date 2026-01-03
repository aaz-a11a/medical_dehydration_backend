package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func (h *Handler) DeleteRequest(ctx *gin.Context) {
	userID := CurrentUserID()

	// Получаем существующий черновик (не создаем новый)
	request, err := h.Repository.GetDraftRequest(userID)
	if err != nil {
		// Если черновика нет - просто редирект
		ctx.Redirect(http.StatusFound, "/")
		return
	}

	// Вызов логического удаления через SQL UPDATE
	if err := h.Repository.DeleteRequest(request.ID); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete request"})
		return
	}
	ctx.Redirect(http.StatusFound, "/")
}

func (h *Handler) ListRequests(ctx *gin.Context) {
	userID := CurrentUserID()
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
	wr, err := h.Repository.GetRequestWithSymptoms(uint(id))
	if err != nil || wr == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed"})
		return
	}
	if wr.Request.Status == "удален" || wr.Request.Status == "удалён" {
		ctx.Status(http.StatusNotFound)
		return
	}
	// вычислим результат
	var result *float64
	if wr.Request.PatientWeight != nil && wr.Request.DehydrationPercent != nil {
		v := (*wr.Request.PatientWeight) * (*wr.Request.DehydrationPercent) * 0.01
		result = &v
	} else if wr.Request.FluidDeficit != nil {
		result = wr.Request.FluidDeficit
	}
	ctx.HTML(http.StatusOK, "patient_case.html", gin.H{
		"request":    wr.Request,
		"symptoms":   wr.Symptoms,
		"result":     result,
		"minio_base": h.GetMinIOBaseURL(),
	})
}
