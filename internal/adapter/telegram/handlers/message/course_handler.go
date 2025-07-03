package message

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/dto"
	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/formatters"
	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/keyboards"
	sc "github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/stateconstants"
	ui "github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/uiconstants"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/course"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/quiz"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/user"
	"github.com/2000ostd/enssi-tel-bot/internal/platform/observability/logger"
	"github.com/2000ostd/enssi-tel-bot/pkg/tgmarkdown"

	advanceCmd "github.com/2000ostd/enssi-tel-bot/internal/usecase/commands/course"
	startCmd "github.com/2000ostd/enssi-tel-bot/internal/usecase/commands/course"
	cacheCmd "github.com/2000ostd/enssi-tel-bot/internal/usecase/commands/word"
	courseQueries "github.com/2000ostd/enssi-tel-bot/internal/usecase/queries/course"
	"gopkg.in/telebot.v4"
)

// CourseHandler holds dependencies for course-related message handlers.
type CourseHandler struct {
	logger       logger.Logger
	getOverview  courseQueries.GetOverviewHandler
	advanceWord  advanceCmd.AdvanceWordHandler
	startSession startCmd.StartSessionHandler
	userRepo     user.Repository
	quizRepo     quiz.Repository
	courseRepo   course.Repository
	cacheMedia   cacheCmd.CacheMediaHandler
}

// NewCourseHandler creates a new course handler.
func NewCourseHandler(
	appLogger logger.Logger,
	getOverview courseQueries.GetOverviewHandler,
	advanceWord advanceCmd.AdvanceWordHandler,
	startSession startCmd.StartSessionHandler,
	userRepo user.Repository,
	courseRepo course.Repository,
	quizRepo quiz.Repository,
	cacheMedia cacheCmd.CacheMediaHandler,
) *CourseHandler {
	return &CourseHandler{
		logger:       appLogger,
		getOverview:  getOverview,
		advanceWord:  advanceWord,
		startSession: startSession,
		userRepo:     userRepo,
		courseRepo:   courseRepo,
		quizRepo:     quizRepo,
		cacheMedia:   cacheMedia,
	}
}

// Handle routes incoming course-related messages based on user state.
func (h *CourseHandler) Handle(c telebot.Context, u user.User, userInput string) error {
	stateBase, stateID, _ := strings.Cut(u.LastMenu, ":")

	switch {
	case stateBase == sc.StateCourseList:
		return h.handleCourseSelection(c, u, userInput)

	case stateBase == sc.StateCourseDetails && stateID != "":
		return h.handleCourseAction(c, u, userInput, stateID)

	case stateBase == sc.StateInCourse && stateID != "":
		if userInput == ui.BtnNextWordText {
			return h.handleNextWord(c, u, stateID)
		}
	}
	return nil // Should not happen if routed correctly
}

func (h *CourseHandler) handleCourseSelection(c telebot.Context, u user.User, selectionText string) error {

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

func (h *CourseHandler) displayCourseOverview(c telebot.Context, u user.User, courseID uint) error {
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

func (h *CourseHandler) handleCourseAction(c telebot.Context, u user.User, actionText, courseIDStr string) error {
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

func (h *CourseHandler) handleNextWord(c telebot.Context, u user.User, courseIDStr string) error {
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
func (h *CourseHandler) sendLearningContext(c telebot.Context, res startCmd.StartSessionResult, courseID, userID uint) error {
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

func (h *CourseHandler) sendWordImageByUrlAndCache(c telebot.Context, wordData startCmd.WordDisplayData) {
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

func (h *CourseHandler) sendWordAudioByUrlAndCache(c telebot.Context, pronData startCmd.PronunciationDisplayData) {
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
