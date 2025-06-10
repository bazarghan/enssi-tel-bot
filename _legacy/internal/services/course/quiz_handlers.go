package course

import (
	"github.com/2000ostd/enssi-tel-bot/internal/models"
	"github.com/2000ostd/enssi-tel-bot/internal/services/quiz"
	"log"
)

func (s *Service) handleReviewQuizCompletion(
	userID uint,
	courseID uint,
) (*LearningContext, error) {

	log.Printf(
		"CourseService: Review quiz completed for UserID %d. No course progress change.", userID,
	)
	return &LearningContext{
		UserID:        userID,
		CourseID:      courseID,
		IsCourseEnded: true,
		MessageToUser: "جلسه مرور شما به پایان رسید. می‌توانید به یادگیری ادامه دهید.",
	}, nil
}

func (s *Service) handleCourseBlockQuizPassed(
	userID uint,
	courseID uint,
	quizOutcome *quiz.QuizResult,
	completedQuizModels models.Quiz,
) (*LearningContext, error) {

	log.Printf(
		"CourseService: COURSE_BLOCK Quiz (ID %d) passed for UserID %d, CourseID %d.",
		quizOutcome.QuizID,
		userID,
		courseID,
	)

	quizBlockEndIndex := completedQuizModels.TriggerProgress
	quizBlockStartIndex := uint(1)
	if quizBlockEndIndex >= wordsPerQuizBlock {
		quizBlockStartIndex = quizBlockEndIndex - wordsPerQuizBlock + 1
	}

	log.Printf(
		"CourseService: Marking words from index %d to %d in CourseID %d as studied for UserID %d.",
		quizBlockStartIndex,
		quizBlockEndIndex,
		courseID,
		userID,
	)
	var wordsInBlock []models.CourseWord
	err := s.db.Where("course_id = ? AND index >= ? AND index <= ?", courseID, quizBlockStartIndex, quizBlockEndIndex).
		Find(&wordsInBlock).Error
	if err != nil {
		log.Printf(
			"CourseService: Error fetching words for block (CourseID %d, Index %d-%d) to mark as studied: %v",
			courseID,
			quizBlockStartIndex,
			quizBlockEndIndex,
			err,
		)
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

	// --- ADD/REPLACE ACHIEVEMENT PROGRESS LOGIC HERE ---
	var achievementInfo *AchievementUpdateInfo

	var currentCourseModels models.Course
	if err := s.db.First(&currentCourseModels, courseID).Error; err == nil {
		if currentCourseModels.LinkedAchievementID != 0 {
			linkedAchID := currentCourseModels.LinkedAchievementID
			totalWords, _ := s.getTotalWordsInCourseInternal(courseID)

			if completedQuizModels.TriggerProgress >= uint(totalWords) {
				log.Printf("CourseService: Final quiz passed for CourseID %d. Completing achievement %d.", courseID, linkedAchID)
				_, newBitsetState, errAch := s.userService.CompleteAchievement(userID, linkedAchID)
				if errAch != nil {
					log.Printf(
						"CourseService: Error awarding achievement progress for UserID %d, AchievementID %d: %v",
						userID,
						linkedAchID,
						errAch,
					)
				} else {
					achievementInfo = s.populateAchievementInfo(linkedAchID, newBitsetState)
				}
			} else {
				itemsToRevealThisQuiz := 12
				log.Printf("CourseService: Quiz passed for CourseID %d. Awarding %d items for achievement %d.", courseID, itemsToRevealThisQuiz, linkedAchID)
				_, newBitsetState, errAch := s.userService.AwardAchievementProgress(userID, linkedAchID, itemsToRevealThisQuiz)
				if errAch != nil {
					log.Printf(
						"CourseService: Error awarding achievement progress for UserID %d, AchievementID %d: %v",
						userID,
						linkedAchID,
						errAch,
					)
				} else {
					achievementInfo = s.populateAchievementInfo(linkedAchID, newBitsetState)
				}
			}
		}
	} else {
		log.Printf("CourseService: Could not fetch course details for CourseID %d to check for linked achievement.", courseID)
	}

	log.Printf("CourseService: Quiz passed for UserID %d. Progress updated. Awaiting user action to continue.", userID)

	uc, err := s.GetUserCourse(userID, courseID)
	if err != nil {
		log.Printf("CourseService: Failed to get UserCourse for UserID %d, CourseID %d after quiz pass: %v", userID, courseID, err)
		return s.StartOrResumeLearningSession(courseID, userID)
	}

	totalWords, _ := s.getTotalWordsInCourseInternal(courseID)
	isCourseNowComplete := totalWords > 0 && uc.Progress >= uint(totalWords)

	learningContext := &LearningContext{
		UserID:                userID,
		CourseID:              courseID,
		UserProgress:          uc.Progress,
		IsQuizDue:             false,
		QuizState:             nil,
		WordToDisplay:         nil,
		MessageToUser:         "",
		IsCourseEnded:         isCourseNowComplete,
		AchievementProgressed: achievementInfo,
	}
	return learningContext, nil

}

func (s *Service) handleCourseBlockQuizFailed(
	userID uint,
	courseID uint,
	quizOutcome *quiz.QuizResult,
) (*LearningContext, error) {

	log.Printf(
		"CourseService: COURSE_BLOCK Quiz (ID %d) failed for UserID %d, CourseID %d.",
		quizOutcome.QuizID,
		userID,
		courseID,
	)
	if quizOutcome.ShouldResetProgress {
		log.Printf("CourseService: Resetting progress to %d for UserID %d, CourseID %d.", quizOutcome.SuggestedNewProgress, userID, courseID)
		_, err := s.UpdateUserCourseProgress(userID, courseID, quizOutcome.SuggestedNewProgress)
		if err != nil {
			log.Printf("CourseService: Error updating progress after quiz failure for UserID %d: %v", userID, err)
		}
	}

	// --- START OF MODIFICATION (QUIZ FAILED) ---

	// Fetch the user's course state after the potential progress reset.
	uc, err := s.GetUserCourse(userID, courseID)
	if err != nil {
		log.Printf("CourseService: Failed to get UserCourse for UserID %d, CourseID %d after quiz fail: %v", userID, courseID, err)
		return s.StartOrResumeLearningSession(courseID, userID) // Fallback
	}

	// Return a context indicating the quiz is over, with no next word.
	learningContext := &LearningContext{
		UserID:        userID,
		CourseID:      courseID,
		UserProgress:  uc.Progress,
		IsCourseEnded: false,
		WordToDisplay: nil, // This is the key change.
	}
	return learningContext, nil

}

// ────────────────────────────────────────────────
// HandleQuizCompletion function
// ────────────────────────────────────────────────

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

	switch quizOutcome.QuizType {

	case models.QuizTypeReview:
		return s.handleReviewQuizCompletion(userID, courseID)

	case models.QuizTypeCourseBlock:
		var completedQuizModels models.Quiz
		if err := s.db.First(&completedQuizModels, quizOutcome.QuizID).Error; err != nil {
			log.Printf("CourseService: Error fetching Quiz (ID %d) details for HandleQuizCompletion: %v", quizOutcome.QuizID, err)
			return s.StartOrResumeLearningSession(courseID, userID)
		}

		if quizOutcome.Passed {
			return s.handleCourseBlockQuizPassed(userID, courseID, quizOutcome, completedQuizModels)
		}
		return s.handleCourseBlockQuizFailed(userID, courseID, quizOutcome)
	}

	log.Printf("CourseService: HandleQuizCompletion received unhandled quiz type '%s' for QuizID %d.", quizOutcome.QuizType, quizOutcome.QuizID)
	return s.StartOrResumeLearningSession(courseID, userID)
}
