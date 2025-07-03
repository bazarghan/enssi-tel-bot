package message

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/bits-and-blooms/bitset"

	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/dto"
	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/formatters"
	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/keyboards"
	sc "github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/stateconstants"
	ui "github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/uiconstants"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/achievement"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/user"
	"github.com/2000ostd/enssi-tel-bot/internal/platform/observability/logger"
	"github.com/2000ostd/enssi-tel-bot/pkg/tgmarkdown"

	achQueries "github.com/2000ostd/enssi-tel-bot/internal/usecase/queries/achievement"
	getProfileQry "github.com/2000ostd/enssi-tel-bot/internal/usecase/queries/user"
	"gopkg.in/telebot.v4"
)

// ProfileHandler holds dependencies for profile-related message handlers.
type ProfileHandler struct {
	logger             logger.Logger
	getProfile         getProfileQry.GetProfileHandler
	getAllAchievements achQueries.GetAllHandler
	userRepo           user.Repository
	achRepo            achievement.Repository
	imgSvc             achievement.ImageGenerator
}

// NewProfileHandler creates a new profile handler.
func NewProfileHandler(
	appLogger logger.Logger,
	getProfile getProfileQry.GetProfileHandler,
	getAllAchievements achQueries.GetAllHandler,
	userRepo user.Repository,
	achRepo achievement.Repository,
	imgSvc achievement.ImageGenerator,
) *ProfileHandler {
	return &ProfileHandler{
		logger:             appLogger,
		getProfile:         getProfile,
		getAllAchievements: getAllAchievements,
		userRepo:           userRepo,
		achRepo:            achRepo,
		imgSvc:             imgSvc,
	}
}

// Handle routes incoming profile-related messages based on user state.
func (h *ProfileHandler) Handle(c telebot.Context, u user.User, userInput string) error {
	stateBase, _, _ := strings.Cut(u.LastMenu, ":")

	switch {
	case stateBase == sc.StateProfileMenu:
		if userInput == ui.BtnViewAchievementsText {
			return h.handleViewMyAchievements(c, u)
		}
	case u.LastMenu == sc.StateAchievementList:
		if userInput == ui.BtnReturnToProfileText {
			return h.handleReturnToProfile(c, u)
		} else {
			return h.handleAchievementSelection(c, u, userInput)
		}
	}
	return nil // Should not happen if routed correctly
}

func (h *ProfileHandler) handleGetProfile(c telebot.Context, u user.User) error {

	query := getProfileQry.GetProfileQuery{UserID: u.ID}
	result, err := h.getProfile.Handle(context.Background(), query)
	if err != nil {
		h.logger.Error("Could not get user profile", "user_id", u.ID, "error", err)
		return c.Send("متاسفانه در دریافت اطلاعات پروفایل مشکلی پیش آمد.")
	}

	// Map usecase result to a presentation DTO
	profileDTO := dto.UserProfile{
		FirstName:     result.FirstName,
		Username:      result.Username,
		LastName:      result.LastName,
		Score:         result.Score,
		WordsStudied:  result.WordsStudied,
		CoursesActive: result.CoursesActive,
	}

	formattedProfile := formatters.FormatUserProfile(profileDTO)

	h.userRepo.UpdateLastMenu(context.Background(), u.ID, sc.StateProfileMenu)

	// We send the profile message and the profile menu keyboard
	return c.Send(formattedProfile, keyboards.ProfileMenuKeyboard(), telebot.ModeMarkdownV2)
}

func (h *ProfileHandler) handleViewMyAchievements(c telebot.Context, u user.User) error {
	// 1. Call the use case to get all defined achievements.
	allAchievements, err := h.getAllAchievements.Handle(context.Background())
	if err != nil {
		h.logger.Error("Failed to get achievements", "user_id", u.ID, "error", err)
		return c.Send("متاسفانه در دریافت لیست دستاوردها مشکلی پیش آمد.")
	}

	// 2. Map the use case result to the DTO needed for the keyboard.
	// We are listing all achievements; the specific "EarnedOn" date isn't needed here.
	achDTOs := make([]dto.AchievementView, len(allAchievements))
	for i, ach := range allAchievements {
		achDTOs[i] = dto.AchievementView{
			ID:          ach.ID,
			Title:       ach.Title,
			Description: ach.Description,
		}
	}

	// 3. Generate the keyboard with the list of achievements.
	kb := keyboards.AchievementsListKeyboard(achDTOs)

	// 4. Update the user's state so the bot knows they are in the achievements menu.
	h.userRepo.UpdateLastMenu(context.Background(), u.ID, sc.StateAchievementList)

	// 5. Send the message with the inline keyboard.
	return c.Send("می توانید با کلیک بر روی هر دستاورد، پیشرفت خود را مشاهده کنید:", kb)
}

func (h *ProfileHandler) handleAchievementSelection(c telebot.Context, u user.User, userInput string) error {
	// Assume any other button is an achievement title
	// 1. Clean the button text to get the real title
	cleanTitle := strings.TrimPrefix(userInput, "🏆 ")

	// 2. Find the achievement by its title
	ach, err := h.achRepo.FindByTitle(context.Background(), cleanTitle)
	if err != nil {
		h.logger.Error("Could not find achievement by title", "title", cleanTitle, "error", err)
		return nil // Ignore if the title is not found
	}

	// 3. Re-implement the logic to generate and send the image
	userAch, err := h.achRepo.GetUserAchievement(context.Background(), u.ID, ach.ID)
	if err != nil {
		if errors.Is(err, achievement.ErrUserAchNotFound) {
			userAch = achievement.UserAchievement{
				UserID: u.ID, AchievementID: ach.ID, State: bitset.New(ach.TotalItems),
			}
		} else {
			h.logger.Error("Could not get user achievement progress", "error", err)
			return c.Send("Could not retrieve achievement progress.")
		}
	}

	generatedPath, err := h.imgSvc.Generate(ach.ImageURL, userAch.State, ach.GridWidth, ach.GridHeight)
	if err != nil {
		h.logger.Error("Failed to generate achievement image", "error", err)
		return c.Send("Could not create achievement image.")
	}
	defer os.Remove(generatedPath)

	caption := fmt.Sprintf("🏆 *%s*\n\n`%s`", tgmarkdown.Escape(ach.Title), tgmarkdown.Escape(ach.Description))
	photo := &telebot.Photo{File: telebot.FromDisk(generatedPath), Caption: caption}

	_, err = c.Bot().Send(c.Chat(), photo, telebot.ModeMarkdownV2)
	return err
}

func (h *ProfileHandler) handleReturnToProfile(c telebot.Context, u user.User) error {
	// This is the same logic from your /myprofile command
	query := getProfileQry.GetProfileQuery{UserID: u.ID}
	result, err := h.getProfile.Handle(context.Background(), query)
	if err != nil {
		return c.Send("Could not get profile.")
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
	h.userRepo.UpdateLastMenu(context.Background(), u.ID, sc.StateProfileMenu)
	return c.Send(formattedProfile, keyboards.ProfileMenuKeyboard(), telebot.ModeMarkdownV2)
}
