package services

import (
	"errors"
	"time"

	"carfix/app/models"
	"carfix/bootstrap"
	"gorm.io/gorm"
)

type QuotaManagerService struct {
	db *gorm.DB
}

func NewQuotaManagerService(db ...*gorm.DB) *QuotaManagerService {
	var database *gorm.DB
	if len(db) > 0 && db[0] != nil {
		database = db[0]
	} else {
		database = bootstrap.DB
	}
	return &QuotaManagerService{
		db: database,
	}
}

// CheckAndPrepareQuota checks if the user has available quota, resetting daily quota if midnight passed
func (s *QuotaManagerService) CheckAndPrepareQuota(userID uint) (*models.UserQuota, error) {
	var quota models.UserQuota
	if err := s.db.Where("user_id = ?", userID).First(&quota).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("لم يتم العثور على باقة رصيد لهذا المستخدم")
		}
		return nil, err
	}

	now := time.Now()

	// 1. Daily Reset Check: Compare Year and YearDay or date string
	loc := now.Location()
	lastResetInLoc := quota.LastResetDate.In(loc)
	isDifferentDay := lastResetInLoc.Year() != now.Year() || lastResetInLoc.YearDay() != now.YearDay()

	if isDifferentDay {
		quota.UsedToday = 0
		quota.LastResetDate = now
		if err := s.db.Model(&quota).Updates(map[string]interface{}{
			"used_today":       0,
			"last_reset_date": now,
		}).Error; err != nil {
			return nil, err
		}
	}

	// 2. Check Expiration
	if quota.ExpiresAt != nil && !quota.ExpiresAt.IsZero() && quota.ExpiresAt.Before(now) {
		return &quota, errors.New("انتهت صلاحية باقة الاشتراك الخاصة بك")
	}

	// 3. Check Total Limit
	if quota.TotalLimit > 0 && quota.UsedTotal >= quota.TotalLimit {
		return &quota, errors.New("لقد استنفدت رصيد البحث الكلي المخصص لحسابك")
	}

	// 4. Check Daily Limit
	if quota.DailyLimit > 0 && quota.UsedToday >= quota.DailyLimit {
		return &quota, errors.New("لقد وصلت إلى الحد الأقصى للبحث اليومي، يتجدد الرصيد غداً")
	}

	return &quota, nil
}

// ConsumeQuota atomically increments used_total and used_today
func (s *QuotaManagerService) ConsumeQuota(userID uint) error {
	res := s.db.Model(&models.UserQuota{}).
		Where("user_id = ? AND used_total < total_limit AND used_today < daily_limit", userID).
		Updates(map[string]interface{}{
			"used_total": gorm.Expr("used_total + 1"),
			"used_today": gorm.Expr("used_today + 1"),
		})

	if res.Error != nil {
		return res.Error
	}

	if res.RowsAffected == 0 {
		return errors.New("فشل استهلاك الرصيد: قد يكون الرصيد الكلي أو اليومي قد نفد بالفعل")
	}

	return nil
}
