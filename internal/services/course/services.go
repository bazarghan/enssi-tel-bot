package course

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/2000ostd/enssi-tel-bot/internal/models"
	"github.com/2000ostd/enssi-tel-bot/internal/services/quiz"
	"github.com/2000ostd/enssi-tel-bot/internal/services/word"
	"gorm.io/gorm"
)

//==============================================================================
//  Service Definition & Initialization
//==============================================================================

// Service implements the CourseService interface.
type Service struct {
	db          *gorm.DB
	quizService quiz.QuizService
	wordService word.WordService
}

// NewService creates a new instance of the course Service.
func NewService(db *gorm.DB, qs quiz.QuizService, ws word.WordService) *Service {
	return &Service{
		db:          db,
		quizService: qs,
		wordService: ws,
	}
}

// Ensure Service implements CourseService interface
var _ CourseService = (*Service)(nil)

//==============================================================================
//  Core Internal Helper Functions
//==============================================================================

// --- UserCourse Management (Internal) ---

func (s *Service) getOrCreateUserCourseInternal(
	tx *gorm.DB,
	userID uint,
	courseID uint,
) (*models.UserCourse, bool, error) {

	var uc models.UserCourse
	isNew := false
	err := tx.Where("user_id = ? AND course_id = ?", userID, courseID).First(&uc).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("CourseService: No UserCourse found for UserID %d, CourseID %d. Creating.", userID, courseID)
			uc = models.UserCourse{UserID: userID, CourseID: courseID, Progress: 0}
			if createErr := tx.Create(&uc).Error; createErr != nil {
				return nil, false, fmt.Errorf("%w: %w", ErrEnrollmentFailed, createErr)
			}
			isNew = true
		} else {
			return nil, false, fmt.Errorf(
				"%w (UID %d, CID %d): %w",
				ErrUserCourseQueryFailed,
				userID,
				courseID,
				err,
			)
		}
	}
	return &uc, isNew, nil
}

// --- Course Data Retrieval (Internal) ---

func (s *Service) getTotalWordsInCourseInternal(courseID uint) (int64, error) {
	var totalWords int64
	if err := s.db.Model(&models.CourseWord{}).Where("course_id = ?", courseID).Count(&totalWords).Error; err != nil {
		return 0, fmt.Errorf("%w %d: %w", ErrTotalWordCountFailed, courseID, err)
	}
	return totalWords, nil
}

//==============================================================================
//  Learning Step Determination Logic
//==============================================================================

// --- Main Orchestrator for Learning Steps ---

func (s *Service) determineNextLearningStep(
	userID uint,
	courseID uint,
	userCourse *models.UserCourse,
	totalWordsInCourse int64,
) (*LearningContext, error) {

	if lc, handled := s.handleInitialOrEmptyCourse(
		userID,
		courseID,
		userCourse,
		totalWordsInCourse,
	); handled {
		return lc, nil
	}

	if lc, err := s.checkCourseCompletion(
		userID,
		courseID,
		userCourse,
		totalWordsInCourse,
	); err != nil || lc != nil {
		return lc, err
	}

	if lc, err := s.checkQuizDue(userID, courseID, userCourse); err != nil || lc != nil {
		return lc, err
	}

	return s.presentNextWord(userID, courseID, userCourse)
}

// --- Helper: Initial State or Empty Course ---

func (s *Service) handleInitialOrEmptyCourse(
	userID uint,
	courseID uint,
	userCourse *models.UserCourse,
	totalWordsInCourse int64,
) (*LearningContext, bool) {

	currentProgress := userCourse.Progress
	if totalWordsInCourse == 0 {
		return &LearningContext{
			UserID:        userID,
			CourseID:      courseID,
			UserProgress:  currentProgress,
			IsCourseEnded: true,
			MessageToUser: MsgCourseNoContent,
		}, true
	}
	return nil, false
}

// --- Helper: Course Completion Check ---

