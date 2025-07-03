package message

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/dto"
	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/formatters"
	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/keyboards"
	sc "github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/stateconstants"
	ui "github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/uiconstants"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/quiz"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/user"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/word"
	"github.com/2000ostd/enssi-tel-bot/internal/platform/observability/logger"
	quizCmd "github.com/2000ostd/enssi-tel-bot/internal/usecase/commands/quiz"
	courseQueries "github.com/2000ostd/enssi-tel-bot/internal/usecase/queries/course"
	reviewQueries "github.com/2000ostd/enssi-tel-bot/internal/usecase/queries/review"
	"gopkg.in/telebot.v4"
)

// MainMenuHandler holds dependencies for main menu message handlers.
type MainMenuHandler struct {
	logger           logger.Logger
	listCourses      courseQueries.ListCoursesHandler
	userRepo         user.Repository
	quizRepo         quiz.Repository
	wordRepo         word.Repository
	createReviewQuiz quizCmd.CreateReviewQuizHandler
	hasPendingReview reviewQueries.Handler
	profileHandler   *ProfileHandler
}

// NewMainMenuHandler creates a new main menu handler.
func NewMainMenuHandler(
	appLogger logger.Logger,
	listCourses courseQueries.ListCoursesHandler,
	userRepo user.Repository,
	quizRepo quiz.Repository,
	wordRepo word.Repository,
	createReviewQuiz quizCmd.CreateReviewQuizHandler,
	hasPendingReview reviewQueries.Handler,
	profileHandler *ProfileHandler,
) *MainMenuHandler {
	return &MainMenuHandler{
		logger:           appLogger,
		listCourses:      listCourses,
		userRepo:         userRepo,
		quizRepo:         quizRepo,
		wordRepo:         wordRepo,
		createReviewQuiz: createReviewQuiz,
		hasPendingReview: hasPendingReview,
		profileHandler:   profileHandler,
	}
}

// Handle routes incoming main menu messages based on user input.
func (h *MainMenuHandler) Handle(c telebot.Context, u user.User, userInput string) error {
	if u.LastMenu == sc.StateMain {
		if userInput == ui.BtnStartLearningText {
			return h.handleStartLearning(c, u)
		}
		if userInput == ui.BtnMyProfileText {
			return h.profileHandler.handleGetProfile(c, u)
		}
		if userInput == ui.BtnDailyReviewText {
			return h.handleDailyReview(c, u)
		}
		if userInput == ui.BtnAdminPanelText {
			return h.handleAdminPanel(c, u)
		}
	}
	return nil // Should not happen if routed correctly
}

func (h *MainMenuHandler) handleStartLearning(c telebot.Context, u user.User) error {

	reviewQuery := reviewQueries.HasPendingReviewQuery{UserID: u.ID}
	hasPendingReview, err := h.hasPendingReview.Handle(context.Background(), reviewQuery)
	if err != nil {
		// Log the error but proceed with a non-review menu as a safe default
		h.logger.Error("Could not check for pending review", "user_id", u.ID, "error", err)
		hasPendingReview = false
	}

	if hasPendingReview {
		// If a pending review exists, block the user and tell them what to do.
		blockMsg := "شما یک آزمون مرور روزانه برای انجام دادن دارید. لطفا ابتدا آن را با استفاده از دکمه 'مرور روزانه' تکمیل کنید."
		return c.Send(blockMsg)
	}

	query := courseQueries.ListCoursesQuery{UserID: u.ID}
	courses, err := h.listCourses.Handle(context.Background(), query)
	if err != nil {
		h.logger.Error("Error listing courses", "user_id", u.ID, "error", err)
		return c.Send("متاسفانه در دریافت لیست دوره‌ها مشکلی پیش آمد.")
	}

	courseDTOs := make([]dto.CourseSummary, len(courses))
	for i, co := range courses {
		courseDTOs[i] = dto.CourseSummary{
			ID:                 co.ID,
			PersianTitle:       co.PersianTitle,
			ProgressPercentage: co.ProgressPercentage,
			IsCompleted:        co.IsCompleted,
		}
	}

	msg := formatters.FormatCourseListMessage(courseDTOs)
	kb := keyboards.CourseListKeyboard(courseDTOs)

	if err := h.userRepo.UpdateLastMenu(context.Background(), u.ID, sc.StateCourseList); err != nil {
		h.logger.Error("Failed to update user state", "user_id", u.ID, "error", err)
	}

	return c.Send(msg, kb, telebot.ModeMarkdownV2)
}

