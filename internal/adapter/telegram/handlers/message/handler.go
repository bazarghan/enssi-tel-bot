package message

import (
	"context"
	"errors"
	"log"
	"strings"

	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/dto"
	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/formatters"
	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/keyboards"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/course"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/user"
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
	userRepo    user.Repository
	courseRepo  course.Repository
}

// NewHandler creates a new message handler.
func NewHandler(
	listCourses courseQueries.ListCoursesHandler,
	getOverview courseQueries.GetOverviewHandler,
	userRepo user.Repository,
	courseRepo course.Repository,
) *Handler {
	return &Handler{
		listCourses: listCourses,
		getOverview: getOverview,
		userRepo:    userRepo,
		courseRepo:  courseRepo,
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

	// In a later slice, user state will be updated to e.g., "course_details:123"
	// For now, we leave it in the course list state.

	return c.Send(msg, kb)
}

func (h *Handler) handleReturnToMainMenu(c telebot.Context, u user.User) error {
	if err := h.userRepo.UpdateLastMenu(context.Background(), u.ID, StateMain); err != nil {
		log.Printf("[handleReturnToMainMenu] Failed to update user state for UserID %d: %v", u.ID, err)
	}
	return c.Send("به منوی اصلی بازگشتید.", keyboards.NewMainMenu(u.IsAdmin))
}