func (s *Service) checkCourseCompletion(
	userID uint,
	courseID uint,
	userCourse *models.UserCourse,
	totalWordsInCourse int64,
) (*LearningContext, error) {

	currentProgress := userCourse.Progress
	if totalWordsInCourse > 0 && currentProgress > uint(totalWordsInCourse) {
		finalQuizTriggerProgress := uint(totalWordsInCourse)
		if finalQuizTriggerProgress > 0 && finalQuizTriggerProgress%wordsPerQuizBlock == 0 {
			quizState, err := s.quizService.StartOrResumeQuiz(userID, courseID, finalQuizTriggerProgress)
			if err != nil {
				log.Printf("CourseService: Error checking for final quiz (UID %d, CID %d, Progress %d): %v", userID, courseID, finalQuizTriggerProgress, err)
			}
			if quizState != nil && !quizState.IsCompleted {
				return &LearningContext{
					UserID:        userID,
					CourseID:      courseID,
					UserProgress:  currentProgress,
					IsQuizDue:     true,
					QuizState:     quizState,
					MessageToUser: MsgFinalQuizForBlockPrompt,
				}, nil
			}
		}
		return &LearningContext{
			UserID:        userID,
			CourseID:      courseID,
			UserProgress:  currentProgress,
			IsCourseEnded: true,
			MessageToUser: MsgCourseCompleted,
		}, nil
	}
	return nil, nil
}

// --- Helper: Quiz Due Check ---

func (s *Service) checkQuizDue(
	userID uint,
	courseID uint,
	userCourse *models.UserCourse,
) (*LearningContext, error) {

	currentProgress := userCourse.Progress
	if currentProgress > 1 {

		if currentProgress > 0 && currentProgress%wordsPerQuizBlock == 0 {

			quizTriggerPoint := currentProgress

			// --- START: Gatekeeper check for previously passed quiz ---
			var passedAttemptsCount int64

			// Query to count successful, completed attempts for this specific milestone
			err := s.db.Model(&models.QuizAttempt{}).
				Joins("JOIN quizzes ON quizzes.id = quiz_attempts.quiz_id").
				Where("quiz_attempts.user_id = ? AND quizzes.course_id = ? AND quizzes.trigger_progress = ? AND quiz_attempts.is_completed = ? AND quiz_attempts.score >= ?",
					userID, courseID, quizTriggerPoint, true, quizPassThreshold).
				Count(&passedAttemptsCount).Error

			if err != nil {
				// A DB error here is serious, log and prevent quiz from starting
				log.Printf("CourseService: DB error checking for previously passed quiz for UserID %d, CourseID %d, Trigger %d: %v", userID, courseID, quizTriggerPoint, err)
				// Returning the error will halt the process
				return nil, err
			}

			if passedAttemptsCount > 0 {
				log.Printf("CourseService: UserID %d has already passed the quiz for trigger point %d. Skipping quiz.", userID, quizTriggerPoint)
				return nil, nil // Return successfully, indicating no quiz is due
			}
			// --- END: Gatekeeper check ---

			quizState, err := s.quizService.StartOrResumeQuiz(userID, courseID, quizTriggerPoint)
			if err != nil {
				return nil, fmt.Errorf("%w: checking quiz at progress %d (trigger %d): %w", ErrQuizIntegration, currentProgress, quizTriggerPoint, err)
			}

			messageToUser := fmt.Sprintf(MsgQuizDueAfterBlockPromptFmt, wordsPerQuizBlock)
			if quizState.CurrentQuestionNum != 0 {
				messageToUser = msgResumeActiveQuiz
			}

			if quizState != nil && !quizState.IsCompleted {
				return &LearningContext{
					UserID:        userID,
					CourseID:      courseID,
					UserProgress:  currentProgress,
					IsQuizDue:     true,
					QuizState:     quizState,
					MessageToUser: messageToUser,
				}, nil
			}
		}
	}
	return nil, nil
}

// --- Helper: Word Presentation ---

func (s *Service) presentNextWord(
	userID uint,
	courseID uint,
	userCourse *models.UserCourse,
) (*LearningContext, error) {

	currentProgress := userCourse.Progress
	wordData, err := s.wordService.GetWordDetailsForCourse(courseID, currentProgress+1, userID)
	if err != nil {
		if errors.Is(err, word.ErrCourseWordLinkNotFound) || errors.Is(err, word.ErrWordNotFound) {
			log.Printf("CourseService: Word not found for course %d at index %d. UserID %d. Considering end of available words.", courseID, currentProgress, userID)
			return &LearningContext{
				UserID:        userID,
				CourseID:      courseID,
				UserProgress:  currentProgress,
				IsCourseEnded: true,
				MessageToUser: MsgEndOfAvailableWords,
			}, nil
		}
		return nil, fmt.Errorf("%w: fetching word at progress %d: %w", ErrWordIntegration, currentProgress, err)
	}

	return &LearningContext{
		UserID:        userID,
		CourseID:      courseID,
		UserProgress:  currentProgress,
		IsQuizDue:     false,
		WordToDisplay: wordData,
		MessageToUser: MsgHereIsYourNextWord,
	}, nil
}

