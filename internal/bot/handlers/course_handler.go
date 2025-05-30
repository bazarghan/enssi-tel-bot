package handlers

import (
	"errors"
	"fmt"
	"github.com/2000ostd/enssi-tel-bot/internal/bot/keyboards"
	"github.com/2000ostd/enssi-tel-bot/internal/models"
	"gopkg.in/telebot.v4"
	"gorm.io/gorm"
	"log"
	"strconv"
	"strings"
)

func handleStartCourse(ctx telebot.Context, db *gorm.DB) error {

	user, err := fetchUser(ctx, db)
	if err != nil {
		return err
	}

	// 2) Parse courseId out of LastMenu
	const prefix = "courseId:"
	if !strings.HasPrefix(user.LastMenu, prefix) {
		return ctx.Send("ابتدا باید یک دوره انتخاب کنید")
	}
	idStr := strings.TrimPrefix(user.LastMenu, prefix)
	cid, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return ctx.Send("خطا در خواندن شناسه دوره")
	}
	courseID := uint(cid)

	// 3) Load the Course
	var course models.Course
	if err := db.First(&course, courseID).Error; err != nil {
		return ctx.Send("دوره‌ای با این شناسه پیدا نشد")
	}

	var uc models.UserCourse
	if err := db.
		Where("user_id = ? AND course_id = ?", user.ID, course.ID).
		First(&uc).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			// not enrolled yet → start at 1
			uc = models.UserCourse{
				UserID:   user.ID,
				CourseID: course.ID,
				Progress: 1,
			}
			if err := db.Create(&uc).Error; err != nil {
				return ctx.Send("خطا در ثبت پیشرفت دوره")
			}
		} else {
			return ctx.Send("خطا در بررسی دوره کاربر")
		}
	}

	return handleStartOrResumeCourse(ctx, db, user, &course, &uc)

}

func handleStartOrResumeCourse(ctx telebot.Context, db *gorm.DB, user *models.User, course *models.Course, uc *models.UserCourse) error {

	// check for active quiz attempt
	blockEndProgress := ((uc.Progress-1)/WordsPerQuizBlock + 1) * WordsPerQuizBlock
	if uc.Progress == 0 {
		blockEndProgress = WordsPerQuizBlock
	}
	activeAttemptForBlock, _ := findActiveQuizAttemptForBlock(db, user.ID, course.ID, blockEndProgress)
	if activeAttemptForBlock != nil {
		ctx.Send("شما یک آزمون نیمه‌تمام برای این بخش از دوره دارید. ادامه می‌دهیم...")
		return sendCurrentQuizQuestion(ctx, db, activeAttemptForBlock.ID)
	}

	// If uc.Progress is exactly at a quiz trigger point (e.g. 12, 24), a quiz is due now.
	if uc.Progress > 0 && uc.Progress%WordsPerQuizBlock == 0 {
		// Check if they already passed the quiz for *this specific block*
		var passedAttemptForThisBlock models.QuizAttempt
		err := db.Joins("JOIN quizzes ON quizzes.id = quiz_attempts.quiz_id").
			Where("quiz_attempts.user_id = ? AND quizzes.course_id = ? AND quizzes.trigger_progress = ? AND quiz_attempts.is_completed = ? AND quiz_attempts.score >= ?",
				user.ID, course.ID, uc.Progress, true, QuizPassThresholdCorrectAnswers).
			First(&passedAttemptForThisBlock).Error

		if errors.Is(err, gorm.ErrRecordNotFound) { // Not found a passed attempt means quiz is due or re-due
			return startQuizFlow(ctx, db, user.ID, course.ID, uc.Progress)
		}
	}

	if err := sendCourseWord(ctx, db, course.ID, uc.Progress); err != nil {
		if strings.Contains(err.Error(), "به انتهای دوره رسیده باشید") {
			ctx.Send("شما تمام کلمات این دوره را مطالعه کرده‌اید!", keyboards.Main())
			return nil
		}
		return err
	}
	return ctx.Send("برای دریافت کلمه بعدی روی 'کلمه بعدی' کلیک کنید.", keyboards.Course())
}

