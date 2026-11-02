package controllers

import (
	"strconv"
	"strings"
	"time"

	"carfix/app/models"
	"carfix/bootstrap"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AdminUserController struct{}

// UserWithStats response structure
type UserWithStats struct {
	models.User
	SearchesCount int64 `json:"searches_count"`
}

// ListUsers returns all users with quota and search logs count
func (a *AdminUserController) ListUsers(c *fiber.Ctx) error {
	q := strings.TrimSpace(c.Query("q"))
	if q == "" {
		q = strings.TrimSpace(c.Query("search"))
	}

	query := bootstrap.DB.Model(&models.User{}).Preload("Quota")

	if q != "" {
		pattern := "%" + q + "%"
		query = query.Where("full_name ILIKE ? OR username ILIKE ? OR phone ILIKE ?", pattern, pattern, pattern)
	}

	var users []models.User
	if err := query.Order("id DESC").Find(&users).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "فشل جلب قائمة المستخدمين"})
	}

	userIDs := make([]uint, len(users))
	for i, u := range users {
		userIDs[i] = u.ID
	}

	// Fetch search counts for these users in a single query
	countsMap := make(map[uint]int64)
	if len(userIDs) > 0 {
		type CountResult struct {
			UserID uint
			Total  int64
		}
		var counts []CountResult
		bootstrap.DB.Model(&models.SearchLog{}).
			Select("user_id, count(*) as total").
			Where("user_id IN ?", userIDs).
			Group("user_id").
			Scan(&counts)

		for _, cr := range counts {
			countsMap[cr.UserID] = cr.Total
		}
	}

	response := make([]UserWithStats, len(users))
	for i, u := range users {
		response[i] = UserWithStats{
			User:          u,
			SearchesCount: countsMap[u.ID],
		}
	}

	return c.JSON(fiber.Map{
		"users": response,
		"data":  response,
	})
}

