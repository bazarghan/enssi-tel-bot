package handlers

import (
	"github.com/2000ostd/enssi-tel-bot/internal/models"
	"gopkg.in/telebot.v4"
	"gorm.io/gorm"
)

func fetchUser(ctx telebot.Context, db *gorm.DB) (*models.User, error) {

	sender := ctx.Sender()
	tgID := int64(sender.ID)

	user := models.User{TelegramID: tgID}
	if err := db.
		FirstOrCreate(&user, models.User{TelegramID: tgID, LastMenu: "main"}).
		Error; err != nil {
		return nil, err
	}

	return &user, nil

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
	if err := db.Where("courseID = ? AND Index = ?", courseID, Index).First(&cw).Error; err != nil {
		return nil, err
	}
	return &cw, nil
}
