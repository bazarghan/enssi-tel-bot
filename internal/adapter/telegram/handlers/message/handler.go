package message

import (
	"context"
	"errors"
	"fmt"
	"github.com/bits-and-blooms/bitset"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/dto"
	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/keyboards"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/course"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/quiz"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/user"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/word"
	"github.com/2000ostd/enssi-tel-bot/pkg/tgmarkdown"

	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/formatters"
	"github.com/2000ostd/enssi-tel-bot/internal/platform/observability/logger"

	"github.com/2000ostd/enssi-tel-bot/internal/domain/achievement"

	sc "github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/stateconstants"
	ui "github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/uiconstants"
	adminCmd "github.com/2000ostd/enssi-tel-bot/internal/usecase/commands/admin"
	advanceCmd "github.com/2000ostd/enssi-tel-bot/internal/usecase/commands/course"
	startCmd "github.com/2000ostd/enssi-tel-bot/internal/usecase/commands/course"
	quizCmd "github.com/2000ostd/enssi-tel-bot/internal/usecase/commands/quiz"
	cacheCmd "github.com/2000ostd/enssi-tel-bot/internal/usecase/commands/word"
	achQueries "github.com/2000ostd/enssi-tel-bot/internal/usecase/queries/achievement"
	adminQueries "github.com/2000ostd/enssi-tel-bot/internal/usecase/queries/admin"
	courseQueries "github.com/2000ostd/enssi-tel-bot/internal/usecase/queries/course"
	reviewQueries "github.com/2000ostd/enssi-tel-bot/internal/usecase/queries/review"
	getProfileQry "github.com/2000ostd/enssi-tel-bot/internal/usecase/queries/user"

	"gopkg.in/telebot.v4"
)

// Handler holds dependencies for message handlers.
type Handler struct {
	logger             logger.Logger
	listCourses        courseQueries.ListCoursesHandler
	getOverview        courseQueries.GetOverviewHandler
	getProfile         getProfileQry.GetProfileHandler
	getStats           adminQueries.GetStatsHandler
	Broadcast          adminCmd.BroadcastHandler
	getAllAchievements achQueries.GetAllHandler
	advanceWord        advanceCmd.AdvanceWordHandler
	startSession       startCmd.StartSessionHandler
	userRepo           user.Repository
	quizRepo           quiz.Repository
	courseRepo         course.Repository
	wordRepo           word.Repository
	createReviewQuiz   quizCmd.CreateReviewQuizHandler
	cacheMedia         cacheCmd.CacheMediaHandler
	achRepo            achievement.Repository
	imgSvc             achievement.ImageGenerator
	hasPendingReview   reviewQueries.Handler
}

// NewHandler creates a new message handler.
func NewHandler(
	appLogger logger.Logger,
	listCourses courseQueries.ListCoursesHandler,
	getOverview courseQueries.GetOverviewHandler,
	getProfile getProfileQry.GetProfileHandler,
	getStats adminQueries.GetStatsHandler,
	getAllAchievements achQueries.GetAllHandler,
	achRepo achievement.Repository,
	imgSvc achievement.ImageGenerator,
	advanceWord advanceCmd.AdvanceWordHandler,
	startSession startCmd.StartSessionHandler,
	userRepo user.Repository,
	courseRepo course.Repository,
	quizRepo quiz.Repository,
	wordRepo word.Repository,
	createReviewQuiz quizCmd.CreateReviewQuizHandler,
	cacheMedia cacheCmd.CacheMediaHandler,
	hasPendingReview reviewQueries.Handler,

) *Handler {

	return &Handler{
		logger:             appLogger,
		listCourses:        listCourses,
		getOverview:        getOverview,
		getProfile:         getProfile,
		getStats:           getStats,
		getAllAchievements: getAllAchievements,
		startSession:       startSession,
		advanceWord:        advanceWord,
		userRepo:           userRepo,
		courseRepo:         courseRepo,
		achRepo:            achRepo,
		quizRepo:           quizRepo,
		wordRepo:           wordRepo,
		createReviewQuiz:   createReviewQuiz,
		imgSvc:             imgSvc,
		cacheMedia:         cacheMedia, // Dependency assigned here
		hasPendingReview:   hasPendingReview,
	}
}

