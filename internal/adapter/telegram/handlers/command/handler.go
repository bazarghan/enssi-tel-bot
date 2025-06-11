package command

import (
	"context"
	"fmt"
	"log"

	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/dto"
	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/formatters"
	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/keyboards"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/user"
	registerCmd "github.com/2000ostd/enssi-tel-bot/internal/usecase/commands/user"
	getProfileQry "github.com/2000ostd/enssi-tel-bot/internal/usecase/queries/user"
	"github.com/2000ostd/enssi-tel-bot/pkg/tgmarkdown"
	"gopkg.in/telebot.v4"
)

// Handler holds dependencies for command handlers.
type Handler struct {
	registerUser registerCmd.RegisterUserHandler
	getProfile   getProfileQry.GetProfileHandler
	userRepo     user.Repository
}

// NewHandler creates a new command handler.
func NewHandler(
	registerUser registerCmd.RegisterUserHandler,
	getProfile getProfileQry.GetProfileHandler,
	userRepo user.Repository,
) *Handler {
	return &Handler{
		registerUser: registerUser,
		getProfile:   getProfile,
		userRepo:     userRepo,
	}
}

// HandleStart processes the /start command.
func (h *Handler) HandleStart(c telebot.Context) error {
	cmd := registerCmd.RegisterUserCommand{
		TelegramID: c.Sender().ID,
		Username:   c.Sender().Username,
		FirstName:  c.Sender().FirstName,
		LastName:   c.Sender().LastName,
	}

	result, err := h.registerUser.Handle(context.Background(), cmd)
	if err != nil {
		log.Printf("[HandleStart] Error ensuring user for TelegramID %d: %v", c.Sender().ID, err)
		return c.Send("متاسفم، مشکلی در شروع گفتگو پیش آمد. لطفا دوباره با /start تلاش کنید.")
	}

	// TODO: Logic to cancel active quiz needs to be re-implemented in a later slice.

	if err := h.userRepo.UpdateLastMenu(context.Background(), result.User.ID, "main"); err != nil {
		log.Printf("[HandleStart] Error updating last menu for UserID %d: %v", result.User.ID, err)
	}

	startText := fmt.Sprintf(
		"سلام %s! 👋 به ربات آموزش زبان خوش آمدید. برای شروع یادگیری از دکمه‌های زیر استفاده کنید.",
		result.User.Profile.FirstName,
	)
	if result.User.Profile.FirstName == "" {
		startText = "سلام! 👋 به ربات آموزش زبان خوش آمدید. برای شروع یادگیری از دکame‌های زیر استفاده کنید."
	}

	mainMenuKeyboard := keyboards.NewMainMenu(result.User.IsAdmin)
	return c.Send(tgmarkdown.Escape(startText), mainMenuKeyboard, telebot.ModeMarkdownV2)
}

// HandleMyProfile processes the /myprofile command.
func (h *Handler) HandleMyProfile(c telebot.Context) error {
	// The user object should be in the context from the middleware.
	ctxUser, ok := c.Get("dbUser").(user.User)
	if !ok {
		log.Printf("[HandleMyProfile] Critical: User not found in context for telegram ID %d", c.Sender().ID)
		return c.Send("خطا در پردازش اطلاعات کاربر.")
	}

	query := getProfileQry.GetProfileQuery{UserID: ctxUser.ID}
	result, err := h.getProfile.Handle(context.Background(), query)
	if err != nil {
		log.Printf("[HandleMyProfile] Could not get user profile for user ID %d: %v", ctxUser.ID, err)
		return c.Send("متاسفانه در دریافت اطلاعات پروفایل مشکلی پیش آمد.")
	}

	// Map usecase result to a presentation DTO (they are the same in this slice)
	profileDTO := dto.UserProfile{
		FirstName:     result.FirstName,
		Username:      result.Username,
		LastName:      result.LastName,
		Score:         result.Score,
		WordsStudied:  result.WordsStudied,
		CoursesActive: result.CoursesActive,
	}

	formattedProfile := formatters.FormatUserProfile(profileDTO)

	h.userRepo.UpdateLastMenu(context.Background(), ctxUser.ID, "profile_menu")

	return c.Send(formattedProfile, telebot.ModeMarkdownV2)
}
