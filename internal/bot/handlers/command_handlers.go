package handlers

import (
	"fmt"
	"log"

	"github.com/2000ostd/enssi-tel-bot/internal/bot/formatters"
	"github.com/2000ostd/enssi-tel-bot/internal/bot/keyboards"
	"github.com/2000ostd/enssi-tel-bot/internal/services"
	"gopkg.in/telebot.v4"
)

// HandleStartCommand processes the /start command.
func HandleStartCommand(c telebot.Context, appServices *services.AppServices) error {
	dbUser, err := appServices.User().GetOrCreateUserByTelegramID(
		c.Sender().ID,
		c.Sender().Username,
		c.Sender().FirstName,
		c.Sender().LastName,
	)
	if err != nil {
		log.Printf("[HandleStartCommand] Error ensuring user from service for TelegramID %d: %v", c.Sender().ID, err)
		return c.Send("متاسفم، مشکلی در شروع گفتگو پیش آمد. لطفا دوباره با /start تلاش کنید.")
	}

	// Check for mandatory daily review
	reviewHandled, reviewErr := CheckAndInitiateReview(c, appServices, dbUser)
	if reviewErr != nil {
		// Error already logged by CheckAndInitiateReview, send a generic message to user
		return SendServiceError(c, "checking for daily review", reviewErr)
	}
	if reviewHandled {
		return nil // Review process has taken over
	}

	// Proceed with normal start command logic
	err = appServices.User().UpdateUserLastMenu(dbUser.ID, StateMain)
	if err != nil {
		log.Printf("[HandleStartCommand] Error updating last menu to 'main' for UserID %d: %v", dbUser.ID, err)
		// Non-fatal for sending the start message, but log it.
	}

	// Fetch start text - assuming it's defined elsewhere or using a fallback
	// For example, from a localization service or embedded texts.
	// startText := appServices.Text().Get("start_command_greeting", dbUser.Profile.FirstName)
	startText := fmt.Sprintf(
		"سلام %s! 👋 به ربات آموزش زبان خوش آمدید. برای شروع یادگیری از دکمه‌های زیر استفاده کنید.",
		dbUser.Profile.FirstName,
	)
	if dbUser.Profile.FirstName == "" { // Fallback if first name is empty
		startText = "سلام! 👋 به ربات آموزش زبان خوش آمدید. برای شروع یادگیری از دکمه‌های زیر استفاده کنید."
	}

	return c.Send(formatters.EscapeMarkdownV2(startText), keyboards.MainMenu, telebot.ModeMarkdownV2)
}

// HandleMyProfileCommand displays the user's profile.
func HandleMyProfileCommand(c telebot.Context, appServices *services.AppServices) error {
	dbUser, err := appServices.User().GetOrCreateUserByTelegramID(
		c.Sender().ID,
		c.Sender().Username,
		c.Sender().FirstName,
		c.Sender().LastName,
	)
	if err != nil {
		return SendServiceError(c, "fetching user for profile", err)
	}

	// Check for mandatory daily review
	reviewHandled, reviewErr := CheckAndInitiateReview(c, appServices, dbUser)
	if reviewErr != nil {
		return SendServiceError(c, "checking for daily review (profile)", reviewErr)
	}
	if reviewHandled {
		return nil // Review process has taken over
	}

	// Proceed with profile display
	userProfileView, err := appServices.User().GetUserProfile(dbUser.ID)
	if err != nil {
		return SendServiceError(c, "fetching user profile data", err)
	}

	formattedProfile := formatters.FormatUserProfile(userProfileView)
	// TODO: Add keyboard for profile actions (e.g., edit, view achievements)
	// For now, sending with MainMenu, but a ProfileMenu would be better.
	return c.Send(formattedProfile, keyboards.MainMenu, telebot.ModeMarkdownV2)
}