// CreateUser creates a new user and their quota atomically
func (a *AdminUserController) CreateUser(c *fiber.Ctx) error {
	var input struct {
		FullName   string     `json:"full_name"`
		Phone      string     `json:"phone"`
		Username   string     `json:"username"`
		Password   string     `json:"password"`
		Role       string     `json:"role"`
		IsActive   *bool      `json:"is_active"`
		TotalLimit int        `json:"total_limit"`
		DailyLimit int        `json:"daily_limit"`
		ExpiresAt  *time.Time `json:"expires_at"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "بيانات غير صالحة"})
	}

	input.Username = strings.TrimSpace(input.Username)
	input.FullName = strings.TrimSpace(input.FullName)
	if input.FullName == "" {
		input.FullName = input.Username
	}

	if input.Username == "" || input.Password == "" {
		return c.Status(400).JSON(fiber.Map{"message": "اسم المستخدم وكلمة المرور مطلوبة"})
	}

	// Check if username is already taken
	var count int64
	bootstrap.DB.Model(&models.User{}).Where("username = ?", input.Username).Count(&count)
	if count > 0 {
		return c.Status(409).JSON(fiber.Map{"message": "اسم المستخدم مسجل مسبقاً"})
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "فشل تشفير كلمة المرور"})
	}

	role := "user"
	if input.Role == "admin" {
		role = "admin"
	}

	isActive := true
	if input.IsActive != nil {
		isActive = *input.IsActive
	}

	user := models.User{
		FullName: input.FullName,
		Phone:    strings.TrimSpace(input.Phone),
		Username: input.Username,
		Password: string(hashedPassword),
		Role:     role,
		IsActive: isActive,
	}

	totalLimit := input.TotalLimit
	if totalLimit < 0 {
		totalLimit = 0
	}
	dailyLimit := input.DailyLimit
	if dailyLimit < 0 {
		dailyLimit = 0
	}

	err = bootstrap.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&user).Error; err != nil {
			return err
		}

		quota := models.UserQuota{
			UserID:        user.ID,
			TotalLimit:    totalLimit,
			UsedTotal:     0,
			DailyLimit:    dailyLimit,
			UsedToday:     0,
			LastResetDate: time.Now(),
			ExpiresAt:     input.ExpiresAt,
		}

		if err := tx.Create(&quota).Error; err != nil {
			return err
		}

		user.Quota = &quota
		return nil
	})

	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "فشل إنشاء المستخدم: " + err.Error()})
	}

	return c.Status(201).JSON(user)
}

// UpdateUser updates basic user profile information
func (a *AdminUserController) UpdateUser(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "معرف المستخدم غير صحيح"})
	}

	var user models.User
	if err := bootstrap.DB.Preload("Quota").First(&user, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"message": "المستخدم غير موجود"})
	}

	var input struct {
		FullName string `json:"full_name"`
		Username string `json:"username"`
		Phone    string `json:"phone"`
		Password string `json:"password"`
		Role     string `json:"role"`
		IsActive *bool  `json:"is_active"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "بيانات غير صالحة"})
	}

	updates := make(map[string]interface{})
	if strings.TrimSpace(input.FullName) != "" {
		updates["full_name"] = strings.TrimSpace(input.FullName)
		user.FullName = strings.TrimSpace(input.FullName)
	}
	if strings.TrimSpace(input.Username) != "" && strings.TrimSpace(input.Username) != user.Username {
		newUsername := strings.TrimSpace(input.Username)
		var existing int64
		bootstrap.DB.Model(&models.User{}).Where("username = ? AND id != ?", newUsername, user.ID).Count(&existing)
		if existing > 0 {
			return c.Status(409).JSON(fiber.Map{"message": "اسم المستخدم هذا مستخدم بالفعل لحساب آخر"})
		}
		updates["username"] = newUsername
		user.Username = newUsername
	}
	if input.Phone != "" {
		updates["phone"] = strings.TrimSpace(input.Phone)
		user.Phone = strings.TrimSpace(input.Phone)
	}
	if input.Role == "admin" || input.Role == "user" {
		updates["role"] = input.Role
		user.Role = input.Role
	}
	if input.IsActive != nil {
		updates["is_active"] = *input.IsActive
		user.IsActive = *input.IsActive
	}
	if strings.TrimSpace(input.Password) != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"message": "فشل تشفير كلمة المرور الجديدة"})
		}
		updates["password"] = string(hashed)
	}

	if len(updates) > 0 {
		if err := bootstrap.DB.Model(&user).Updates(updates).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"message": "فشل تحديث بيانات المستخدم"})
		}
	}

	return c.JSON(user)
}

// DeleteUser removes a user account and their quota
func (a *AdminUserController) DeleteUser(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "معرف المستخدم غير صحيح"})
	}

	if currentID, ok := c.Locals("user_id").(uint); ok && currentID == uint(id) {
		return c.Status(400).JSON(fiber.Map{"message": "لا يمكنك حذف حسابك الشخصي الحالي"})
	}

	var user models.User
	if err := bootstrap.DB.First(&user, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"message": "المستخدم غير موجود"})
	}

	err = bootstrap.DB.Transaction(func(tx *gorm.DB) error {
		tx.Where("user_id = ?", user.ID).Delete(&models.UserQuota{})
		tx.Where("user_id = ?", user.ID).Delete(&models.SearchLog{})
		return tx.Delete(&user).Error
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "فشل حذف المستخدم"})
	}

	return c.JSON(fiber.Map{"message": "تم حذف الحساب بنجاح"})
}

