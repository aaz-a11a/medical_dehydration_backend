package handler

import (
	"net/http"

	"dehydrotationlab4/internal/app/ds"
	"dehydrotationlab4/internal/app/middleware"
	"dehydrotationlab4/internal/app/pkg/auth"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// ApiRegisterUser регистрация нового пользователя
// @Summary Регистрация нового пользователя
// @Tags auth
// @Accept json
// @Produce json
// @Param request body object{login=string,password=string,is_moderator=bool} true "Данные для регистрации"
// @Success 200 {object} object{user=ds.User}
// @Failure 400 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Router /api/users/register [post]
func (h *Handler) ApiRegisterUser(ctx *gin.Context) {
	type requestBody struct {
		Login       string `json:"login" binding:"required,min=3,max=50"`
		Password    string `json:"password" binding:"required,min=6"`
		IsModerator *bool  `json:"is_moderator,omitempty"`
	}

	var body requestBody
	if err := ctx.ShouldBindJSON(&body); err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}

	// Проверяем, не существует ли уже пользователь с таким логином
	if existing, err := h.Repository.GetUserByLogin(body.Login); err == nil && existing != nil {
		// Если пользователь уже существует и пароль совпадает — ведем себя как login
		if bcrypt.CompareHashAndPassword([]byte(existing.Password), []byte(body.Password)) == nil {
			// Повышение до модератора, если запрошено
			if body.IsModerator != nil && *body.IsModerator && !existing.IsModerator {
				_ = h.Repository.UpdateUser(existing.ID, map[string]interface{}{"is_moderator": true})
				existing.IsModerator = true
			}
			token, err := h.JWTService.Generate(existing.ID, existing.Login, existing.IsModerator)
			if err != nil {
				h.errorHandler(ctx, http.StatusInternalServerError, err)
				return
			}

			// Создаем сессию в Redis
			sessionID := uuid.New().String()
			sessionData := auth.SessionData{
				UserID:      existing.ID,
				Login:       existing.Login,
				IsModerator: existing.IsModerator,
			}
			if err := h.SessionService.Create(ctx.Request.Context(), sessionID, sessionData); err != nil {
				h.errorHandler(ctx, http.StatusInternalServerError, err)
				return
			}

			// Устанавливаем cookie с session_id
			ctx.SetCookie("session_id", sessionID, 86400, "/", "", false, true)

			jsonResponse(ctx, gin.H{
				"user":       existing,
				"token":      token,
				"session_id": sessionID,
				"note":       "user already existed; password matched -> logged in",
			}, 1, gin.H{})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "user already exists"})
		return
	}

	// Хешируем пароль
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	// Создаем пользователя
	isMod := false
	if body.IsModerator != nil {
		isMod = *body.IsModerator
	}
	user := &ds.User{
		Login:       body.Login,
		Password:    string(hashedPassword),
		IsModerator: isMod,
	}

	if err := h.Repository.CreateUser(user); err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}
	// Перечитаем из БД на всякий случай
	created, _ := h.Repository.GetUserByLogin(user.Login)

	// Если это модератор — сразу выдаем токен и сессию
	if created != nil && created.IsModerator {
		token, err := h.JWTService.Generate(created.ID, created.Login, created.IsModerator)
		if err != nil {
			h.errorHandler(ctx, http.StatusInternalServerError, err)
			return
		}
		sessionID := uuid.New().String()
		sessionData := auth.SessionData{
			UserID:      created.ID,
			Login:       created.Login,
			IsModerator: created.IsModerator,
		}
		if err := h.SessionService.Create(ctx.Request.Context(), sessionID, sessionData); err != nil {
			h.errorHandler(ctx, http.StatusInternalServerError, err)
			return
		}
		ctx.SetCookie("session_id", sessionID, 86400, "/", "", false, true)
		jsonResponse(ctx, gin.H{"user": created, "token": token, "session_id": sessionID}, 1, gin.H{})
		return
	}

	jsonResponse(ctx, gin.H{"user": created}, 1, gin.H{})
}

