package handlers

import (
	"fmt"
	"strings"

	"github.com/2000ostd/enssi-tel-bot/internal/models"
	"gopkg.in/telebot.v4"
	"gorm.io/gorm"
)

func sendCourseWord(ctx telebot.Context, db *gorm.DB, telegramUserID int64, courseID uint, idx uint) error {
	// 1) Load our User (to ensure they exist / are enrolled, etc.)
	var user models.User
	if err := db.
		Where("telegram_id = ?", telegramUserID).
		First(&user).Error; err != nil {
		return ctx.Send("مشکلی در یافتن کاربر پیش آمد")
	}

	// 2) Fetch the CourseWord by courseID + idx
	var cw models.CourseWord
	if err := db.
		Where("course_id = ? AND index = ?", courseID, idx).
		First(&cw).Error; err != nil {
		return ctx.Send("کلمه‌ای با این شماره پیدا نشد")
	}

	// 3) Load the Word
	var word models.Word
	if err := db.First(&word, cw.WordID).Error; err != nil {
		return ctx.Send("خطا در بارگذاری کلمه")
	}

	// 4) Load the first WordSource (with its IPA and voice data)
	var ws models.WordSource
	if err := db.
		Where("word_id = ?", word.ID).
		Preload("Phonetics").
		Preload("Pronunciations").
		First(&ws).Error; err != nil {
		return ctx.Send("خطا در بارگذاری تلفظ و تعاریف")
	}

	// 5) Send the image (use Photo or Document depending on what you stored)
	photo := &telebot.Photo{File: telebot.File{FileID: cw.TelgramImageID}}
	photoFile := &telebot.Document{File: telebot.File{FileID: cw.TelgramImageDocID}}

	if err := ctx.Send(photo); err != nil {
		return err
	}

	if err := ctx.Send(photoFile); err != nil {
		return err
	}

	// 6) Build and send the text with definitions and IPA
	//    Collect all IPA strings from Phonetics
	var ipas []string
	for _, p := range ws.Phonetics {
		ipas = append(ipas, p.Title)
	}
	defText := fmt.Sprintf(
		"🔤 «%s\"\n\n📝 تعاریف:\n• %s\n• %s\n\n🎙 IPA: %s",
		word.Title,
		ws.DefPrimary,
		ws.DefSecondary,
		strings.Join(ipas, " ، "),
	)
	if err := ctx.Send(defText); err != nil {
		return err
	}

	// 7) Send the first available voice pronunciation
	if len(ws.Pronunciations) > 0 {
		voice := &telebot.Voice{
			File: telebot.File{FileID: ws.Pronunciations[0].TelgramVoiceID},
		}
		if err := ctx.Send(voice); err != nil {
			return err
		}
	}

	return nil
}
