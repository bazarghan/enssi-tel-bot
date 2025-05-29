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
	// 1) Load the user from DB
	tgID := ctx.Sender().ID
	var user models.User
	if err := db.
		Where("telegram_id = ?", tgID).
		First(&user).Error; err != nil {
		return ctx.Send("مشکلی در یافتن کاربر پیش آمد")
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

	return handleStartOrResumeCourse(ctx, db, &user, &course, &uc)

}

// This function checks if a user is currently in an active (uncompleted) quiz for ANY course.
// For a more focused check (current course only), pass courseID.
func findAnyActiveQuizAttempt(db *gorm.DB, userID uint) (*models.QuizAttempt, error) {
	var attempt models.QuizAttempt
	err := db.Where("user_id = ? AND is_completed = ?", userID, false).
		Order("created_at DESC").First(&attempt).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // No active quiz attempt found
		}
		return nil, err // Other DB error
	}
	return &attempt, nil
}

func handleStartOrResumeCourse(ctx telebot.Context, db *gorm.DB, user *models.User, course *models.Course, uc *models.UserCourse) error {
	// Check if user has an active, uncompleted quiz FOR THIS SPECIFIC BLOCK they are landing on.
	// A quiz is due after completing a block, i.e. uc.Progress is N*WordsPerQuizBlock
	// Or if their current progress means they are *within* a block for which they have an active quiz.

	// Calculate the end-progress of the block the user is currently in or has just completed.
	// Example: progress 1-11 -> block ends at 12. progress 12 -> block ends at 12.
	// progress 13-23 -> block ends at 24. progress 24 -> block ends at 24.
	blockEndProgress := ((uc.Progress-1)/WordsPerQuizBlock + 1) * WordsPerQuizBlock
	if uc.Progress == 0 {
		blockEndProgress = WordsPerQuizBlock
	} // If progress is 0 (new course), first block ends at 12

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
		// If found a passed attempt or other error, proceed to send word.
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

	// --- CHECK 1: Is there ANY active (uncompleted) quiz for this user? ---
	anyActiveAttempt, errFindActive := findAnyActiveQuizAttempt(db, user.ID)
	if errFindActive != nil {
		// This is an actual DB error, not just "not found"
		fmt.Printf("[ERROR] handleNextWord: Error checking for any active quiz: %v. UserID: %d\n", errFindActive, user.ID)
		return ctx.Send("خطایی در بررسی وضعیت آزمون شما رخ داد. لطفا دوباره تلاش کنید.")
	}

	if anyActiveAttempt != nil {
		// User has an active quiz. Resend the current question of that quiz and stop.
		log.Printf("[DEBUG] handleNextWord: UserID %d has active quiz attempt ID %d. Resending quiz question.", user.ID, anyActiveAttempt.ID)
		ctx.Send("لطفا ابتدا آزمون فعال خود را تکمیل کنید.")
		return sendCurrentQuizQuestion(ctx, db, anyActiveAttempt.ID) // This should handle resuming the quiz display
	}

	// --- If NO active quiz, proceed to the next word ---
	log.Printf("[DEBUG] handleNextWord: UserID %d has no active quiz. Proceeding to next word.", user.ID)

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