func (h *MainMenuHandler) handleDailyReview(c telebot.Context, u user.User) error {

	// 1. First, check for any old pending review and delete it to ensure a fresh start.
	if oldAttempt, err := h.quizRepo.FindPendingReviewAttempt(context.Background(), u.ID); err == nil && oldAttempt.ID != 0 {
		h.logger.Info("Found and deleting stale review attempt", "attempt_id", oldAttempt.ID, "user_id", u.ID)
		h.quizRepo.DeleteAttempt(context.Background(), oldAttempt.ID)
	}

	// 2. Now, find ALL words that are currently due for review.
	wordsToReview, err := h.wordRepo.GetWordsDueForReview(context.Background(), u.ID, time.Now())
	if err != nil {
		h.logger.Error("Failed to get words for review", "user_id", u.ID, "error", err)
		return c.Send("خطا در آماده سازی آزمون مرور شما.")
	}

	if len(wordsToReview) == 0 {
		return c.Send("شما در حال حاضر هیچ کلمه ای برای مرور ندارید. آفرین!")
	}

	// 3. Create a brand new, fresh quiz with these words.
	reviewCmd := quizCmd.CreateReviewQuizCommand{UserID: u.ID, WordsToReview: wordsToReview}
	newQuiz, err := h.createReviewQuiz.Handle(context.Background(), reviewCmd)
	if err != nil {
		h.logger.Error("Failed to create fresh review quiz", "user_id", u.ID, "error", err)
		return c.Send("خطا در ساخت آزمون مرور شما.")
	}

	// 4. Start the new quiz by sending the first question.
	attempt := newQuiz.QuizAttempt
	question := attempt.Questions[attempt.CurrentQuestionIndex]

	newState := fmt.Sprintf("in_quiz:0:%d", attempt.ID)
	h.userRepo.UpdateLastMenu(context.Background(), u.ID, newState)

	rand.Shuffle(len(question.Options), func(i, j int) {
		question.Options[i], question.Options[j] = question.Options[j], question.Options[i]
	})
	msg := formatters.FormatQuizQuestion(question, attempt.CurrentQuestionIndex, len(attempt.Questions))
	kb := keyboards.QuizQuestionOptionsKeyboard(question.Options, attempt.ID)
	sentMsg, err := c.Bot().Send(c.Chat(), msg, kb, telebot.ModeMarkdownV2)
	if err == nil {
		h.quizRepo.UpdateMessageID(context.Background(), attempt.ID, sentMsg.ID)
	}
	return err
}

func (h *MainMenuHandler) handleAdminPanel(c telebot.Context, u user.User) error {

	if !u.IsAdmin {
		return nil // Ignore if a non-admin somehow sends this text
	}
	// Set the user's state to the admin panel
	h.userRepo.UpdateLastMenu(context.Background(), u.ID, "admin_panel")
	// Send the new admin keyboard
	return c.Send("به پنل ادمین خوش آمدید.", keyboards.AdminPanelKeyboard())

}

func (h *MainMenuHandler) handleReturnToMainMenu(c telebot.Context, u user.User) error {

	// 1. Check if the user was in a quiz using their last menu state.
	if strings.HasPrefix(u.LastMenu, "in_quiz:") {
		parts := strings.Split(u.LastMenu, ":")
		if len(parts) == 3 {
			attemptID, err := strconv.ParseUint(parts[2], 10, 64)
			if err == nil {
				// 2. Fetch the quiz attempt from the database to get the message ID.
				attempt, err := h.quizRepo.GetAttempt(context.Background(), uint(attemptID))
				if err != nil {
					h.logger.Error("Could not get quiz attempt to edit message", "attempt_id", attemptID, "error", err)
				} else if attempt.CurrentQuestionMessageID != 0 {
					// 3. Edit the original quiz message to show it's paused.
					//	 By not providing a new keyboard, the inline keyboard is automatically removed.
					pausedMsg := "آزمون متوقف شد. شما به منوی اصلی بازگشتید."

					// We need to create a telebot.Message object to edit it.
					// The Chat ID is important.
					messageToEdit := &telebot.Message{
						ID:   attempt.CurrentQuestionMessageID,
						Chat: c.Chat(),
					}

					if _, err := c.Bot().Edit(messageToEdit, pausedMsg); err != nil {
						// This error is not critical, the user can still proceed.
						h.logger.Warn("Failed to edit old quiz message", "message_id", attempt.CurrentQuestionMessageID, "error", err)
					}
				}
			}
		}
	}

	// --- THIS IS THE REFACTORED CODE ---
	query := reviewQueries.HasPendingReviewQuery{UserID: u.ID}
	hasPendingReview, err := h.hasPendingReview.Handle(context.Background(), query)
	if err != nil {
		// Log the error but proceed with a non-review menu as a safe default
		h.logger.Error("Could not check for pending review", "user_id", u.ID, "error", err)
		hasPendingReview = false
	}
	// --- END OF REFACTORED CODE ---

	if err := h.userRepo.UpdateLastMenu(context.Background(), u.ID, sc.StateMain); err != nil {
		h.logger.Error("Failed to update user state", "user_id", u.ID, "error", err)
	}
	return c.Send("به منوی اصلی بازگشتید.", keyboards.NewMainMenu(u.IsAdmin, hasPendingReview))
}
