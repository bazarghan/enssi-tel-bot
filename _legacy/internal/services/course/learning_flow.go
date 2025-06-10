package course

import (
	"errors"
	"fmt"
	"log"

	"github.com/2000ostd/enssi-tel-bot/internal/models"
	"github.com/2000ostd/enssi-tel-bot/internal/services/word"
)

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