func handleNextWord(ctx telebot.Context, db *gorm.DB) error {
	user, err := fetchUser(ctx, db)
	if err != nil {
		fmt.Printf("[ERROR] handleNextWord: Error fetching user: %v\n", err)
		return ctx.Send("مشکلی در یافتن کاربر پیش آمد.")
	}

	const prefix = "courseId:"
	if !strings.HasPrefix(user.LastMenu, prefix) {
		return ctx.Send("شما در حال حاضر در یک دوره فعال نیستید. لطفا یک دوره را انتخاب کنید.", keyboards.Main())
	}
	idStr := strings.TrimPrefix(user.LastMenu, prefix)
	cid, errConv := strconv.ParseUint(idStr, 10, 64)
	if errConv != nil {
		fmt.Printf("[ERROR] handleNextWord: Error parsing courseID from LastMenu '%s': %v\n", user.LastMenu, errConv)
		return ctx.Send("خطا در خواندن شناسه دوره.")
	}
	courseID := uint(cid)

	var uc models.UserCourse
	if err := db.Where("user_id = ? AND course_id = ?", user.ID, courseID).First(&uc).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// This case should ideally be handled when user first enters a course.
			// If they hit "Next Word" without a UserCourse, something is off.
			fmt.Printf("[WARN] handleNextWord: UserCourse not found for UserID %d, CourseID %d. Guiding to main menu.\n", user.ID, courseID)
			return ctx.Send("اطلاعات پیشرفت شما در این دوره یافت نشد. لطفا دوره را مجددا شروع کنید.", keyboards.Main())
		}
		fmt.Printf("[ERROR] handleNextWord: Error fetching UserCourse for UserID %d, CourseID %d: %v\n", user.ID, courseID, err)
		return ctx.Send("خطا در یافتن اطلاعات پیشرفت شما در دوره.")
	}

	// check for active quiz attempt
	blockEndProgress := ((uc.Progress-1)/WordsPerQuizBlock + 1) * WordsPerQuizBlock
	if uc.Progress == 0 {
		blockEndProgress = WordsPerQuizBlock
	}
	activeAttemptForBlock, _ := findActiveQuizAttemptForBlock(db, user.ID, courseID, blockEndProgress)
	if activeAttemptForBlock != nil {
		ctx.Send("شما یک آزمون نیمه‌تمام برای این بخش از دوره دارید. ادامه می‌دهیم...")
		return sendCurrentQuizQuestion(ctx, db, activeAttemptForBlock.ID)
	}

	// --- Advance Progress ---
	previousProgress := uc.Progress
	uc.Progress++
	log.Printf("[DEBUG] handleNextWord: UserID %d, CourseID %d. Progress advanced from %d to %d.", user.ID, courseID, previousProgress, uc.Progress)

	if err := db.Model(&uc).Update("progress", uc.Progress).Error; err != nil {
		// Rollback in-memory progress if DB update fails
		uc.Progress = previousProgress
		fmt.Printf("[ERROR] handleNextWord: Error updating progress for UserCourseID %d: %v\n", uc.ID, err)
		return ctx.Send("خطا در به‌روز‌رسانی پیشرفت دوره.")
	}

	// --- Send the Content for the New Current Word ---
	errSendWord := sendCourseWord(ctx, db, courseID, uc.Progress) // e.g., uc.Progress is now 12
	if errSendWord != nil {
		if strings.Contains(errSendWord.Error(), "به انتهای دوره رسیده باشید") { // Assuming sendCourseWord returns this specific error string
			log.Printf("[DEBUG] handleNextWord: UserID %d reached end of course %d at/after progress %d.", user.ID, courseID, uc.Progress)
			// User has seen the last word (or tried to go beyond).
			// A quiz might be due for the block they *just completed*.
			// uc.Progress is currently the number *after* the last word.
			// So, the block they completed ended at uc.Progress - 1.
			progressAtActualEndOfBlock := uc.Progress - 1
			if progressAtActualEndOfBlock > 0 && progressAtActualEndOfBlock%WordsPerQuizBlock == 0 {
				log.Printf("[DEBUG] handleNextWord: End of course, quiz due for block ending at %d.", progressAtActualEndOfBlock)
				ctx.Send(fmt.Sprintf("شما تمام کلمات این بخش (%d کلمه) را مطالعه کرده‌اید! آماده آزمون شوید.", WordsPerQuizBlock))
				return startQuizFlow(ctx, db, user.ID, courseID, progressAtActualEndOfBlock)
			}
			return ctx.Send("شما تمام کلمات این دوره را مطالعه کرده‌اید! عالی بود!", keyboards.Main())
		}
		// Other error from sendCourseWord
		fmt.Printf("[ERROR] handleNextWord: Error from sendCourseWord for progress %d: %v\n", uc.Progress, errSendWord)
		return errSendWord // Propagate error
	}

	// --- AFTER successfully sending the word, check if a NEW quiz is due ---
	// e.g., uc.Progress is 12. Word 12 content has just been sent.
	if uc.Progress > 0 && uc.Progress%WordsPerQuizBlock == 0 {
		log.Printf("[DEBUG] handleNextWord: UserID %d reached quiz trigger point. Progress: %d. Starting quiz flow.", user.ID, uc.Progress)
		// Before starting, ensure they haven't *just* passed this exact block's quiz.
		// (This check is also in handleStartOrResumeCourse, but good for belt-and-suspenders
		// if user somehow gets here directly after passing)
		var passedQuizForThisBlock models.QuizAttempt
		errPassed := db.Joins("JOIN quizzes ON quizzes.id = quiz_attempts.quiz_id").
			Where("quiz_attempts.user_id = ? AND quizzes.course_id = ? AND quizzes.trigger_progress = ? AND quiz_attempts.is_completed = ? AND quiz_attempts.score >= ?",
				user.ID, courseID, uc.Progress, true, QuizPassThresholdCorrectAnswers).
			First(&passedQuizForThisBlock).Error

		if errors.Is(errPassed, gorm.ErrRecordNotFound) {
			// Not passed yet, or no attempt. Start the quiz.
			return startQuizFlow(ctx, db, user.ID, courseID, uc.Progress)
		} else if errPassed != nil {
			log.Printf("[ERROR] handleNextWord: DB error checking for passed quiz before starting new one: %v", errPassed)
			// Proceed cautiously or inform user? For now, let's assume we should not start a quiz if this check fails.
			return ctx.Send("خطایی در بررسی وضعیت آزمون رخ داد، لطفا دوباره «کلمه بعدی» را بزنید.")
		}
		// If errPassed is nil, they have already passed this quiz block.
		// This state should ideally mean uc.Progress would have been advanced beyond this point already
		// by the quiz finalization logic. If they are here, it's a bit unusual.
		// Send "next word" prompt, allowing them to move to uc.Progress + 1.
		log.Printf("[WARN] handleNextWord: UserID %d is at progress %d but seems to have already passed this quiz block. Prompting for next word.", user.ID, uc.Progress)
	}

	log.Printf("[DEBUG] handleNextWord: UserID %d, no new quiz triggered. Prompting for next word. Current progress: %d", user.ID, uc.Progress)
	return ctx.Send("برای دریافت کلمه بعدی روی 'کلمه بعدی' کلیک کنید.", keyboards.Course())
}
