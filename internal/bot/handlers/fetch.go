package handlers

import (
	"errors"
	"fmt"
	"github.com/2000ostd/enssi-tel-bot/internal/models"
	"github.com/jackc/pgx/v5/pgconn"
	"gopkg.in/telebot.v4"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func fetchUser(ctx telebot.Context, db *gorm.DB) (*models.User, error) {
	sender := ctx.Sender()
	if sender == nil {
		return nil, fmt.Errorf("fetchUser: sender is nil in context")
	}
	tgID := int64(sender.ID)

	var user models.User

	// 1. Attempt to find the user using a clean Where clause.
	err := db.Where("telegram_id = ?", tgID).First(&user).Error

	if err == nil {
		// User found and is active. Return them.
		return &user, nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// User was not found. Proceed to create.
		newUser := models.User{
			TelegramID: tgID,
			LastMenu:   "main",
		}

		createErr := db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "telegram_id"}},
			DoNothing: true,
		}).Create(&newUser).Error

		if createErr != nil {
			var pgErr *pgconn.PgError
			if !(errors.As(createErr, &pgErr) && pgErr.Code == "23505") {
				return nil, fmt.Errorf("fetchUser: error during create with onconflict: %w", createErr)
			}
		}

		// Fetch the record to get the definitive state (either newly created or existing if raced).
		var finalUser models.User
		if finalFetchErr := db.Where("telegram_id = ?", tgID).First(&finalUser).Error; finalFetchErr != nil {
			return nil, fmt.Errorf("fetchUser: failed to fetch user after onconflict create attempt: %w", finalFetchErr)
		}
		return &finalUser, nil
	}

	return nil, fmt.Errorf("fetchUser: initial fetch for user %d failed: %w", tgID, err)
}

func fetchWord(db *gorm.DB, wordID uint) (*models.Word, error) {

	var word models.Word
	if err := db.First(&word, wordID).Error; err != nil {
		return nil, err
	}
	return &word, nil
}

func fetchCourseWordByCourseIDAndIndex(db *gorm.DB, courseID uint, Index uint) (*models.CourseWord, error) {

	var cw models.CourseWord
	if err := db.Where("course_id = ? AND index = ?", courseID, Index).First(&cw).Error; err != nil {
		return nil, err
	}
	return &cw, nil
}