// UpdateUserQuota updates or resets user search quota limits
func (a *AdminUserController) UpdateUserQuota(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "معرف المستخدم غير صحيح"})
	}

	var quota models.UserQuota
	if err := bootstrap.DB.Where("user_id = ?", id).First(&quota).Error; err != nil {
		quota = models.UserQuota{
			UserID:        uint(id),
			TotalLimit:    50,
			DailyLimit:    10,
			LastResetDate: time.Now(),
		}
		if createErr := bootstrap.DB.Create(&quota).Error; createErr != nil {
			return c.Status(404).JSON(fiber.Map{"message": "سجل الرصيد غير موجود للمستخدم"})
		}
	}

	var input struct {
		TotalLimit *int       `json:"total_limit"`
		DailyLimit *int       `json:"daily_limit"`
		ExpiresAt  *time.Time `json:"expires_at"`
		ResetDaily bool       `json:"reset_daily"`
		ResetToday bool       `json:"reset_today"`
		ResetTotal bool       `json:"reset_total"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "بيانات غير صالحة"})
	}

	updates := make(map[string]interface{})
	if input.TotalLimit != nil && *input.TotalLimit >= 0 {
		updates["total_limit"] = *input.TotalLimit
		quota.TotalLimit = *input.TotalLimit
	}
	if input.DailyLimit != nil && *input.DailyLimit >= 0 {
		updates["daily_limit"] = *input.DailyLimit
		quota.DailyLimit = *input.DailyLimit
	}
	if input.ExpiresAt != nil {
		updates["expires_at"] = input.ExpiresAt
		quota.ExpiresAt = input.ExpiresAt
	}
	if input.ResetDaily || input.ResetToday {
		updates["used_today"] = 0
		updates["last_reset_date"] = time.Now()
		quota.UsedToday = 0
		quota.LastResetDate = time.Now()
	}
	if input.ResetTotal {
		updates["used_total"] = 0
		quota.UsedTotal = 0
	}

	if len(updates) > 0 {
		if err := bootstrap.DB.Model(&quota).Updates(updates).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"message": "فشل تحديث الرصيد"})
		}
	}

	return c.JSON(quota)
}

// ListSearchLogs returns tracked search attempts with filtering, pagination and overview statistics
func (a *AdminUserController) ListSearchLogs(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	userID := strings.TrimSpace(c.Query("user_id"))
	vin := strings.TrimSpace(c.Query("vin"))
	resultStatus := strings.TrimSpace(c.Query("result_status"))
	fetchSource := strings.TrimSpace(c.Query("fetch_source"))

	query := bootstrap.DB.Model(&models.SearchLog{})

	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	if vin != "" {
		query = query.Where("vin ILIKE ?", "%"+vin+"%")
	}
	if resultStatus != "" {
		query = query.Where("result_status = ?", resultStatus)
	}
	if fetchSource != "" {
		query = query.Where("fetch_source = ?", fetchSource)
	}

	var total int64
	query.Count(&total)

	var logs []models.SearchLog
	if err := query.Preload("User").Order("created_at DESC").Limit(limit).Offset(offset).Find(&logs).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "فشل جلب سجلات البحث"})
	}

	// Calculate overview statistics
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	var totalToday int64
	bootstrap.DB.Model(&models.SearchLog{}).Where("created_at >= ?", startOfDay).Count(&totalToday)

	var successToday int64
	bootstrap.DB.Model(&models.SearchLog{}).Where("created_at >= ? AND result_status = ?", startOfDay, "success").Count(&successToday)

	var totalAll int64
	bootstrap.DB.Model(&models.SearchLog{}).Count(&totalAll)

	var cacheHits int64
	bootstrap.DB.Model(&models.SearchLog{}).Where("fetch_source = ?", "cache").Count(&cacheHits)

	cacheRatio := 0.0
	if totalAll > 0 {
		cacheRatio = float64(cacheHits) / float64(totalAll) * 100.0
	}

	return c.JSON(fiber.Map{
		"logs": logs,
		"data": logs,
		"pagination": fiber.Map{
			"page":  page,
			"limit": limit,
			"total": total,
		},
		"stats": fiber.Map{
			"total_today":   totalToday,
			"success_today": successToday,
			"total_all":     totalAll,
			"cache_hits":    cacheHits,
			"cache_ratio":   strconv.FormatFloat(cacheRatio, 'f', 1, 64) + "%",
		},
	})
}