// Handle routes incoming text messages based on content and user state.
func (h *Handler) Handle(c telebot.Context) error {

	ctxUser, ok := c.Get("dbUser").(user.User)

	if !ok {
		h.logger.Error("Error registering or finding user", "error", ok, "telegram_id", c.Sender().ID)
		return c.Send("خطا در پردازش اطلاعات کاربر.")
	}

	userInput := strings.TrimSpace(c.Text())

	stateBase, stateID, _ := strings.Cut(ctxUser.LastMenu, ":")

	// Global commands
	if userInput == ui.BtnReturnToMainMenuText {
		return h.handleReturnToMainMenu(c, ctxUser)
	}

	// State-based routing
	switch {
	case ctxUser.LastMenu == sc.StateMain:

		if userInput == ui.BtnStartLearningText {
			return h.handleStartLearning(c, ctxUser)
		}

		if userInput == ui.BtnMyProfileText {
			return h.handleGetProfile(c, ctxUser)
		}

		if userInput == ui.BtnDailyReviewText {
			return h.handleDailyReview(c, ctxUser)
		}

		if userInput == ui.BtnAdminPanelText {
			return h.handleAdminPanel(c, ctxUser)
		}

	case ctxUser.LastMenu == sc.StateCourseList:

		return h.handleCourseSelection(c, ctxUser, userInput)

	case ctxUser.LastMenu == sc.StateAchievementList:

		if userInput == ui.BtnReturnToProfileText {
			return h.handleReturnToProfile(c, ctxUser)
		} else {
			return h.handleAchievementSelection(c, ctxUser, userInput)
		}

	case ctxUser.LastMenu == sc.StateAdminPanel:

		if userInput == ui.BtnAdminStatsText {
			return h.handleGetStats(c)
		}
		if userInput == ui.BtnAdminBroadcastText {
			return h.handleInitiateAdminBroadcast(c, ctxUser)
		}

	case ctxUser.LastMenu == sc.StateAdminBroadcast:

		return h.handleAdminBroadcast(c, ctxUser, userInput)

	case stateBase == sc.StateCourseDetails && stateID != "":

		return h.handleCourseAction(c, ctxUser, userInput, stateID)

	case stateBase == sc.StateInCourse && stateID != "":

		if userInput == ui.BtnNextWordText {
			return h.handleNextWord(c, ctxUser, stateID)
		}

	case stateBase == sc.StateProfileMenu:

		if userInput == ui.BtnViewAchievementsText {
			return h.handleViewMyAchievements(c, ctxUser)
		}

	}

	h.logger.Info("Unhandled text message", "user_id", ctxUser.ID, "state", ctxUser.LastMenu, "input", userInput)
	return nil
}

