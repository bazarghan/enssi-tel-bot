package message

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/dto"
	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/formatters"
	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/keyboards"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/course"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/user"
	"github.com/2000ostd/enssi-tel-bot/pkg/tgmarkdown"

	startCmd "github.com/2000ostd/enssi-tel-bot/internal/usecase/commands/course"
	courseQueries "github.com/2000ostd/enssi-tel-bot/internal/usecase/queries/course"
	"gopkg.in/telebot.v4"
)

const (
	BtnStartLearning       = "شروع یادگیری"
	StateMain              = "main"
	StateCourseList        = "course_list"
	StateCourseDetailsBase = "course_details"
)

// Handler holds dependencies for message handlers.
type Handler struct {
	listCourses courseQueries.ListCoursesHandler
	getOverview courseQueries.GetOverviewHandler

	startSession startCmd.StartSessionHandler
	userRepo     user.Repository
	courseRepo   course.Repository
}

// NewHandler creates a new message handler.
func NewHandler(
	listCourses courseQueries.ListCoursesHandler,
	getOverview courseQueries.GetOverviewHandler,

	startSession startCmd.StartSessionHandler,
	userRepo user.Repository,
	courseRepo course.Repository,
) *Handler {

	return &Handler{
		listCourses:  listCourses,
		getOverview:  getOverview,
		startSession: startSession,
		userRepo:     userRepo,
		courseRepo:   courseRepo,
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
	if baseActionText == keyboards.StartCourseButtonText || baseActionText == keyboards.ContinueCourseButtonText {
		cmd := startCmd.StartSessionCommand{UserID: u.ID, CourseID: uint(courseID)}
		res, err := h.startSession.Handle(context.Background(), cmd)
		if err != nil {
			log.Printf("[handleCourseAction] Error starting session for UserID %d, CourseID %d: %v", u.ID, courseID, err)
			return c.Send("مشکلی در شروع دوره پیش آمد.")
		}

		// Handle different results from the use case
		switch res.NextStep {
		case startCmd.ShowWord:
			// TODO: Update user state to be 'in_course:ID'
			return c.Send(formatters.FormatWordForDisplay(res.Word), keyboards.InCourseNavigationKeyboard())
		case startCmd.ShowQuiz:
			// This will be implemented in a future slice
			return c.Send("Quiz time! (Not implemented yet)")
		case startCmd.CourseEnded:
			// TODO: Update user state back to 'main' or 'course_list'
			return c.Send(tgmarkdown.Escape(res.MessageToUser), keyboards.BackToCourseListKeyboard())
		}
	}
	return nil
}
