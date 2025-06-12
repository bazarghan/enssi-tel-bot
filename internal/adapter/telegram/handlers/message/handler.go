package message

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/dto"
	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/keyboards"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/course"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/quiz"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/user"
	"github.com/2000ostd/enssi-tel-bot/pkg/tgmarkdown"

	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/formatters"

	advanceCmd "github.com/2000ostd/enssi-tel-bot/internal/usecase/commands/course"
	startCmd "github.com/2000ostd/enssi-tel-bot/internal/usecase/commands/course"
	cacheCmd "github.com/2000ostd/enssi-tel-bot/internal/usecase/commands/word"
	courseQueries "github.com/2000ostd/enssi-tel-bot/internal/usecase/queries/course"
	"gopkg.in/telebot.v4"
)

const (
	BtnStartLearning       = "شروع یادگیری"
	StateMain              = "main"
	StateCourseList        = "course_list"
	StateCourseDetailsBase = "course_details"
	StateProfileMenu       = "profile_menu"
	StateInCourseBase      = "in_course"
)

// Handler holds dependencies for message handlers.
type Handler struct {
	listCourses courseQueries.ListCoursesHandler
	getOverview courseQueries.GetOverviewHandler

	advanceWord advanceCmd.AdvanceWordHandler

	startSession startCmd.StartSessionHandler
	userRepo     user.Repository
	quizRepo     quiz.Repository
	courseRepo   course.Repository

	cacheMedia cacheCmd.CacheMediaHandler
}

// NewHandler creates a new message handler.
func NewHandler(
	listCourses courseQueries.ListCoursesHandler,
	getOverview courseQueries.GetOverviewHandler,
	advanceWord advanceCmd.AdvanceWordHandler,
	startSession startCmd.StartSessionHandler,
	userRepo user.Repository,
	courseRepo course.Repository,
	quizRepo quiz.Repository,
) *Handler {

	return &Handler{
		listCourses:  listCourses,
		getOverview:  getOverview,
		startSession: startSession,
		advanceWord:  advanceWord,
		userRepo:     userRepo,
		courseRepo:   courseRepo,
		quizRepo:     quizRepo,
	}
}

// Handle routes incoming text messages based on content and user state.
func (h *Handler) Handle(c telebot.Context) error {
	ctxUser, ok := c.Get("dbUser").(user.User)
	if !ok {
		log.Printf("[MessageHandler] Critical: User not found in context for telegram ID %d", c.Sender().ID)
		return c.Send("خطا در پردازش اطلاعات کاربر.")
	}

	userInput := strings.TrimSpace(c.Text())

	stateBase, stateID, _ := strings.Cut(ctxUser.LastMenu, ":")

	// Global commands
	if userInput == keyboards.BtnReturnToMainMenu {
		return h.handleReturnToMainMenu(c, ctxUser)
	}

	// State-based routing
	switch {
	case ctxUser.LastMenu == StateMain:
		if userInput == BtnStartLearning {
			return h.handleStartLearning(c, ctxUser)
		}

	case ctxUser.LastMenu == StateCourseList:
		return h.handleCourseSelection(c, ctxUser, userInput)

	case stateBase == StateCourseDetailsBase && stateID != "":
		return h.handleCourseAction(c, ctxUser, userInput, stateID)

	case stateBase == StateInCourseBase && stateID != "":
		if userInput == keyboards.NextWordButtonText {
			return h.handleNextWord(c, ctxUser, stateID)
		}

	case stateBase == StateProfileMenu:
		if userInput == keyboards.BtnViewAchievements.Text {
			return h.handleViewMyAchievements(c, ctxUser)
		}

	}

	log.Printf("[MessageHandler] Unhandled text from UserID %d in state '%s': '%s'", ctxUser.ID, ctxUser.LastMenu, userInput)
	return nil
}

