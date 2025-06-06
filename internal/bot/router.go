package bot

import (
	"log"
	"strings"

	"github.com/2000ostd/enssi-tel-bot/internal/bot/formatters"
	"github.com/2000ostd/enssi-tel-bot/internal/bot/handlers"
	"github.com/2000ostd/enssi-tel-bot/internal/bot/keyboards" // For button text constants
	"github.com/2000ostd/enssi-tel-bot/internal/models"        // For dbUser
	"github.com/2000ostd/enssi-tel-bot/internal/services"
	"gopkg.in/telebot.v4"
)

// RegisterRoutes sets up all command, text, and callback handlers for the bot.
func RegisterRoutes(b *telebot.Bot, appServices *services.AppServices) {
	// Middleware to fetch/create user and make it available in context
	b.Use(UserActivityMiddleware(appServices))
	// General error handler middleware (optional, if you want centralized error logging/reply)
	// b.Use(ErrorHandlerMiddleware) // Assuming ErrorHandlerMiddleware is defined

	// --- Command Handlers ---
	// The CheckAndInitiateReview is inside these command handlers.
	b.Handle("/start", func(c telebot.Context) error {
		return handlers.HandleStartCommand(c, appServices)
	})
	// Add other commands like /myprofile, /settings if they exist
	b.Handle("/profile", func(c telebot.Context) error { // Assuming /profile for MyProfile
		return handlers.HandleMyProfileCommand(c, appServices)
	})

	// ---> ADD ADMIN PANEL BUTTON HANDLER <---
	b.Handle(keyboards.BtnAdminPanel.Text, func(c telebot.Context) error {
		dbUser, ok := c.Get(string(DBUserKey)).(*models.User)
		if !ok || dbUser == nil {
			log.Printf("[Router] BtnAdminPanel: User not found in context.")
			return c.Send("خطا: اطلاعات کاربری یافت نشد.")
		}

		if !dbUser.IsAdmin {
			log.Printf("[Router] BtnAdminPanel: Non-admin UserID %d attempted to access admin panel.", dbUser.ID)
			return c.Send("شما اجازه دسترسی به این بخش را ندارید.")
		}

		// Update user's state to indicate they are in the admin panel
		err := appServices.User().UpdateUserLastMenu(dbUser.ID, handlers.StateInAdminPanel)
		if err != nil {
			log.Printf("[Router] BtnAdminPanel: Error updating last menu for UserID %d: %v", dbUser.ID, err)
			return handlers.SendServiceError(c, "entering admin panel", err)
		}

		adminPanelMessage := "به پنل ادمین خوش آمدید. 👨‍💻\n" +
			"اکنون می‌توانید کوئری‌های SQL خام را مستقیماً ارسال کنید.\n" +
			"هشدار: اجرای کوئری‌های نادرست می‌تواند به داده‌ها آسیب برساند.\n" +
			"برای خروج از پنل ادمین، از دکمه 'بازگشت به منوی اصلی' استفاده کنید."

		// Send message with only "Return to Main Menu" keyboard
		return c.Send(formatters.EscapeMarkdownV2(adminPanelMessage), keyboards.BackToMainMenuKeyboard(), telebot.ModeMarkdownV2)
	})
	// ---> END OF ADMIN PANEL BUTTON HANDLER <---

	// --- Text Handlers for specific Reply Keyboard Buttons ---
	// These are matched before the general telebot.OnText handler.
	// It's crucial to add the CheckAndInitiateReview logic here if these lead to content.

	b.Handle(keyboards.BtnStartLearning.Text, func(c telebot.Context) error {
		dbUser, ok := c.Get(string(DBUserKey)).(*models.User) // Get user from context
		if !ok || dbUser == nil {
			// Fallback if not in context (should not happen with middleware)
			var errGetUser error
			dbUser, errGetUser = appServices.User().GetOrCreateUserByTelegramID(c.Sender().ID, c.Sender().Username, c.Sender().FirstName, c.Sender().LastName)
			if errGetUser != nil {
				log.Printf("[Router] BtnStartLearning: Error getting user: %v", errGetUser)
				return c.Send("خطا در پردازش درخواست شما.")
			}
		}

		reviewHandled, reviewErr := handlers.CheckAndInitiateReview(c, appServices, dbUser)
		if reviewErr != nil {
			return handlers.SendServiceError(c, "checking review (start learning)", reviewErr)
		}
		if reviewHandled {
			return nil
		}
		return handlers.HandleStartLearningJourney(c, dbUser.ID, appServices)
	})

	b.Handle(keyboards.BtnMyProfile.Text, func(c telebot.Context) error {
		// MyProfile command already handles review check.
		return handlers.HandleMyProfileCommand(c, appServices)
	})

	b.Handle(keyboards.BtnReturnToMainMenu.Text, func(c telebot.Context) error {
		// ReturnToMainMenu should not be blocked by review.
		dbUser, _ := c.Get(string(DBUserKey)).(*models.User) // Error handling for user fetch can be added
		if dbUser == nil {
			var errGetUser error
			dbUser, errGetUser = appServices.User().GetOrCreateUserByTelegramID(c.Sender().ID, c.Sender().Username, c.Sender().FirstName, c.Sender().LastName)
			if errGetUser != nil {
				log.Printf("[Router] BtnReturnToMainMenu: Error getting user: %v", errGetUser)
				return c.Send("خطا.")
			}
		}
		return handlers.HandleReturnToMainMenu(c, dbUser.ID, appServices)
	})

	b.Handle(keyboards.NextWordButtonText, func(c telebot.Context) error {
		dbUser, ok := c.Get(string(DBUserKey)).(*models.User)
		if !ok || dbUser == nil {
			var errGetUser error
			dbUser, errGetUser = appServices.User().GetOrCreateUserByTelegramID(c.Sender().ID, c.Sender().Username, c.Sender().FirstName, c.Sender().LastName)
			if errGetUser != nil {
				log.Printf("[Router] NextWordButton: Error getting user: %v", errGetUser)
				return c.Send("خطا.")
			}
		}

		reviewHandled, reviewErr := handlers.CheckAndInitiateReview(c, appServices, dbUser)
		if reviewErr != nil {
			return handlers.SendServiceError(c, "checking review (next word)", reviewErr)
		}
		if reviewHandled {
			return nil
		}

		// Course ID needs to be derived from user's current state/LastMenu
		courseID, err := handlers.ParseIDFromState(dbUser.LastMenu, handlers.StateInCoursePrefix)
		if err != nil {
			log.Printf("[Router] NextWordButton: UserID %d, LastMenu '%s' not in 'in_course:ID' state or invalid courseID. Error: %v", dbUser.ID, dbUser.LastMenu, err)
			// Attempt to guide user if context is lost
			appServices.User().UpdateUserLastMenu(dbUser.ID, handlers.StateMain)
			return c.Send("به نظر میرسد از دوره خارج شده اید. لطفا از منوی اصلی دوباره شروع کنید.", keyboards.MainMenu)
		}
		return handlers.HandleAdvanceWord(c, dbUser.ID, courseID, appServices)
	})

	// --- General Text Handler (for course titles from reply keyboards, etc.) ---
	// This comes after specific text button handlers.
	// The CheckAndInitiateReview is inside HandleTextMessage itself.
	b.Handle(telebot.OnText, func(c telebot.Context) error {
		return handlers.HandleTextMessage(c, appServices)
	})

	// --- Callback Query Handler (Unified) ---
	// The CheckAndInitiateReview is inside the specific callback handlers like HandleCourseSelectionCallback.
	// HandleQuizAnswerCallback does not need it as user is already in a quiz.
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
		case strings.HasPrefix(data, keyboards.CourseDetailsCallbackPrefix): // For inline course selection
			return handlers.HandleCourseSelectionCallback(c, appServices)
		// Add other callback prefixes here if any
		// case strings.HasPrefix(data, handlers.SomeOtherCallbackPrefix):
		// return handlers.HandleSomeOtherCallback(c, appServices)
		default:
			log.Printf("[Router OnCallback] Unhandled callback data: %s", data)
			return c.Respond(&telebot.CallbackResponse{
				Text:      "این دکمه دیگر کار نمی‌کند یا عملیات نامعتبر است.",
				ShowAlert: false, // True if you want a more prominent alert
			})
		}
	})

	log.Println("Bot routes registered.")
}
