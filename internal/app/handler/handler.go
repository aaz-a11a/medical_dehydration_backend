package handler

import (
	"dehydrotationlab3/internal/app/config"
	"dehydrotationlab3/internal/app/repository"
	"fmt"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type Handler struct {
	Repository    *repository.Repository
	Config        *config.Config
	Storage       interface{}
	CurrentUserID uint
}

func NewHandler(r *repository.Repository, cfg *config.Config) *Handler {
	return &Handler{
		Repository:    r,
		Config:        cfg,
		CurrentUserID: 1,
	}
}

// singleton текущего пользователя (лабораторная: фиксированный пользователь)
// Используем sync.Once, чтобы явно реализовать семантику синглтона функции
// и иметь единое место для возможного расширения (например, вытягивать из контекста).
// По методичке — фиксированный пользователь с id=1.
// В коде ниже обращаемся к CurrentUserID() вместо поля структуры.
var (
	userOnce     sync.Once
	cachedUserID uint
)

// CurrentUserID возвращает идентификатор текущего пользователя как функц-синглтон.
func CurrentUserID() uint {
	userOnce.Do(func() {
		cachedUserID = 1
	})
	return cachedUserID
}

// RegisterHandler Функция, в которой мы отдельно регистрируем маршруты
// func (h *Handler) RegisterHandler(router *gin.Engine) {
// 	router.GET("/", h.GetSymptoms)
// 	router.GET("/order/:id", h.GetSymptom)
// 	router.GET("/calculation", h.GetCalculationPage)
// 	router.POST("/add-to-request/:id", h.AddToRequest)
// 	router.POST("/delete-request", h.DeleteRequest)
// 	router.GET("/requests", h.ListRequests)
// 	router.GET("/request/:id", h.ViewRequest)
// }

// RegisterHandler Функция, в которой мы отдельно регистрируем маршруты
func (h *Handler) RegisterHandler(router *gin.Engine) {
	router.GET("/", h.GetSymptoms)
	router.GET("/symptom/:id", h.GetSymptom)
	router.GET("/dehydration-calc", h.GetCalculationPage)
	router.POST("/add-symptom/:id", h.AddToRequest)
	router.POST("/clear-calc", h.DeleteRequest)
	router.GET("/patient-history", h.ListRequests)
	router.GET("/patient-case/:id", h.ViewRequest)
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

// BuildPublicImageURL собирает публичный URL для изображения из ключа (image_url)
func (h *Handler) BuildPublicImageURL(key string) string {
	if key == "" {
		return ""
	}
	return fmt.Sprintf("%s/%s/%s", h.GetMinIOBaseURL(), h.Config.MinIOBucket, key)
}

// errorHandler для более удобного вывода ошибок
func (h *Handler) errorHandler(ctx *gin.Context, errorStatusCode int, err error) {
	logrus.Error(err.Error())
	ctx.JSON(errorStatusCode, gin.H{
		"status":      "error",
		"description": err.Error(),
	})
}

// RegisterAPI регистрирует REST API маршруты
func (h *Handler) RegisterAPI(router *gin.Engine) {
	api := router.Group("/api")

	// Symptoms
	api.GET("/symptoms", h.ApiListSymptoms)
	api.GET("/symptoms/:id", h.ApiGetSymptom)
	api.POST("/symptoms", h.ApiCreateSymptom)
	api.PUT("/symptoms/:id", h.ApiUpdateSymptom)
	api.DELETE("/symptoms/:id", h.ApiDeleteSymptom)
	api.POST("/symptoms/:id/image", h.ApiUploadSymptomImage)

	// Requests
	api.GET("/requests/cart", h.ApiGetCart)
	api.GET("/requests", h.ApiListRequests)
	api.GET("/requests/:id", h.ApiGetRequest)
	api.PUT("/requests/:id", h.ApiUpdateRequest)
	api.PUT("/requests/:id/form", h.ApiFormRequest)
	api.PUT("/requests/:id/complete", h.ApiCompleteRequest)
	api.DELETE("/requests/:id", h.ApiDeleteRequest)

	// Request-Symptoms
	api.POST("/request-symptoms", h.ApiAddRequestSymptom)
	api.DELETE("/request-symptoms", h.ApiDeleteRequestSymptom)
	api.PUT("/request-symptoms", h.ApiUpdateRequestSymptom)

	// Users
	api.POST("/users/register", h.ApiRegisterUser)
	api.GET("/users/profile", h.ApiGetProfile)
	api.PUT("/users/profile", h.ApiUpdateProfile)
	api.POST("/users/login", h.ApiLogin)
	api.POST("/users/logout", h.ApiLogout)
}

// jsonResponse — единый формат ответа
func jsonResponse(ctx *gin.Context, data interface{}, total int64, filters gin.H) {
	ctx.JSON(200, gin.H{
		"data":    data,
		"total":   total,
		"filters": filters,
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