//==============================================================================
//  Public API Methods
//==============================================================================

func (s *Service) ListAvailableCourses(userID uint) ([]CourseSummaryView, error) {
	log.Printf("CourseService: ListAvailableCourses called for UserID: %d", userID)
	var courses []models.Course
	if err := s.db.Order("id ASC").Find(&courses).Error; err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCourseFetchFailed, err)
	}

	summaries := make([]CourseSummaryView, 0, len(courses))
	for _, course := range courses {
		var uc models.UserCourse
		s.db.Where("user_id = ? AND course_id = ?", userID, course.ID).First(&uc)

		totalWords, _ := s.getTotalWordsInCourseInternal(course.ID)
		isCompleted := false
		progressPercentage := 0
		wordsActuallyCompleted := uint(0)
		if uc.ID != 0 && uc.Progress > 0 {
			wordsActuallyCompleted = uc.Progress
		}
		isStartedByUser := false
		if uc.ID != 0 {
			isStartedByUser = true
		}

		if totalWords > 0 {
			if wordsActuallyCompleted >= uint(totalWords) {
				isCompleted = true
				progressPercentage = 100
			} else {
				progressPercentage = int((float64(wordsActuallyCompleted) / float64(totalWords)) * 100)
			}
		} else {
			if uc.ID != 0 && uc.Progress > 0 {
				isCompleted = true
				progressPercentage = 100
			} else {
				isCompleted = false
				progressPercentage = 0
			}
		}

		summaries = append(summaries, CourseSummaryView{
			ID:                 course.ID,
			Title:              course.Title,
			PersianTitle:       course.PersianTitle,
			ShortDescription:   course.Description,
			TotalWords:         totalWords,
			UserProgressWords:  wordsActuallyCompleted,
			ProgressPercentage: progressPercentage,
			IsCompletedByUser:  isCompleted,
			IsStartedByUser:    isStartedByUser,
		})
	}
	return summaries, nil
}

func (s *Service) GetCourseOverview(courseID uint, userID uint) (*CourseOverview, error) {
	log.Printf("CourseService: GetCourseOverview called for CourseID: %d, UserID: %d", courseID, userID)
	var course models.Course
	if err := s.db.First(&course, courseID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCourseNotFound
		}
		return nil, fmt.Errorf("failed to fetch course %d: %w", courseID, err)
	}

	var uc models.UserCourse
	s.db.Where("user_id = ? AND course_id = ?", userID, courseID).First(&uc)

	totalWords, _ := s.getTotalWordsInCourseInternal(courseID)
	isCompleted := false
	progressPercentage := 0
	wordsActuallyCompleted := uint(0)
	if uc.ID != 0 && uc.Progress > 0 {
		wordsActuallyCompleted = uc.Progress
	}
	isStartedByUser := false
	if uc.ID != 0 {
		isStartedByUser = true
	}

	if totalWords > 0 {
		if wordsActuallyCompleted >= uint(totalWords) {
			isCompleted = true
			progressPercentage = 100
		} else {
			progressPercentage = int((float64(wordsActuallyCompleted) / float64(totalWords)) * 100)
		}
	} else {
		if uc.ID != 0 && uc.Progress > 0 {
			isCompleted = true
			progressPercentage = 100
		} else {
			isCompleted = false
			progressPercentage = 0
		}
	}

	return &CourseOverview{
		ID:                     course.ID,
		Title:                  course.Title,
		PersianTitle:           course.PersianTitle,
		FullDescription:        course.Description,
		PersianFullDescription: course.PersianDescription,
		TotalWords:             totalWords,
		UserProgressWords:      wordsActuallyCompleted,
		ProgressPercentage:     progressPercentage,
		IsCompletedByUser:      isCompleted,
		IsStartedByUser:        isStartedByUser,
	}, nil
}