func (h *Handler) handleStartLearning(c telebot.Context, u user.User) error {
	query := courseQueries.ListCoursesQuery{UserID: u.ID}
	courses, err := h.listCourses.Handle(context.Background(), query)
	if err != nil {
		log.Printf("[handleStartLearning] Error listing courses for UserID %d: %v", u.ID, err)
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

	if err := h.userRepo.UpdateLastMenu(context.Background(), u.ID, StateCourseList); err != nil {
		log.Printf("[handleStartLearning] Failed to update user state for UserID %d: %v", u.ID, err)
	}

	return c.Send(msg, kb)
}

func (h *Handler) handleCourseSelection(c telebot.Context, u user.User, selectionText string) error {
	// The text might have progress details, so find the base title.
	baseTitle := strings.Split(selectionText, " (")[0]

	domainCourse, err := h.courseRepo.FindByPersianTitle(context.Background(), baseTitle)
	if err != nil {
		if errors.Is(err, course.ErrNotFound) {
			log.Printf("[handleCourseSelection] User %d selected a course not found in DB: '%s'", u.ID, baseTitle)
			return nil // Ignore invalid input
		}
		log.Printf("[handleCourseSelection] DB error finding course by title '%s' for UserID %d: %v", baseTitle, u.ID, err)
		return c.Send("مشکلی در یافتن دوره پیش آمد.")
	}

	return h.displayCourseOverview(c, u, domainCourse.ID)
}

func (h *Handler) displayCourseOverview(c telebot.Context, u user.User, courseID uint) error {
	query := courseQueries.GetOverviewQuery{UserID: u.ID, CourseID: courseID}
	overview, err := h.getOverview.Handle(context.Background(), query)
	if err != nil {
		log.Printf("[displayCourseOverview] Error getting overview for UserID %d, CourseID %d: %v", u.ID, courseID, err)
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

	newState := fmt.Sprintf("%s:%d", StateCourseDetailsBase, courseID)
	if err := h.userRepo.UpdateLastMenu(context.Background(), u.ID, newState); err != nil {
		log.Printf("[displayCourseOverview] Failed to update user state for UserID %d: %v", u.ID, err)
	}

	return c.Send(msg, kb)
}

func (h *Handler) handleReturnToMainMenu(c telebot.Context, u user.User) error {
	if err := h.userRepo.UpdateLastMenu(context.Background(), u.ID, StateMain); err != nil {
		log.Printf("[handleReturnToMainMenu] Failed to update user state for UserID %d: %v", u.ID, err)
	}
	return c.Send("به منوی اصلی بازگشتید.", keyboards.NewMainMenu(u.IsAdmin))
}

func (h *Handler) handleCourseAction(c telebot.Context, u user.User, actionText, courseIDStr string) error {
	courseID, _ := strconv.ParseUint(courseIDStr, 10, 32)
	if courseID == 0 {
		return nil // Invalid state
	}

	baseActionText := strings.Split(actionText, " (")[0]
	if baseActionText != keyboards.StartCourseButtonText && baseActionText != keyboards.ContinueCourseButtonText {
		return nil // Not a start/continue action
	}

	cmd := startCmd.StartSessionCommand{UserID: u.ID, CourseID: uint(courseID)}
	res, err := h.startSession.Handle(context.Background(), cmd)
	if err != nil {
		log.Printf("[handleCourseAction] Error starting session for UserID %d, CourseID %d: %v", u.ID, courseID, err)
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
		log.Printf("[handleNextWord] Error advancing word for UserID %d, CourseID %d: %v", u.ID, courseID, err)
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
		newState = fmt.Sprintf("%s:%d", StateInCourseBase, courseID)

		// Format word text using the domain entity within the DTO
		// This assumes the DTO can be easily converted or contains the necessary domain object.

		// Unpack the DTO and pass the pure domain entity to the formatter.
		msg := formatters.FormatWordForDisplay(wordData.DomainWord)
		kb := keyboards.InCourseNavigationKeyboard()

		if err := c.Send(msg, kb, telebot.ModeMarkdownV2); err != nil {
			log.Printf("Error sending word text message: %v", err)
			return err
		}

		// --- Image Sending Logic ---
		if wordData.TelegramImageID != "" {
			photo := &telebot.Photo{File: telebot.File{FileID: wordData.TelegramImageID}}
			if _, err := c.Bot().Send(c.Chat(), photo); err != nil {
				log.Printf("Failed to send image by FileID %s, falling back to URL. Error: %v", wordData.TelegramImageID, err)
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
					log.Printf("Failed to send voice by FileID %s, falling back to URL. Error: %v", pron.TelegramVoiceID, err)
					h.sendWordAudioByUrlAndCache(c, pron)
				}
			} else {
				h.sendWordAudioByUrlAndCache(c, pron)
			}
		}

	case startCmd.CourseEnded:
		newState = StateCourseList
		msg = tgmarkdown.Escape(res.MessageToUser)
		kb = keyboards.BackToCourseListKeyboard()
		sendErr = c.Send(msg, kb, telebot.ModeMarkdownV2)
	case startCmd.ShowQuiz:
		attempt := res.QuizAttempt
		question := attempt.Questions[attempt.CurrentQuestionIndex]
		newState = fmt.Sprintf("in_quiz:%d:%d", courseID, attempt.ID) // A more specific state
		msg = formatters.FormatQuizQuestion(question, attempt.CurrentQuestionIndex, len(attempt.Questions))
		kb = keyboards.QuizQuestionOptionsKeyboard(question.Options, attempt.ID)

		sentMsg, err := c.Bot().Send(c.Chat(), msg, kb, telebot.ModeMarkdownV2)
		if err != nil {
			log.Printf("Failed to send initial quiz question: %v", err)
			return err
		}
		// Save message ID so the quiz can be edited later by the callback handler
		h.quizRepo.UpdateMessageID(context.Background(), attempt.ID, sentMsg.ID)
		h.userRepo.UpdateLastMenu(context.Background(), userID, newState)
		return nil // Return early as the message is already sent and state is updated

	default:
		// Fallback for unhandled steps
		newState = StateMain
		msg = "An unknown error occurred. Returning to main menu."
		kb = keyboards.NewMainMenu(false) // Assuming non-admin for safety
		sendErr = c.Send(msg)
	}

	if sendErr != nil {
		log.Printf("Error sending learning context message: %v", sendErr)
	}

	h.userRepo.UpdateLastMenu(context.Background(), userID, newState)
	return sendErr
}

func (h *Handler) handleViewMyAchievements(c telebot.Context, u user.User) error {
	// This requires a use case to get achievements.
	// For now, we assume a simplified query.
	// In a full implementation, you'd call a GetUserAchievements use case.
	log.Printf("User %d viewing achievements.", u.ID)
	// Placeholder DTOs
	achDTOs := []dto.AchievementView{
		// This would be populated from a use case result
	}
	return c.Send("Here are your achievements:", keyboards.AchievementsListKeyboard(achDTOs))
}

func (h *Handler) sendWordImageByUrlAndCache(c telebot.Context, wordData startCmd.WordDisplayData) {
	if wordData.ImageURL == "" {
		return
	}
	photo := &telebot.Photo{File: telebot.FromURL(wordData.ImageURL)}
	sentMsg, err := c.Bot().Send(c.Chat(), photo)
	if err != nil {
		log.Printf("Failed to send image by URL %s: %v", wordData.ImageURL, err)
		return
	}
	if sentMsg.Photo != nil {
		cmd := cacheCmd.CacheImageCommand{
			CourseWordID: wordData.CourseWordID,
			ImageFileID:  sentMsg.Photo.FileID,
		}
		if err := h.cacheMedia.HandleImage(context.Background(), cmd); err != nil {
			log.Printf("Failed to cache new image FileID: %v", err)
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
		log.Printf("Failed to send voice by URL %s: %v", pronData.AudioURL, err)
		return
	}
	if sentMsg.Voice != nil {
		cmd := cacheCmd.CacheVoiceCommand{
			PronunciationID: pronData.ID,
			VoiceFileID:     sentMsg.Voice.FileID,
		}
		if err := h.cacheMedia.HandleVoice(context.Background(), cmd); err != nil {
			log.Printf("Failed to cache new voice FileID: %v", err)
		}
	}
}
