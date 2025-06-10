package course

import (
	"errors"
	"fmt"
	"gorm.io/gorm"
	"log"
	"strings"
)

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
