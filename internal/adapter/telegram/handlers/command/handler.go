package command

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/dto"
	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/formatters"
	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/keyboards"

	"github.com/2000ostd/enssi-tel-bot/internal/domain/quiz"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/user"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/word"

	"github.com/2000ostd/enssi-tel-bot/internal/platform/observability/logger"
	registerCmd "github.com/2000ostd/enssi-tel-bot/internal/usecase/commands/user"
	reviewQueries "github.com/2000ostd/enssi-tel-bot/internal/usecase/queries/review"
	getProfileQry "github.com/2000ostd/enssi-tel-bot/internal/usecase/queries/user"
	"github.com/2000ostd/enssi-tel-bot/pkg/tgmarkdown"
	"gopkg.in/telebot.v4"
)

// Handler holds dependencies for command handlers.
type Handler struct {
	logger           logger.Logger
	registerUser     registerCmd.RegisterUserHandler
	getProfile       getProfileQry.GetProfileHandler
	userRepo         user.Repository
	quizRepo         quiz.Repository
	wordRepo         word.Repository
	hasPendingReview reviewQueries.Handler
}

// NewHandler creates a new command handler.
func NewHandler(
	appLogger logger.Logger,
	registerUser registerCmd.RegisterUserHandler,
	getProfile getProfileQry.GetProfileHandler,
	userRepo user.Repository,
	quizRepo quiz.Repository,
	wordRepo word.Repository,
	hasPendingReview reviewQueries.Handler,
) *Handler {
	return &Handler{
		logger:           appLogger.With("handler", "command"),
		registerUser:     registerUser,
		getProfile:       getProfile,
		userRepo:         userRepo,
		quizRepo:         quizRepo,
		wordRepo:         wordRepo,
		hasPendingReview: hasPendingReview,
	}
}

// HandleStart processes the /start command.
func (h *Handler) HandleStart(c telebot.Context) error {
	h.logger.Info("Handling /start command", "telegram_id", c.Sender().ID, "username", c.Sender().Username)
	cmd := registerCmd.RegisterUserCommand{
		TelegramID: c.Sender().ID,
		Username:   c.Sender().Username,
		FirstName:  c.Sender().FirstName,
		LastName:   c.Sender().LastName,
	}

	result, err := h.registerUser.Handle(context.Background(), cmd)
	if err != nil {
		h.logger.Error("Error registering or finding user", "error", err, "telegram_id", c.Sender().ID)
		return c.Send("متاسفم، مشکلی در شروع گفتگو پیش آمد. لطفا دوباره با /start تلاش کنید.")
	}

	logger := h.logger.With("user_id", result.User.ID, "telegram_id", c.Sender().ID)

	// --- NEW LOGIC: Check for and pause any active quiz ---
	if strings.HasPrefix(result.User.LastMenu, "in_quiz:") {
		parts := strings.Split(result.User.LastMenu, ":")
		if len(parts) == 3 {
			attemptID, err := strconv.ParseUint(parts[2], 10, 64)
			if err == nil {
				attempt, err := h.quizRepo.GetAttempt(context.Background(), uint(attemptID))
				if err == nil && attempt.CurrentQuestionMessageID != 0 {
					pausedMsg := "آزمون متوقف شد. شما به منوی اصلی بازگشتید."
					messageToEdit := &telebot.Message{
						ID:   attempt.CurrentQuestionMessageID,
						Chat: c.Chat(),
					}
					if _, err := c.Bot().Edit(messageToEdit, pausedMsg); err != nil {
						logger.Warn(
							"Failed to edit old quiz message on /start",
							"error", err,
							"message_id", attempt.CurrentQuestionMessageID,
						)
					}
				}
			}
		}
	}
	// --- END OF NEW LOGIC ---

	if err := h.userRepo.UpdateLastMenu(context.Background(), result.User.ID, "main"); err != nil {
		logger.Error("Error updating last menu", "error", err)
	}

	startText := fmt.Sprintf(
		"سلام %s! 👋 به ربات آموزش زبان خوش آمدید. برای شروع یادگیری از دکمه‌های زیر استفاده کنید.",
		result.User.Profile.FirstName,
	)
	if result.User.Profile.FirstName == "" {
		startText = "سلام! 👋 به ربات آموزش زبان خوش آمدید. برای شروع یادگیری از دکمه‌های زیر استفاده کنید."
	}

	query := reviewQueries.HasPendingReviewQuery{UserID: result.User.ID}
	hasPendingReview, err := h.hasPendingReview.Handle(context.Background(), query)
	if err != nil {
		// Log the error but proceed with a non-review menu as a safe default
		logger.Error("Could not check for pending review", "error", err)
		hasPendingReview = false
	}

	mainMenuKeyboard := keyboards.NewMainMenu(result.User.IsAdmin, hasPendingReview)
	logger.Info("User started bot interaction") // CORRECTED LINE

	return c.Send(tgmarkdown.Escape(startText), mainMenuKeyboard, telebot.ModeMarkdownV2)
}

// HandleMyProfile processes the /myprofile command.
func (h *Handler) HandleMyProfile(c telebot.Context) error {
	ctxUser, ok := c.Get("dbUser").(user.User)
	if !ok {
		h.logger.Error("User not found in context", "telegram_id", c.Sender().ID)
		return c.Send("خطا در پردازش اطلاعات کاربر.")
	}

	logger := h.logger.With("user_id", ctxUser.ID, "telegram_id", c.Sender().ID)
	logger.Info("Handling /myprofile command")

	query := getProfileQry.GetProfileQuery{UserID: ctxUser.ID}
	result, err := h.getProfile.Handle(context.Background(), query)
	if err != nil {
		logger.Error("Could not get user profile", "error", err)
		return c.Send("متاسفانه در دریافت اطلاعات پروفایل مشکلی پیش آمد.")
	}

	profileDTO := dto.UserProfile{
		FirstName:     result.FirstName,
		Username:      result.Username,
		LastName:      result.LastName,
		Score:         result.Score,
		WordsStudied:  result.WordsStudied,
		CoursesActive: result.CoursesActive,
	}

	formattedProfile := formatters.FormatUserProfile(profileDTO)

	if err := h.userRepo.UpdateLastMenu(context.Background(), ctxUser.ID, "profile_menu"); err != nil {
		logger.Error("Failed to update user menu state", "error", err, "menu", "profile_menu")
	}

	logger.Info("Successfully fetched and sent user profile")
	return c.Send(formattedProfile, telebot.ModeMarkdownV2)
}
