package handlers

import (
	"fmt"
	"strings"

	"github.com/2000ostd/enssi-tel-bot/internal/models"
	"gopkg.in/telebot.v4"
	"gorm.io/gorm"
)

func sendCourseWord(ctx telebot.Context, db *gorm.DB, courseID uint, idx uint) error {

	cw, err := fetchCourseWordByCourseIDAndIndex(db, courseID, idx)

	if err != nil {
		return err
	}

	word, err := fetchWord(db, cw.WordID)
	if err != nil {
		return ctx.Send("can't find the word")
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

func createQuiz(ctx telebot.Context, db *gorm.DB, courseID uint, idx uint) error {

	user, err := fetchUser(ctx, db)
	if err != nil {
		return err
	}

	cw, err := fetchCourseWordByCourseIDAndIndex(db, courseID, idx)
	if err != nil {
		return err
	}

	word, err := fetchWord(db, cw.CourseID)

	// 2. Create the UserQuiz model instance
	newQuiz := models.UserQuiz{
		UserID:               user.ID,
		CourseID:             courseID,
		Type:                 "multi-option",
		IsCompleted:          false,
		Score:                0,
		CurrentQuestionIndex: 0,
		TotalQuestions:       12,
		LastQuestionWordID:   word.ID,
	}

	if err := db.Create(&newQuiz).Error; err != nil {
		// Log the actual error for server-side debugging
		// log.Printf("Error creating quiz for user %d, course %d: %v", user.ID, courseID, err)
		return ctx.Send("مشکلی در ایجاد آزمون جدید پیش آمد. لطفا دوباره تلاش کنید.")
	}

	confirmationMessage := fmt.Sprintf(
		"✅ آزمون جدید برای دوره با شناسه %d با موفقیت برای شما ایجاد شد.\n"+
			"نوع آزمون: چند گزینه‌ای\n"+
			"تعداد سوالات: %d\n\n"+
			"برای شروع آزمون، دستور /startquiz %d را ارسال کنید (یا دکمه مربوطه را فشار دهید).", // Assuming newQuiz.ID is what you'd use to start/refer to it
		courseID,
		newQuiz.TotalQuestions,
		newQuiz.ID, // Or some other identifier if you don't want to expose DB ID directly
	)
	// If you want to immediately start the quiz, you would call a function here like:
	// return sendQuizQuestion(ctx, db, newQuiz.ID, newQuiz.CurrentQuestionIndex)

	return ctx.Send(confirmationMessage)
}