func (h *Handler) handleStartLearning(c telebot.Context, u user.User) error {

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

func (h *Handler) handleGetProfile(c telebot.Context, u user.User) error {

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

func (h *Handler) handleCourseSelection(c telebot.Context, u user.User, selectionText string) error {
	// The text might have progress details, so find the base title.
	baseTitle := strings.Split(selectionText, " (")[0]

	domainCourse, err := h.courseRepo.FindByPersianTitle(context.Background(), baseTitle)
	if err != nil {
		if errors.Is(err, course.ErrNotFound) {
			h.logger.Warn("User selected a course not found in DB", "user_id", u.ID, "title", baseTitle)
			return nil // Ignore invalid input
		}
		h.logger.Error("DB error finding course by title", "title", baseTitle, "user_id", u.ID, "error", err)
		return c.Send("مشکلی در یافتن دوره پیش آمد.")
	}

	return h.displayCourseOverview(c, u, domainCourse.ID)
}

func (h *Handler) displayCourseOverview(c telebot.Context, u user.User, courseID uint) error {
	query := courseQueries.GetOverviewQuery{UserID: u.ID, CourseID: courseID}
	overview, err := h.getOverview.Handle(context.Background(), query)
	if err != nil {
		h.logger.Error("Error getting course overview", "user_id", u.ID, "course_id", courseID, "error", err)
		return c.Send("مشکلی در نمایش اطلاعات دوره پیش آمد.")
	}

	dto := dto.CourseOverview{
		ID:                     overview.ID,
		Title:                  overview.Title,
		PersianTitle:           overview.PersianTitle,
		PersianFullDescription: overview.PersianFullDescription,
		TotalWords:             overview.TotalWords,
		ProgressPercentage:     overview.ProgressPercentage,
		IsCompleted:            overview.IsCompleted,
		IsStarted:              overview.IsStarted,
	}

	msg := formatters.FormatCourseOverview(dto)
	kb := keyboards.CourseDetailsKeyboard(dto)

	newState := fmt.Sprintf("%s:%d", sc.StateCourseDetails, courseID)
	if err := h.userRepo.UpdateLastMenu(context.Background(), u.ID, newState); err != nil {
		h.logger.Error("Failed to update user state", "user_id", u.ID, "error", err)
	}

	return c.Send(msg, kb, telebot.ModeMarkdownV2)
}

func (h *Handler) handleReturnToMainMenu(c telebot.Context, u user.User) error {

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

func (h *Handler) handleCourseAction(c telebot.Context, u user.User, actionText, courseIDStr string) error {
	courseID, _ := strconv.ParseUint(courseIDStr, 10, 32)
	if courseID == 0 {
		return nil // Invalid state
	}

	baseActionText := strings.Split(actionText, " (")[0]
	if baseActionText != ui.BtnStartCourseText && baseActionText != ui.BtnContinueCourseText {
		return nil // Not a start/continue action
	}

	cmd := startCmd.StartSessionCommand{UserID: u.ID, CourseID: uint(courseID)}
	res, err := h.startSession.Handle(context.Background(), cmd)
	if err != nil {
		h.logger.Error("Error starting session", "user_id", u.ID, "course_id", courseID, "error", err)
		return c.Send("There was a problem starting the course.")
	}
	return h.sendLearningContext(c, res, uint(courseID), u.ID)
}

func (h *Handler) handleNextWord(c telebot.Context, u user.User, courseIDStr string) error {
	courseID, _ := strconv.ParseUint(courseIDStr, 10, 32)
	if courseID == 0 {
		return nil // Invalid state
	}

	cmd := advanceCmd.AdvanceWordCommand{UserID: u.ID, CourseID: uint(courseID)}
	res, err := h.advanceWord.Handle(context.Background(), cmd)
	if err != nil {
		h.logger.Error("Error advancing word", "user_id", u.ID, "course_id", courseID, "error", err)
		return c.Send("There was a problem getting the next word.")
	}
	return h.sendLearningContext(c, res, uint(courseID), u.ID)
}

// sendLearningContext processes the result from a course use case and sends the appropriate message.
func (h *Handler) sendLearningContext(c telebot.Context, res startCmd.StartSessionResult, courseID, userID uint) error {
	var newState string
	var kb *telebot.ReplyMarkup = &telebot.ReplyMarkup{RemoveKeyboard: true}
	var msg string
	var sendErr error

	switch res.NextStep {
	case startCmd.ShowWord:
		wordData := res.Word
		newState = fmt.Sprintf("%s:%d", sc.StateInCourse, courseID)

		// Format word text using the domain entity within the DTO
		// This assumes the DTO can be easily converted or contains the necessary domain object.

		// Unpack the DTO and pass the pure domain entity to the formatter.
		msg := formatters.FormatWordForDisplay(wordData.DomainWord)
		kb := keyboards.InCourseNavigationKeyboard()

		if err := c.Send(msg, kb, telebot.ModeMarkdownV2); err != nil {
			h.logger.Error("Error sending word text message", "error", err)
			return err
		}

		// --- Image Sending Logic ---
		if wordData.TelegramImageID != "" {
			photo := &telebot.Photo{File: telebot.File{FileID: wordData.TelegramImageID}}
			if _, err := c.Bot().Send(c.Chat(), photo); err != nil {
				h.logger.Warn("Failed to send image by FileID, falling back to URL", "file_id", wordData.TelegramImageID, "error", err)
				// Invalidate bad FileID here if needed
				h.sendWordImageByUrlAndCache(c, wordData)
			}
		} else {
			h.sendWordImageByUrlAndCache(c, wordData)
		}

		// --- Audio Sending Logic ---
		for _, pron := range wordData.Pronunciations {
			if pron.TelegramVoiceID != "" {
				voice := &telebot.Voice{File: telebot.File{FileID: pron.TelegramVoiceID}, Caption: pron.Region}
				if _, err := c.Bot().Send(c.Chat(), voice); err != nil {
					h.logger.Warn("Failed to send voice by FileID, falling back to URL", "file_id", pron.TelegramVoiceID, "error", err)
					h.sendWordAudioByUrlAndCache(c, pron)
				}
			}
		}

	case startCmd.CourseEnded:

		newState = sc.StateCourseList
		msg = tgmarkdown.Escape(res.MessageToUser)
		kb = keyboards.BackToCourseListKeyboard()
		sendErr = c.Send(msg, kb, telebot.ModeMarkdownV2)

	case startCmd.ShowQuiz:

		if res.IsNewQuiz {
			introMsg := "شما این بخش را به پایان رساندید! 🎉 حالا بیایید ببینیم چقدر یاد گرفته اید. یک آزمون کوتاه در پیش است."
			err := c.Send(tgmarkdown.Escape(introMsg), keyboards.QuizKeyboard(), telebot.ModeMarkdownV2)
			if err != nil {
				h.logger.Error("Something went wrong when sending the keyboard", "error", err)
			}
		}

		attempt := res.QuizAttempt
		question := attempt.Questions[attempt.CurrentQuestionIndex]
		newState = fmt.Sprintf("in_quiz:%d:%d", courseID, attempt.ID) // A more specific state
		msg = formatters.FormatQuizQuestion(question, attempt.CurrentQuestionIndex, len(attempt.Questions))
		kb = keyboards.QuizQuestionOptionsKeyboard(question.Options, attempt.ID)

		sentMsg, err := c.Bot().Send(c.Chat(), msg, kb, telebot.ModeMarkdownV2)
		if err != nil {
			h.logger.Error("Failed to send initial quiz question", "error", err)
			return err
		}
		// Save message ID so the quiz can be edited later by the callback handler
		h.quizRepo.UpdateMessageID(context.Background(), attempt.ID, sentMsg.ID)
		h.userRepo.UpdateLastMenu(context.Background(), userID, newState)
		return nil // Return early as the message is already sent and state is updated

	default:
		// Fallback for unhandled steps
		newState = sc.StateMain
		msg = "An unknown error occurred. Returning to main menu."
		kb = keyboards.NewMainMenu(false, false)
		sendErr = c.Send(msg)
	}

	if sendErr != nil {
		h.logger.Error("Error sending learning context message", "error", sendErr)
	}

	h.userRepo.UpdateLastMenu(context.Background(), userID, newState)
	return sendErr
}

func (h *Handler) handleViewMyAchievements(c telebot.Context, u user.User) error {
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
	h.userRepo.UpdateLastMenu(context.Background(), u.ID, "achievements_list")

	// 5. Send the message with the inline keyboard.
	return c.Send("می توانید با کلیک بر روی هر دستاورد، پیشرفت خود را مشاهده کنید:", kb)
}

func (h *Handler) sendWordImageByUrlAndCache(c telebot.Context, wordData startCmd.WordDisplayData) {
	if wordData.ImageURL == "" {
		return
	}
	photo := &telebot.Photo{File: telebot.FromURL(wordData.ImageURL)}
	sentMsg, err := c.Bot().Send(c.Chat(), photo)
	if err != nil {
		h.logger.Error("Failed to send image by URL", "url", wordData.ImageURL, "error", err)
		return
	}
	if sentMsg.Photo != nil {
		cmd := cacheCmd.CacheImageCommand{
			CourseWordID: wordData.CourseWordID,
			ImageFileID:  sentMsg.Photo.FileID,
		}
		if err := h.cacheMedia.HandleImage(context.Background(), cmd); err != nil {
			h.logger.Error("Failed to cache new image FileID", "error", err)
		}
	}
}

func (h *Handler) sendWordAudioByUrlAndCache(c telebot.Context, pronData startCmd.PronunciationDisplayData) {
	if pronData.AudioURL == "" {
		return
	}
	voice := &telebot.Voice{File: telebot.FromURL(pronData.AudioURL), Caption: pronData.Region}
	sentMsg, err := c.Bot().Send(c.Chat(), voice)
	if err != nil {
		h.logger.Error("Failed to send voice by URL", "url", pronData.AudioURL, "error", err)
		return
	}
	if sentMsg.Voice != nil {
		cmd := cacheCmd.CacheVoiceCommand{
			PronunciationID: pronData.ID,
			VoiceFileID:     sentMsg.Voice.FileID,
		}
		if err := h.cacheMedia.HandleVoice(context.Background(), cmd); err != nil {
			h.logger.Error("Failed to cache new voice FileID", "error", err)
		}
	}
}

func (h *Handler) handleDailyReview(c telebot.Context, u user.User) error {

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

	rand.Shuffle(len(question.Options), func(i, j int) { question.Options[i], question.Options[j] = question.Options[j], question.Options[i] })
	msg := formatters.FormatQuizQuestion(question, attempt.CurrentQuestionIndex, len(attempt.Questions))
	kb := keyboards.QuizQuestionOptionsKeyboard(question.Options, attempt.ID)
	sentMsg, err := c.Bot().Send(c.Chat(), msg, kb, telebot.ModeMarkdownV2)
	if err == nil {
		h.quizRepo.UpdateMessageID(context.Background(), attempt.ID, sentMsg.ID)
	}
	return err
}

func (h *Handler) handleReturnToProfile(c telebot.Context, u user.User) error {
	// This is the same logic from your /myprofile command
	query := getProfileQry.GetProfileQuery{UserID: u.ID}
	_, err := h.getProfile.Handle(context.Background(), query)
	if err != nil {
		return c.Send("Could not get profile.")
	}
	profileDTO := dto.UserProfile{ /* ... map fields ... */ }
	formattedProfile := formatters.FormatUserProfile(profileDTO)
	h.userRepo.UpdateLastMenu(context.Background(), u.ID, sc.StateProfileMenu)
	return c.Send(formattedProfile, keyboards.ProfileMenuKeyboard(), telebot.ModeMarkdownV2)
}

func (h *Handler) handleAdminPanel(c telebot.Context, u user.User) error {

	if !u.IsAdmin {
		return nil // Ignore if a non-admin somehow sends this text
	}
	// Set the user's state to the admin panel
	h.userRepo.UpdateLastMenu(context.Background(), u.ID, "admin_panel")
	// Send the new admin keyboard
	return c.Send("به پنل ادمین خوش آمدید.", keyboards.AdminPanelKeyboard())

}

func (h *Handler) handleGetStats(c telebot.Context) error {
	stats, err := h.getStats.Handle(context.Background())
	if err != nil {
		h.logger.Error("Failed to get admin stats", "error", err)
		return c.Send("خطا در دریافت آمار.")
	}

	// Format the message and send it back to the admin
	statsMsg := fmt.Sprintf("📊 *آمار ربات*\n\nتعداد کل کاربران: *%d*", stats.TotalUsers)
	return c.Send(statsMsg, telebot.ModeMarkdownV2)

}

func (h *Handler) handleInitiateAdminBroadcast(c telebot.Context, u user.User) error {

	h.userRepo.UpdateLastMenu(context.Background(), u.ID, sc.StateAdminBroadcast)
	return c.Send("لطفا پیامی که میخواهید برای همه کاربران ارسال شود را وارد کنید:")

}

func (h *Handler) handleAdminBroadcast(c telebot.Context, u user.User, userInput string) error {

	// Any text received in this state is the message to be broadcast
	broadcastCmd := adminCmd.BroadcastCommand{Message: userInput} // Assuming adminCmd alias

	recipients, err := h.Broadcast.Handle(context.Background(), broadcastCmd) // Assuming dependency is `broadcast`
	if err != nil {
		h.logger.Error("Broadcast failed", "error", err)
		return c.Send("ارسال پیام همگانی با خطا مواجه شد.")
	}

	// Send confirmation to the admin and return them to the admin panel
	confirmationMsg := fmt.Sprintf("✅ پیام شما برای %d کاربر ارسال شد.", recipients)
	c.Send(confirmationMsg)

	h.userRepo.UpdateLastMenu(context.Background(), u.ID, "admin_panel")
	return c.Send("به پنل ادمین بازگشتید.", keyboards.AdminPanelKeyboard())

}

func (h *Handler) handleAchievementSelection(c telebot.Context, u user.User, userInput string) error {
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