// ApiLogin вход пользователя
// @Summary Вход пользователя
// @Tags auth
// @Accept json
// @Produce json
// @Param request body object{login=string,password=string} true "Данные для входа"
// @Success 200 {object} object{user=ds.User,token=string,session_id=string}
// @Failure 400 {object} object{error=string}
// @Failure 401 {object} object{error=string}
// @Router /api/users/login [post]
func (h *Handler) ApiLogin(ctx *gin.Context) {
	type requestBody struct {
		Login    string `json:"login" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	var body requestBody
	if err := ctx.ShouldBindJSON(&body); err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}

	// Находим пользователя
	user, err := h.Repository.GetUserByLogin(body.Login)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// Проверяем пароль
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(body.Password)); err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// Генерируем JWT токен
	token, err := h.JWTService.Generate(user.ID, user.Login, user.IsModerator)
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	// Создаем сессию в Redis
	sessionID := uuid.New().String()
	sessionData := auth.SessionData{
		UserID:      user.ID,
		Login:       user.Login,
		IsModerator: user.IsModerator,
	}
	if err := h.SessionService.Create(ctx.Request.Context(), sessionID, sessionData); err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	// Устанавливаем cookie с session_id
	ctx.SetCookie("session_id", sessionID, 86400, "/", "", false, true)

	jsonResponse(ctx, gin.H{
		"user":       user,
		"token":      token,
		"session_id": sessionID,
	}, 1, gin.H{})
}

// ApiLogout выход пользователя
// @Summary Выход пользователя
// @Tags auth
// @Security BearerAuth
// @Security CookieAuth
// @Produce json
// @Success 200 {object} object{message=string}
// @Router /api/users/logout [post]
func (h *Handler) ApiLogout(ctx *gin.Context) {
	// Удаляем сессию из Redis
	if sessionID, err := ctx.Cookie("session_id"); err == nil && sessionID != "" {
		_ = h.SessionService.Delete(ctx.Request.Context(), sessionID)
	}

	// Удаляем cookie
	ctx.SetCookie("session_id", "", -1, "/", "", false, true)

	jsonResponse(ctx, gin.H{"message": "logged out"}, 1, gin.H{})
}

// ApiGetProfile получение профиля текущего пользователя
// @Summary Получение профиля текущего пользователя
// @Tags auth
// @Security BearerAuth
// @Security CookieAuth
// @Produce json
// @Success 200 {object} object{user=ds.User}
// @Failure 401 {object} object{error=string}
// @Router /api/users/profile [get]
func (h *Handler) ApiGetProfile(ctx *gin.Context) {
	userID, ok := middleware.GetCurrentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	user, err := h.Repository.GetUserByID(userID)
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	// minimal=true -> вернуть сокращенный профиль + агрегаты
	if ctx.Query("minimal") == "true" {
		// посчитаем количество сформированных и завершенных
		formedCnt, _ := h.Repository.CountUserRequestsByStatus(userID, "сформирован")
		completedCnt, _ := h.Repository.CountUserRequestsByStatus(userID, "завершен")
		out := gin.H{
			"id":           user.ID,
			"login":        user.Login,
			"is_moderator": user.IsModerator,
			"result": gin.H{
				"formed":    formedCnt,
				"completed": completedCnt,
			},
		}
		ctx.JSON(200, out)
		return
	}

	jsonResponse(ctx, gin.H{"user": user}, 1, gin.H{})
}

// ApiUpdateProfile обновление профиля текущего пользователя
// @Summary Обновление профиля
// @Tags auth
// @Security BearerAuth
// @Security CookieAuth
// @Accept json
// @Produce json
// @Param request body object{login=string} true "Новые данные профиля"
// @Success 200 {object} object{user=ds.User}
// @Failure 400 {object} object{error=string}
// @Failure 401 {object} object{error=string}
// @Router /api/users/profile [put]
func (h *Handler) ApiUpdateProfile(ctx *gin.Context) {
	userID, ok := middleware.GetCurrentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	type requestBody struct {
		Login *string `json:"login,omitempty"`
	}

	var body requestBody
	if err := ctx.ShouldBindJSON(&body); err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}

	fields := map[string]interface{}{}
	if body.Login != nil {
		fields["login"] = *body.Login
	}

	if len(fields) > 0 {
		if err := h.Repository.UpdateUser(userID, fields); err != nil {
			h.errorHandler(ctx, http.StatusInternalServerError, err)
			return
		}
	}

	user, _ := h.Repository.GetUserByID(userID)
	jsonResponse(ctx, gin.H{"user": user}, 1, gin.H{})
}

// ApiLoginModerator быстрый вход/создание модератора
// @Summary Вход модератора (автосоздание/повышение)
// @Tags auth
// @Accept json
// @Produce json
// @Param request body object{login=string,password=string} true "Логин и пароль модератора"
// @Success 200 {object} object{user=ds.User,token=string,session_id=string}
// @Failure 400 {object} object{error=string}
// @Failure 401 {object} object{error=string}
// @Router /api/users/login-moderator [post]
func (h *Handler) ApiLoginModerator(ctx *gin.Context) {
	type bodyT struct {
		Login    string `json:"login" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	var body bodyT
	if err := ctx.ShouldBindJSON(&body); err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}
	user, err := h.Repository.GetUserByLogin(body.Login)
	if err != nil || user == nil {
		// Создаём нового модератора
		hashed, errHP := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
		if errHP != nil {
			h.errorHandler(ctx, http.StatusInternalServerError, errHP)
			return
		}
		user = &ds.User{Login: body.Login, Password: string(hashed), IsModerator: true}
		if err := h.Repository.CreateUser(user); err != nil {
			h.errorHandler(ctx, http.StatusInternalServerError, err)
			return
		}
	} else {
		// Сравниваем пароль
		if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(body.Password)) != nil {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}
		// Повышаем до модератора, если еще нет
		if !user.IsModerator {
			_ = h.Repository.UpdateUser(user.ID, map[string]interface{}{"is_moderator": true})
			user.IsModerator = true
		}
	}
	// Генерируем токен
	token, err := h.JWTService.Generate(user.ID, user.Login, user.IsModerator)
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}
	sessionID := uuid.New().String()
	sessionData := auth.SessionData{UserID: user.ID, Login: user.Login, IsModerator: user.IsModerator}
	if err := h.SessionService.Create(ctx.Request.Context(), sessionID, sessionData); err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}
	ctx.SetCookie("session_id", sessionID, 86400, "/", "", false, true)
	jsonResponse(ctx, gin.H{"user": user, "token": token, "session_id": sessionID}, 1, gin.H{})
}
