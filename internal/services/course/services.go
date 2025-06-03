package course

import (
	"errors"
	"fmt"
	"github.com/2000ostd/enssi-tel-bot/internal/models"
	"github.com/2000ostd/enssi-tel-bot/internal/services/quiz"
	"github.com/2000ostd/enssi-tel-bot/internal/services/word"
	"gorm.io/gorm"
	"log"
	"strings"
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
			uc = models.UserCourse{UserID: userID, CourseID: courseID, Progress: 0} // Start at 0, session start will advance to 1
			if createErr := tx.Create(&uc).Error; createErr != nil {
				return nil, false, fmt.Errorf("%w: %w", ErrEnrollmentFailed, createErr)
			}
			isNew = true
		} else {
			return nil, false, fmt.Errorf("failed to fetch UserCourse (UID %d, CID %d): %w", userID, courseID, err)
		}
	}
	return &uc, isNew, nil
}

// --- Course Data Retrieval (Internal) ---

func (s *Service) getTotalWordsInCourseInternal(courseID uint) (int64, error) {
	var totalWords int64
	if err := s.db.Model(&models.CourseWord{}).Where("course_id = ?", courseID).Count(&totalWords).Error; err != nil {
		return 0, fmt.Errorf("failed to count total words for course %d: %w", courseID, err)
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

	// -----------------------------------------
	// 1. Handle initial progress or empty course
	if lc, handled := s.handleInitialOrEmptyCourse(
		userID,
		courseID,
		userCourse,
		totalWordsInCourse,
	); handled {
		return lc, nil
	}

	// -----------------------------------------
	// 2. Check if course is completed (all words seen, potential final quiz)
	if lc, err := s.checkCourseCompletion(
		userID,
		courseID,
		userCourse,
		totalWordsInCourse,
	); err != nil || lc != nil {
		return lc, err
	}

	// -----------------------------------------
	// 3. Check if a regular quiz is due
	if lc, err := s.checkQuizDue(userID, courseID, userCourse); err != nil || lc != nil {
		return lc, err
	}

	// -----------------------------------------
	// 4. If no quiz and not completed, present the next word
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

	if currentProgress == 0 && totalWordsInCourse > 0 {
		userCourse.Progress = 1
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
	if currentProgress > uint(totalWordsInCourse) && totalWordsInCourse > 0 {
		// Check for final quiz
		finalQuizTriggerProgress := uint(totalWordsInCourse)
		if finalQuizTriggerProgress > 0 && finalQuizTriggerProgress%wordsPerQuizBlock == 0 {
			quizState, err := s.quizService.StartOrResumeQuiz(userID, courseID, finalQuizTriggerProgress)
			if err != nil {
				return nil, fmt.Errorf("%w: checking final quiz: %w", ErrQuizIntegration, err)
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
		// No final quiz or it's done
		return &LearningContext{
			UserID:        userID,
			CourseID:      courseID,
			UserProgress:  currentProgress,
			IsCourseEnded: true,
			MessageToUser: MsgCourseCompleted,
		}, nil
	}
	return nil, nil // Not completed, or no error
}

// --- Helper: Quiz Due Check ---

func (s *Service) checkQuizDue(userID uint, courseID uint, userCourse *models.UserCourse) (*LearningContext, error) {
	currentProgress := userCourse.Progress
	if currentProgress > 0 && currentProgress%wordsPerQuizBlock == 0 {
		// This is the check before fetching a word AT currentProgress. If a quiz is due AT currentProgress.
		quizState, err := s.quizService.StartOrResumeQuiz(userID, courseID, currentProgress)
		if err != nil {
			return nil, fmt.Errorf("%w: checking quiz at progress %d: %w", ErrQuizIntegration, currentProgress, err)
		}

		messageToUser := fmt.Sprintf(MsgQuizDueAfterBlockPromptFmt, wordsPerQuizBlock)
		if quizState.CurrentQuestionNum != 0 { // Quiz in progress
			messageToUser = "ادامه آزمون... ✍️" // Or a more appropriate message
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
		// Quiz completed for this block. Logic should proceed to word after this block.
		// This indicates that UserCourse.Progress should likely already be currentProgress + 1
		// if the quiz for 'currentProgress' was just completed.
		// If we are here, it implies the quiz is done and we're checking for the next thing.
	}
	return nil, nil // No quiz due, or no error
}

// --- Helper: Word Presentation ---

func (s *Service) presentNextWord(userID uint, courseID uint, userCourse *models.UserCourse) (*LearningContext, error) {
	currentProgress := userCourse.Progress // This is the word index to fetch.

	wordData, err := s.wordService.GetWordDetailsForCourse(courseID, currentProgress, userID)
	if err != nil {
		if errors.Is(err, word.ErrCourseWordLinkNotFound) || errors.Is(err, word.ErrWordNotFound) {
			log.Printf("CourseService: Word not found for course %d at index %d. Considering end of course (or gap in words).", courseID, currentProgress)
			return &LearningContext{
				UserID:        userID,
				CourseID:      courseID,
				UserProgress:  currentProgress,
				IsCourseEnded: true, // Or maybe a specific "word not found" state?
				MessageToUser: MsgEndOfAvailableWords,
			}, nil
		}
		return nil, fmt.Errorf("%w: fetching word at progress %d: %w", ErrWordIntegration, currentProgress, err)
	}

	// The decision to MarkWordAsStudied when presenting vs. when advancing is important.
	// Current AdvanceToNextWord marks the *previous* word. StartOrResume doesn't mark.
	// If presenting means "seen", it could be marked here.
	// However, explicit marking on "next" action is safer.

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

// --- Course Information & User Progress Retrieval ---

func (s *Service) ListAvailableCourses(userID uint) ([]CourseSummaryView, error) {
	log.Printf("CourseService: ListAvailableCourses called for UserID: %d", userID)
	var courses []models.Course
	if err := s.db.Order("id ASC").Find(&courses).Error; err != nil { // Added Order for consistency
		return nil, fmt.Errorf("failed to fetch courses: %w", err)
	}

	summaries := make([]CourseSummaryView, 0, len(courses))
	for _, course := range courses {
		var uc models.UserCourse                                                  // Don't create if not exists for summary
		s.db.Where("user_id = ? AND course_id = ?", userID, course.ID).First(&uc) // Ignore error if not found for summary

		totalWords, _ := s.getTotalWordsInCourseInternal(course.ID)

		isCompleted := false
		progressPercentage := 0
		userProgressWords := uint(0)

		if uc.ID != 0 { // UserCourse exists
			userProgressWords = uc.Progress
			if totalWords > 0 {
				// Progress points to the *next* word. If uc.Progress is 1, 0 words completed.
				// If uc.Progress is totalWords + 1, all words completed.
				wordsCompleted := uint(0)
				if uc.Progress > 1 {
					wordsCompleted = uc.Progress - 1
				}
				if wordsCompleted >= uint(totalWords) {
					isCompleted = true
					progressPercentage = 100
					userProgressWords = uint(totalWords) // Display capped at total words
				} else {
					progressPercentage = int((float64(wordsCompleted) / float64(totalWords)) * 100)
				}
			} else if uc.Progress > 0 { // Has progress in an empty course
				isCompleted = true
				progressPercentage = 100
			}
		}

		summaries = append(summaries, CourseSummaryView{
			ID: course.ID, Title: course.Title, PersianTitle: course.PersianTitle,
			ShortDescription:   course.Description, // Assuming Description is short enough
			TotalWords:         totalWords,
			UserProgressWords:  userProgressWords, // This is the next word index, not words completed
			ProgressPercentage: progressPercentage,
			IsCompletedByUser:  isCompleted,
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
	s.db.Where("user_id = ? AND course_id = ?", userID, courseID).First(&uc) // Ignore error

	totalWords, _ := s.getTotalWordsInCourseInternal(courseID)
	isCompleted := false
	progressPercentage := 0
	userProgressWords := uint(0)

	if uc.ID != 0 {
		userProgressWords = uc.Progress
		if totalWords > 0 {
			wordsCompleted := uint(0)
			if uc.Progress > 1 {
				wordsCompleted = uc.Progress - 1
			}
			if wordsCompleted >= uint(totalWords) {
				isCompleted = true
				progressPercentage = 100
				userProgressWords = uint(totalWords)
			} else {
				progressPercentage = int((float64(wordsCompleted) / float64(totalWords)) * 100)
			}
		} else if uc.Progress > 0 {
			isCompleted = true
			progressPercentage = 100
		}
	}

	return &CourseOverview{
		ID: course.ID, Title: course.Title, PersianTitle: course.PersianTitle,
		FullDescription: course.Description, PersianFullDescription: course.PersianDescription,
		TotalWords: totalWords, UserProgressWords: userProgressWords,
		ProgressPercentage: progressPercentage, IsCompletedByUser: isCompleted,
	}, nil
}

func (s *Service) GetUserCourse(userID uint, courseID uint) (*models.UserCourse, error) {
	log.Printf("CourseService: GetUserCourse called for UserID %d, CourseID %d", userID, courseID)
	var uc models.UserCourse
	err := s.db.Where("user_id = ? AND course_id = ?", userID, courseID).First(&uc).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Return a zero-value UserCourse and ErrUserCourseNotFound if strict about it
			// Or return nil, ErrUserCourseNotFound
			return nil, ErrUserCourseNotFound
		}
		return nil, fmt.Errorf("failed to get UserCourse for UserID %d, CourseID %d: %w", userID, courseID, err)
	}
	return &uc, nil
}

// --- Learning Session Lifecycle ---

func (s *Service) StartOrResumeLearningSession(courseID uint, userID uint) (*LearningContext, error) {
	log.Printf("CourseService: StartOrResumeLearningSession for CourseID: %d, UserID: %d", courseID, userID)
	var learningContext *LearningContext

	err := s.db.Transaction(func(tx *gorm.DB) error {
		uc, isNewUc, errUc := s.getOrCreateUserCourseInternal(tx, userID, courseID)
		if errUc != nil {
			return errUc
		}

		// If it's a brand new UserCourse (uc.Progress is 0) or user chose to "Start Course",
		// set progress to 1 to indicate the first word is now active.
		if isNewUc || uc.Progress == 0 {
			if uc.Progress == 0 { // Only update if it was actually 0
				totalWords, _ := s.getTotalWordsInCourseInternal(courseID) // Check if course has words
				if totalWords > 0 {
					uc.Progress = 1 // Start at the first word
					if errSave := tx.Save(uc).Error; errSave != nil {
						return fmt.Errorf("%w: setting initial progress for UserID %d, CourseID %d: %w", ErrProgressUpdateFailed, userID, courseID, errSave)
					}
					log.Printf("CourseService: Set initial progress for UserID %d, CourseID %d to 1.", userID, courseID)
				} else {
					// Empty course, progress remains 0
					log.Printf("CourseService: Course %d is empty, UserID %d progress remains 0.", courseID, userID)
				}
			}
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
			// If UserCourse doesn't exist, AdvanceToNextWord is not logical.
			if errors.Is(errUc, ErrEnrollmentFailed) && strings.Contains(errUc.Error(), "UserCourse not found") { // Check if it was not found vs creation failed
				return ErrCannotAdvanceNoSession
			}
			return errUc
		}

		// Mark current word (uc.Progress) as studied before advancing
		// This assumes uc.Progress points to the word just completed / being displayed
		// Only mark if progress is valid ( > 0 and within total word count)
		totalWords, errTw := s.getTotalWordsInCourseInternal(courseID)
		if errTw != nil {
			return errTw
		}
		if uc.Progress > 0 && uc.Progress <= uint(totalWords) {
			// Need WordID for uc.Progress
			var cw models.CourseWord
			if errCw := tx.Where("course_id = ? AND index = ?", courseID, uc.Progress).First(&cw).Error; errCw == nil {
				if errMark := s.wordService.MarkWordAsStudied(userID, cw.WordID, courseID); errMark != nil {
					log.Printf("CourseService: Failed to mark word (Index %d, ID %d) as studied for UserID %d: %v", uc.Progress, cw.WordID, userID, errMark)
					// Non-fatal for advancing, but log it.
				}
			} else {
				log.Printf("CourseService: Could not find CourseWord for CourseID %d, Index %d to mark as studied: %v", courseID, uc.Progress, errCw)
			}
		}

		// Now advance progress to the next item
		uc.Progress++
		if errSave := tx.Save(uc).Error; errSave != nil {
			return fmt.Errorf("%w: advancing progress for UserID %d, CourseID %d: %w", ErrProgressUpdateFailed, userID, courseID, errSave)
		}
		log.Printf("CourseService: Advanced progress for UserID %d, CourseID %d to %d.", userID, courseID, uc.Progress)

		// totalWords already fetched above for MarkWordAsStudied check

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

// --- Quiz Result Processing ---

func (s *Service) HandleQuizCompletion(userID uint, courseID uint, quizOutcome *quiz.QuizResult) (*LearningContext, error) {
	log.Printf("CourseService: HandleQuizCompletion for UserID %d, CourseID %d. Quiz Passed: %t", userID, courseID, quizOutcome.Passed)

	if !quizOutcome.Passed && quizOutcome.ShouldResetProgress {
		log.Printf("CourseService: Quiz failed for UserID %d, CourseID %d. Resetting progress to %d.", userID, courseID, quizOutcome.SuggestedNewProgress)
		_, err := s.UpdateUserCourseProgress(userID, courseID, quizOutcome.SuggestedNewProgress)
		if err != nil {
			// Log the error, but proceed to determine next step from the (attempted) new progress
			log.Printf("CourseService: Error updating progress after quiz failure for UserID %d: %v", userID, err)
			// Even if save failed, we proceed with the intended reset progress for determining next step
		}
	}
	// If quiz passed, progress isn't reset by quiz. User is still at quiz trigger point.
	// We need to advance them past the quiz to the next word.
	// This means if quiz trigger was at progress N, they should now be at N+1 if they pass.
	// StartOrResumeLearningSession will pick up from the current UserCourse.Progress.
	// If they passed, their UserCourse.Progress is still at the quiz trigger.
	// If they failed and progress was reset, UserCourse.Progress is now at the reset point.

	if quizOutcome.Passed {
		// If they passed, they need to move to the content *after* the quiz block.
		// The quiz was triggered at quizOutcome.QuizTriggerProgress (assuming this info is on QuizResult or we fetch Quiz by quizOutcome.QuizID)
		// For now, let's assume the current UserCourse.Progress is still at the quiz trigger point.
		// We call AdvanceToNextWord which will increment progress from there.
		log.Printf("CourseService: Quiz passed for UserID %d, CourseID %d. Advancing to content after quiz.", userID, courseID)
		return s.AdvanceToNextWord(courseID, userID)
	} else {
		// If failed (and progress reset), or passed but no reset needed (and we aren't explicitly advancing past quiz here)
		// just determine the next step from current (potentially reset) progress.
		log.Printf("CourseService: Quiz failed (or passed with no explicit advance here) for UserID %d, CourseID %d. Determining next step from current progress.", userID, courseID)
		return s.StartOrResumeLearningSession(courseID, userID)
	}
}

// --- Direct User Progress Updates ---

func (s *Service) UpdateUserCourseProgress(userID uint, courseID uint, newProgress uint) (*models.UserCourse, error) {
	log.Printf("CourseService: UpdateUserCourseProgress UserID %d, CourseID %d, NewProgress %d", userID, courseID, newProgress)
	var uc models.UserCourse
	// Use transaction for get-or-create and update
	err := s.db.Transaction(func(tx *gorm.DB) error {
		loadedUc, _, errUc := s.getOrCreateUserCourseInternal(tx, userID, courseID)
		if errUc != nil {
			return errUc
		}
		uc = *loadedUc // Assign to outer scope variable
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
