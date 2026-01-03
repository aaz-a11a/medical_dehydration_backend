package handler

import (
	"dehydrotationlab4/internal/app/config"
	"dehydrotationlab4/internal/app/middleware"
	"dehydrotationlab4/internal/app/pkg/auth"
	"dehydrotationlab4/internal/app/repository"
	"fmt"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type Handler struct {
	Repository     *repository.Repository
	Config         *config.Config
	Storage        interface{}
	JWTService     *auth.JWTService
	SessionService *auth.SessionService
	CurrentUserID  uint
}

func NewHandler(r *repository.Repository, cfg *config.Config, jwtSvc *auth.JWTService, sessSvc *auth.SessionService) *Handler {
	return &Handler{
		Repository:     r,
		Config:         cfg,
		JWTService:     jwtSvc,
		SessionService: sessSvc,
		CurrentUserID:  1,
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

	// Public auth endpoints (без авторизации)
	api.POST("/users/register", h.ApiRegisterUser)
	api.POST("/users/login", h.ApiLogin)
	api.POST("/users/login-moderator", h.ApiLoginModerator)

	// Импортируем middleware для удобства
	authSvc := &middleware.AuthService{
		JWT:     h.JWTService,
		Session: h.SessionService,
	}

	// Public read endpoints (без авторизации, опциональная аутентификация для фильтрации)
	publicSymptoms := api.Group("/symptoms")
	{
		publicSymptoms.GET("", h.ApiListSymptoms)
		publicSymptoms.GET("/:id", h.ApiGetSymptom)
	}

	// Protected endpoints (требуют аутентификации)
	protected := api.Group("")
	protected.Use(middleware.AuthMiddleware(authSvc))
	{
		// User profile
		protected.POST("/users/logout", h.ApiLogout)
		protected.GET("/users/profile", h.ApiGetProfile)
		protected.PUT("/users/profile", h.ApiUpdateProfile)

		// Requests (требуют авторизации)
		protected.GET("/requests/cart", h.ApiGetCart)
		protected.GET("/requests/draft-status", h.ApiDraftStatus)
		protected.GET("/requests", h.ApiListRequests)
		protected.GET("/requests/:id", h.ApiGetRequest)
		protected.PUT("/requests/:id", h.ApiUpdateRequest)
		protected.PUT("/requests/:id/form", h.ApiFormRequest)
		protected.DELETE("/requests/:id", h.ApiDeleteRequest)

		// Request-Symptoms (требуют авторизации)
		protected.POST("/request-symptoms", h.ApiAddRequestSymptom)
		protected.DELETE("/request-symptoms", h.ApiDeleteRequestSymptom)
		protected.PUT("/request-symptoms", h.ApiUpdateRequestSymptom)
	}

	// Moderator endpoints (требуют роль модератора)
	moderator := api.Group("")
	moderator.Use(middleware.AuthMiddleware(authSvc))
	moderator.Use(middleware.RequireModeratorMiddleware())
	{
		// Symptoms (CRUD для модератора)
		moderator.POST("/symptoms", h.ApiCreateSymptom)
		moderator.PUT("/symptoms/:id", h.ApiUpdateSymptom)
		moderator.DELETE("/symptoms/:id", h.ApiDeleteSymptom)
		moderator.POST("/symptoms/:id/image", h.ApiUploadSymptomImage)

		// Complete/reject requests (только модератор)
		moderator.PUT("/requests/:id/complete", h.ApiCompleteRequest)
		moderator.GET("/requests/all", h.ApiListAllRequests)
	}
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
