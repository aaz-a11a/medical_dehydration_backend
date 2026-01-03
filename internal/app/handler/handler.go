package handler

import (
	"dehydrotationlab2/internal/app/config"
	"dehydrotationlab2/internal/app/repository"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type Handler struct {
	Repository *repository.Repository
	Config     *config.Config
}

func NewHandler(r *repository.Repository, cfg *config.Config) *Handler {
	return &Handler{
		Repository: r,
		Config:     cfg,
	}
}

// RegisterHandler Функция, в которой мы отдельно регистрируем маршруты
func (h *Handler) RegisterHandler(router *gin.Engine) {
	router.GET("/", h.GetSymptoms)
	router.GET("/symptom/:id", h.GetSymptom)
	router.GET("/dehydration-calc", h.GetCalculationPage)
	router.POST("/add-symptom/:id", h.AddToRequest)
	router.POST("/clear-calc", h.DeleteRequest)

	router.GET("/patient-history", h.ListRequests)
	router.GET("/dehydration-request/:id", h.ViewRequest)
}

// RegisterStatic То же самое, что и с маршрутами, регистрируем статику
func (h *Handler) RegisterStatic(router *gin.Engine) {
	router.LoadHTMLGlob("templates/*")
	router.Static("/static", "./resources")
}

// GetMinIOBaseURL возвращает базовый URL для MinIO
func (h *Handler) GetMinIOBaseURL() string {
	return fmt.Sprintf("http://%s:%s", h.Config.MinIOHost, h.Config.MinIOPort)
}

// errorHandler для более удобного вывода ошибок
func (h *Handler) errorHandler(ctx *gin.Context, errorStatusCode int, err error) {
	logrus.Error(err.Error())
	ctx.JSON(errorStatusCode, gin.H{
		"status":      "error",
		"description": err.Error(),
	})
}

// func calculateDehydration(weight float64, symptoms []Symptom) float64 {
//     // Расчет процента обезвоживания на основе симптомов
//     totalSeverity := 0.0
//     symptomCount := 0

//     // Суммируем тяжесть симптомов
//     for _, symptom := range symptoms {
//         switch symptom.Severity {
//         case "Легкая (1-2%)":
//             totalSeverity += 1.5
//         case "Средняя (3-6%)":
//             totalSeverity += 4.5
//         case "Тяжелая (7-9%)":
//             totalSeverity += 8.0
//         }
//         symptomCount++
//     }

//     // Рассчитываем средний процент обезвоживания
//     if symptomCount > 0 {
//         dehydrationPercent := totalSeverity / float64(symptomCount)
//         return dehydrationPercent
//     }

//     // Если симптомов нет, возвращаем 0
//     return 0.0
// }
