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
	// User is already GetOrCreate'd by middleware, and activity updated.
	// We need the user's DB ID to update their LastMenu.
	dbUser, err := appServices.User().GetOrCreateUserByTelegramID(
		c.Sender().ID,
		c.Sender().Username,
		c.Sender().FirstName,
		c.Sender().LastName,
	)
	if err != nil {
		// Middleware should have caught critical errors, but double check.
		log.Printf("[HandleStartCommand] Error ensuring user from service for TelegramID %d: %v", c.Sender().ID, err)
		return c.Send("Sorry, there was a problem starting our conversation. Please try `/start` again.")
	}

	err = appServices.User().UpdateUserLastMenu(dbUser.ID, StateMain)
	if err != nil {
		log.Printf("[HandleStartCommand] Error updating last menu to 'main' for UserID %d: %v", dbUser.ID, err)
		// Non-fatal for sending the start message, but log it.
	}

	startText := "this is start text"
	if startText == "" {
		log.Println("[HandleStartCommand] Warning: 'start' text not found in embedded texts. Using fallback.")
		startText = fmt.Sprintf("Welcome, %s!", formatters.EscapeMarkdownV2(dbUser.Profile.FirstName))
	}

	return c.Send(startText, keyboards.MainMenu, telebot.ModeMarkdownV2)
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
		return sendServiceError(c, "fetching user for profile", err)
	}

	userProfileView, err := appServices.User().GetUserProfile(dbUser.ID)
	if err != nil {
		return sendServiceError(c, "fetching user profile data", err)
	}

	formattedProfile := formatters.FormatUserProfile(userProfileView)
	// TODO: Add keyboard for profile actions (e.g., edit, view achievements)
	return c.Send(formattedProfile, keyboards.MainMenu, telebot.ModeMarkdownV2)
}