func (s *Service) GetUserCourse(userID uint, courseID uint) (*models.UserCourse, error) {
	log.Printf("CourseService: GetUserCourse called for UserID %d, CourseID %d", userID, courseID)
	var uc models.UserCourse
	err := s.db.Where("user_id = ? AND course_id = ?", userID, courseID).First(&uc).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserCourseNotFound
		}
		return nil, fmt.Errorf("failed to get UserCourse for UserID %d, CourseID %d: %w", userID, courseID, err)
	}
	return &uc, nil
}

func (s *Service) StartOrResumeLearningSession(courseID uint, userID uint) (*LearningContext, error) {
	log.Printf("CourseService: StartOrResumeLearningSession for CourseID: %d, UserID: %d", courseID, userID)
	var learningContext *LearningContext

	err := s.db.Transaction(func(tx *gorm.DB) error {
		uc, _, errUc := s.getOrCreateUserCourseInternal(tx, userID, courseID)
		if errUc != nil {
			return errUc
		}

		totalWords, errTw := s.getTotalWordsInCourseInternal(courseID)
		if errTw != nil {
			return errTw
		}

		nextStep, errNext := s.determineNextLearningStep(userID, courseID, uc, totalWords)
		if errNext != nil {
			return errNext
		}
		learningContext = nextStep
		return nil
	})
	if err != nil {
		return nil, err
	}
	return learningContext, nil
}

func (s *Service) AdvanceToNextWord(courseID uint, userID uint) (*LearningContext, error) {

	log.Printf("CourseService: AdvanceToNextWord for CourseID: %d, UserID: %d", courseID, userID)
	var learningContext *LearningContext

	err := s.db.Transaction(func(tx *gorm.DB) error {
		uc, _, errUc := s.getOrCreateUserCourseInternal(tx, userID, courseID)
		if errUc != nil {
			if errors.Is(errUc, ErrEnrollmentFailed) && strings.Contains(errUc.Error(), "UserCourse not found") {
				return ErrCannotAdvanceNoSession
			}
			return errUc
		}

		totalWords, errTw := s.getTotalWordsInCourseInternal(courseID)
		if errTw != nil {
			return errTw
		}

		uc.Progress++
		if errSave := tx.Save(uc).Error; errSave != nil {
			return fmt.Errorf(
				"%w: advancing progress for UserID %d, CourseID %d: %w",
				ErrProgressUpdateFailed,
				userID,
				courseID,
				errSave,
			)
		}
		log.Printf(
			"CourseService: Advanced progress for UserID %d, CourseID %d to %d.",
			userID,
			courseID,
			uc.Progress,
		)

		nextStep, errNext := s.determineNextLearningStep(userID, courseID, uc, totalWords)
		if errNext != nil {
			return errNext
		}
		learningContext = nextStep
		return nil
	})

	if err != nil {
		return nil, err
	}
	return learningContext, nil
}

