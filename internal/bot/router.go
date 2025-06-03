package bot

import (
	"log"
	"strings"

	"github.com/2000ostd/enssi-tel-bot/internal/bot/handlers"
	"github.com/2000ostd/enssi-tel-bot/internal/bot/keyboards" // For button text constants
	"github.com/2000ostd/enssi-tel-bot/internal/services"
	"gopkg.in/telebot.v4"
)

// RegisterRoutes sets up all command, text, and callback handlers for the bot.
func RegisterRoutes(b *telebot.Bot, appServices *services.AppServices) {
	b.Use(UserActivityMiddleware(appServices))

	// --- Command Handlers ---
	b.Handle("/start", func(c telebot.Context) error {
		return handlers.HandleStartCommand(c, appServices)
	})

	// --- Text Handlers for specific Reply Keyboard Buttons ---
	// These are matched before the general telebot.OnText handler.
	b.Handle(keyboards.BtnStartLearning.Text, func(c telebot.Context) error {
		// User already fetched by middleware, but handlers might need the ID explicitly.
		// For simplicity in these direct handlers, we can re-fetch or rely on context if populated.
		// Let's assume handlers get user ID from c.Sender() if middleware doesn't inject *models.User directly.
		currentUser, err := appServices.User().GetOrCreateUserByTelegramID(c.Sender().ID, c.Sender().Username, c.Sender().FirstName, c.Sender().LastName)
		if err != nil {
			log.Printf("[Router] Error getting user for BtnStartLearning: %v", err)
			return c.Send("An error occurred, please try again.")
		}
		return handlers.HandleStartLearningJourney(c, currentUser.ID, appServices)
	})

	b.Handle(keyboards.BtnMyProfile.Text, func(c telebot.Context) error {
		return handlers.HandleMyProfileCommand(c, appServices)
	})

	b.Handle(keyboards.BtnReturnToMainMenu.Text, func(c telebot.Context) error {
		currentUser, err := appServices.User().GetOrCreateUserByTelegramID(c.Sender().ID, c.Sender().Username, c.Sender().FirstName, c.Sender().LastName)
		if err != nil {
			log.Printf("[Router] Error getting user for BtnReturnToMainMenu: %v", err)
			return c.Send("An error occurred, please try again.")
		}
		return handlers.HandleReturnToMainMenu(c, currentUser.ID, appServices)
	})

	b.Handle(keyboards.NextWordButtonText, func(c telebot.Context) error {
		currentUser, err := appServices.User().GetOrCreateUserByTelegramID(c.Sender().ID, c.Sender().Username, c.Sender().FirstName, c.Sender().LastName)
		if err != nil {
			log.Printf("[Router] Error getting user for NextWordButtonText: %v", err)
			return c.Send("An error occurred, please try again.")
		}
		// Course ID needs to be derived from user's current state/LastMenu
		courseID, err := handlers.ParseIDFromState(currentUser.LastMenu, handlers.StateInCoursePrefix)
		if err != nil {
			log.Printf("[Router] NextWordButton: UserID %d, LastMenu '%s' not in 'in_course:ID' state or invalid courseID. Error: %v", currentUser.ID, currentUser.LastMenu, err)
			return c.Send("لطفا ابتدا یک دوره را شروع کنید یا ادامه دهید.", keyboards.MainMenu)
		}
		return handlers.HandleAdvanceWord(c, currentUser.ID, courseID, appServices)
	})

	// --- General Text Handler (for course titles, etc.) ---
	// This comes after specific text button handlers.
	b.Handle(telebot.OnText, func(c telebot.Context) error {
		return handlers.HandleTextMessage(c, appServices)
	})

	// --- Callback Query Handler (Unified) ---
	b.Handle(telebot.OnCallback, func(c telebot.Context) error {
		callback := c.Callback()
		if callback == nil {
			log.Println("[Router] Received OnCallback event but c.Callback() is nil")
			return nil // Or c.Respond() with an error
		}

		data := strings.TrimSpace(callback.Data)
		log.Printf("[Router OnCallback] Received callback with data: %s from UserID: %d", data, c.Sender().ID)

		switch {
		case strings.HasPrefix(data, keyboards.QuizAnswerCallbackPrefix):
			return handlers.HandleQuizAnswerCallback(c, appServices)
		case strings.HasPrefix(data, keyboards.CourseDetailsCallbackPrefix):
			return handlers.HandleCourseSelectionCallback(c, appServices)
		default:
			log.Printf("[Router OnCallback] Unhandled callback data: %s", data)
			return c.Respond(&telebot.CallbackResponse{
				Text:      "این دکمه دیگر کار نمی‌کند.", // This button no longer works.
				ShowAlert: false,
			})
		}
	})
}
