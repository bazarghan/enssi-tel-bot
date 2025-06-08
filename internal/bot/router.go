package bot

import (
	"log"
	"strings"

	"github.com/2000ostd/enssi-tel-bot/internal/bot/handlers"
	"github.com/2000ostd/enssi-tel-bot/internal/bot/keyboards"
	"github.com/2000ostd/enssi-tel-bot/internal/models"
	"github.com/2000ostd/enssi-tel-bot/internal/services"
	"gopkg.in/telebot.v4"
)

// RegisterRoutes sets up all command, text, and callback handlers for the bot.
func RegisterRoutes(
	b *telebot.Bot,
	appServices *services.AppServices,
	lockManager *UserLockManager,
) {
	// Middleware is applied sequentially to all incoming requests.
	// 1. UserActivityMiddleware: Identifies or creates the user.
	// 2. UserLockMiddleware: Prevents concurrent requests from the same user.
	b.Use(UserActivityMiddleware(appServices))
	b.Use(UserLockMiddleware(lockManager))

	// --- GLOBAL COMMANDS ---
	// /start should always work to reset the user's state and bring them to the main menu.
	b.Handle("/start", func(c telebot.Context) error {
		return handlers.HandleStartCommand(c, appServices)
	})

	// --- PRIMARY TEXT HANDLER (STATE MACHINE) ---
	b.Handle(telebot.OnText, func(c telebot.Context) error {
		// This handler needs the user from context. Let's pass it.
		// It's better to fetch it here once rather than in every handler it calls.
		dbUser, ok := c.Get(string(DBUserKey)).(*models.User) // Use the key from the same package
		if !ok || dbUser == nil {
			log.Printf("[Router OnText] Critical: User not found in context.")
			return c.Send("خطا در پردازش اطلاعات کاربر.")
		}
		return handlers.HandleStateBasedText(c, dbUser, appServices) // Pass the user object
	})

	// --- UNIFIED CALLBACK HANDLER ---
	// Callbacks are handled separately as they are not text messages.
	// Their logic is self-contained based on the callback data prefix.
	b.Handle(telebot.OnCallback, func(c telebot.Context) error {
		callback := c.Callback()
		if callback == nil {
			log.Println("[Router] Received OnCallback event but c.Callback() is nil")
			return c.Respond(&telebot.CallbackResponse{Text: "خطای دکمه.", ShowAlert: true})
		}

		data := strings.TrimSpace(callback.Data)
		log.Printf("[Router OnCallback] Received callback with data: '%s' from UserID: %d", data, c.Sender().ID)

		switch {
		case strings.HasPrefix(data, keyboards.QuizAnswerCallbackPrefix):
			return handlers.HandleQuizAnswerCallback(c, appServices)
		case strings.HasPrefix(data, keyboards.CourseDetailsCallbackPrefix):
			return handlers.HandleCourseSelectionCallback(c, appServices)
		default:
			log.Printf("[Router OnCallback] Unhandled callback data: %s", data)
			return c.Respond(&telebot.CallbackResponse{
				Text:      "این دکمه دیگر کار نمی‌کند.",
				ShowAlert: false,
			})
		}
	})

}