func (s *Service) HandleQuizCompletion(
	userID uint,
	courseID uint,
	quizOutcome *quiz.QuizResult,
) (*LearningContext, error) {

	log.Printf(
		"CourseService: HandleQuizCompletion for UserID %d, CourseID %d. Quiz Type: %s, Passed: %t",
		userID,
		courseID,
		quizOutcome.QuizType,
		quizOutcome.Passed,
	)

	if quizOutcome.QuizType == models.QuizTypeReview {
		log.Printf("CourseService: Review quiz (ID: %d) completed for UserID %d. No course progress change.", quizOutcome.QuizID, userID)
		return &LearningContext{
			UserID:        userID,
			CourseID:      courseID,
			IsCourseEnded: true,
			MessageToUser: "جلسه مرور شما به پایان رسید. می‌توانید به یادگیری ادامه دهید.",
		}, nil
	}

	if quizOutcome.QuizType == models.QuizTypeCourseBlock {
		// Fetch the Quiz model to get its TriggerProgress
		var completedQuizModels models.Quiz
		if err := s.db.First(&completedQuizModels, quizOutcome.QuizID).Error; err != nil {
			log.Printf("CourseService: Error fetching Quiz (ID %d) details for HandleQuizCompletion: %v", quizOutcome.QuizID, err)
			// Fallback: proceed without marking words if quiz details can't be fetched
			return s.StartOrResumeLearningSession(courseID, userID)
		}

		if quizOutcome.Passed {

			log.Printf(
				"CourseService: COURSE_BLOCK Quiz (ID %d) passed for UserID %d, CourseID %d.",
				quizOutcome.QuizID,
				userID,
				courseID,
			)

			quizBlockEndIndex := completedQuizModels.TriggerProgress // Use TriggerProgress from the fetched Quiz model
			quizBlockStartIndex := uint(1)
			if quizBlockEndIndex >= wordsPerQuizBlock {
				quizBlockStartIndex = quizBlockEndIndex - wordsPerQuizBlock + 1
			}

			log.Printf("CourseService: Marking words from index %d to %d in CourseID %d as studied for UserID %d.", quizBlockStartIndex, quizBlockEndIndex, courseID, userID)
			var wordsInBlock []models.CourseWord
			err := s.db.Where("course_id = ? AND index >= ? AND index <= ?", courseID, quizBlockStartIndex, quizBlockEndIndex).
				Find(&wordsInBlock).Error
			if err != nil {
				log.Printf("CourseService: Error fetching words for block (CourseID %d, Index %d-%d) to mark as studied: %v", courseID, quizBlockStartIndex, quizBlockEndIndex, err)
			} else {
				for _, cw := range wordsInBlock {
					if errMark := s.wordService.MarkWordAsStudied(userID, cw.WordID, courseID); errMark != nil {
						log.Printf("CourseService: Error marking WordID %d (from CourseWord Index %d) as studied for UserID %d: %v", cw.WordID, cw.Index, userID, errMark)
					} else {
						log.Printf("CourseService: Successfully marked WordID %d (Index %d) as studied for UserID %d.", cw.WordID, cw.Index, userID)
					}
				}
			}

			log.Printf("CourseService: Quiz passed. Directly presenting next word for UserID %d, CourseID %d.", userID, courseID)

			// Fetch the most up-to-date UserCourse record to get the current progress.
			uc, err := s.GetUserCourse(userID, courseID)
			if err != nil {
				log.Printf("CourseService: Failed to get UserCourse for UserID %d, CourseID %d after quiz pass: %v", userID, courseID, err)
				// Fallback to StartOrResumeLearningSession if we can't get the user course.
				return s.StartOrResumeLearningSession(courseID, userID)
			}

			return s.presentNextWord(userID, courseID, uc)

		} else {
			log.Printf("CourseService: COURSE_BLOCK Quiz (ID %d) failed for UserID %d, CourseID %d.", quizOutcome.QuizID, userID, courseID)
			if quizOutcome.ShouldResetProgress {
				log.Printf("CourseService: Resetting progress to %d for UserID %d, CourseID %d.", quizOutcome.SuggestedNewProgress, userID, courseID)
				_, err := s.UpdateUserCourseProgress(userID, courseID, quizOutcome.SuggestedNewProgress)
				if err != nil {
					log.Printf("CourseService: Error updating progress after quiz failure for UserID %d: %v", userID, err)
				}
			}
			return s.StartOrResumeLearningSession(courseID, userID)
		}
	}

	log.Printf("CourseService: HandleQuizCompletion received unhandled quiz type '%s' for QuizID %d.", quizOutcome.QuizType, quizOutcome.QuizID)
	return s.StartOrResumeLearningSession(courseID, userID)
}

func (s *Service) UpdateUserCourseProgress(userID uint, courseID uint, newProgress uint) (*models.UserCourse, error) {
	log.Printf("CourseService: UpdateUserCourseProgress UserID %d, CourseID %d, NewProgress %d", userID, courseID, newProgress)
	var uc models.UserCourse
	err := s.db.Transaction(func(tx *gorm.DB) error {
		loadedUc, _, errUc := s.getOrCreateUserCourseInternal(tx, userID, courseID)
		if errUc != nil {
			return errUc
		}
		uc = *loadedUc
		uc.Progress = newProgress
		if errSave := tx.Save(&uc).Error; errSave != nil {
			return fmt.Errorf("%w for UserID %d, CourseID %d: %w", ErrProgressUpdateFailed, userID, courseID, errSave)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	log.Printf("CourseService: Successfully updated progress for UserID %d, CourseID %d to %d.", userID, courseID, uc.Progress)
	return &uc, nil
}
