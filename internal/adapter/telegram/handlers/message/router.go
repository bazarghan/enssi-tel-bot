package message

import (
	"strings"

	"github.com/2000ostd/enssi-tel-bot/internal/domain/user"
	"github.com/2000ostd/enssi-tel-bot/internal/platform/observability/logger"

	sc "github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/stateconstants"
	ui "github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/uiconstants"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/achievement"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/course"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/quiz"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/word"
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

// Router holds instances of all sub-handlers and routes incoming messages.
type Router struct {
	logger          logger.Logger
	userRepo        user.Repository
	AdminHandler    *AdminHandler
	courseHandler   *CourseHandler
	profileHandler  *ProfileHandler
	mainMenuHandler *MainMenuHandler
}

// NewRouter creates and initializes all sub-handlers and the main router.
func NewRouter(
	appLogger logger.Logger,
	listCourses courseQueries.ListCoursesHandler,
	getOverview courseQueries.GetOverviewHandler,
	getProfile getProfileQry.GetProfileHandler,
	getStats adminQueries.GetStatsHandler,
	//Broadcast adminCmd.BroadcastHandler,
	getAllAchievements achQueries.GetAllHandler,
	advanceWord advanceCmd.AdvanceWordHandler,
	startSession startCmd.StartSessionHandler,
	userRepo user.Repository,
	quizRepo quiz.Repository,
	courseRepo course.Repository,
	wordRepo word.Repository,
	createReviewQuiz quizCmd.CreateReviewQuizHandler,
	cacheMedia cacheCmd.CacheMediaHandler,
	achRepo achievement.Repository,
	imgSvc achievement.ImageGenerator,
	hasPendingReview reviewQueries.Handler,
) *Router {

	adminHandler := NewAdminHandler(
		appLogger,
		getStats,
		userRepo,
	)

	courseHandler := NewCourseHandler(
		appLogger,
		getOverview,
		advanceWord,
		startSession,
		userRepo,
		courseRepo,
		quizRepo,
		cacheMedia,
	)

	profileHandler := NewProfileHandler(
		appLogger,
		getProfile,
		getAllAchievements,
		userRepo,
		achRepo,
		imgSvc,
	)

	mainMenuHandler := NewMainMenuHandler(
		appLogger,
		listCourses,
		userRepo,
		quizRepo,
		wordRepo,
		createReviewQuiz,
		hasPendingReview,
		profileHandler,
	)

	return &Router{
		logger:          appLogger,
		userRepo:        userRepo,
		AdminHandler:    adminHandler,
		courseHandler:   courseHandler,
		profileHandler:  profileHandler,
		mainMenuHandler: mainMenuHandler,
	}
}

// Handle routes incoming text messages to the appropriate sub-handler.
func (r *Router) Handle(c telebot.Context) error {
	ctxUser, ok := c.Get("dbUser").(user.User)
	if !ok {
		r.logger.Error("Error registering or finding user", "error", ok, "telegram_id", c.Sender().ID)
		return c.Send("خطا در پردازش اطلاعات کاربر.")
	}

	userInput := strings.TrimSpace(c.Text())
	stateBase, _, _ := strings.Cut(ctxUser.LastMenu, ":")

	// Global commands
	if userInput == ui.BtnReturnToMainMenuText {
		return r.mainMenuHandler.handleReturnToMainMenu(c, ctxUser)
	}

	// State-based routing to sub-handlers
	switch {
	case ctxUser.LastMenu == sc.StateMain,
		ctxUser.LastMenu == sc.StateDailyReviewMenu:

		return r.mainMenuHandler.Handle(c, ctxUser, userInput)

	case ctxUser.LastMenu == sc.StateCourseList,
		stateBase == sc.StateCourseDetails,
		stateBase == sc.StateInCourse:
		return r.courseHandler.Handle(c, ctxUser, userInput)

	case stateBase == sc.StateProfileMenu,
		ctxUser.LastMenu == sc.StateAchievementList:
		return r.profileHandler.Handle(c, ctxUser, userInput)

	case ctxUser.LastMenu == sc.StateAdminPanel,
		ctxUser.LastMenu == sc.StateAdminBroadcast:
		return r.AdminHandler.Handle(c, ctxUser, userInput)
	}

	r.logger.Info("Unhandled text message", "user_id", ctxUser.ID, "state", ctxUser.LastMenu, "input", userInput)
	return nil
}
