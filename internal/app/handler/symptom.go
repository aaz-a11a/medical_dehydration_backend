package handler

import (
	"net/http"
	"strconv"
	"time"

	"dehydrotationlab3/internal/app/ds"

	"github.com/gin-gonic/gin"
)

func (h *Handler) GetSymptom(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}

	symptom, err := h.Repository.GetSymptom(uint(id))
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.HTML(http.StatusOK, "symptom_detail.html", gin.H{
		"symptom":    symptom,
		"minio_base": h.GetMinIOBaseURL(),
	})
}

func (h *Handler) GetSymptoms(ctx *gin.Context) {
	searchQuery := ctx.Query("query")
	var symptoms []ds.Symptom
	var err error

	if searchQuery == "" {
		symptoms, err = h.Repository.GetActiveSymptoms()
	} else {
		symptoms, err = h.Repository.SearchSymptoms(searchQuery)
	}

	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	// Count items in user's draft request
	userID := uint(1)
	calculationCount, _ := h.Repository.CountDraftItems(userID)

	ctx.HTML(http.StatusOK, "symptoms.html", gin.H{
		"time":             time.Now().Format("15:04:05"),
		"symptoms":         symptoms,
		"query":            searchQuery,
		"calculationCount": calculationCount,
		"minio_base":       h.GetMinIOBaseURL(),
	})
}

func (h *Handler) GetCalculationPage(ctx *gin.Context) {
	userID := uint(1)
	// Возможность открыть конкретную заявку по id: /dehydration-calc?id=123
	if idStr := ctx.Query("id"); idStr != "" {
		if id, err := strconv.ParseUint(idStr, 10, 64); err == nil {
			// Проверим принадлежность заявки пользователю
			if owner, err := h.Repository.IsRequestOwner(userID, uint(id)); err == nil && owner {
				req, err := h.Repository.GetRequestByID(uint(id))
				if err == nil && req != nil {
					syms, _ := h.Repository.GetRequestSymptoms(uint(id))
					ctx.HTML(http.StatusOK, "dehydration_calc.html", gin.H{
						"symptoms":      syms,
						"symptomsCount": len(syms),
						"request":       req,
						"time":          time.Now().Format("15:04:05"),
						"minio_base":    h.GetMinIOBaseURL(),
					})
					return
				}
			}
			// если что-то пошло не так — 404
			ctx.Status(http.StatusNotFound)
			return
		}
	}

	// По умолчанию — работаем с черновиком
	calculationSymptoms, request, err := h.Repository.GetDraftSymptoms(userID)
	if err != nil {
		// Если черновика нет - редирект на главную
		ctx.Redirect(http.StatusFound, "/")
		return
	}
	calculationCount := len(calculationSymptoms)
	if calculationCount == 0 {
		// Удаляем пустой черновик чтобы не копился мусор
		_ = h.Repository.DeleteRequest(request.ID)
		ctx.Redirect(http.StatusFound, "/")
		return
	}

	ctx.HTML(http.StatusOK, "dehydration_calc.html", gin.H{
		"symptoms":      calculationSymptoms,
		"symptomsCount": calculationCount,
		"request":       request,
		"time":          time.Now().Format("15:04:05"),
		"minio_base":    h.GetMinIOBaseURL(),
	})
}

func (h *Handler) AddToRequest(ctx *gin.Context) {
	symptomIDStr := ctx.Param("id")
	symptomID, err := strconv.ParseUint(symptomIDStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid symptom ID"})
		return
	}

	userID := uint(1)

	request, err := h.Repository.GetOrCreateDraftRequest(userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}

	err = h.Repository.AddSymptomToRequest(request.ID, uint(symptomID))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add symptom to request"})
		return
	}

	ctx.Redirect(http.StatusFound, "/")
}
