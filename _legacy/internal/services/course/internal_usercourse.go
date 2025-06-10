package course

import (
	"errors"
	"fmt"
	"log"

	"github.com/2000ostd/enssi-tel-bot/internal/models"
	"gorm.io/gorm"
)

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
			log.Printf(
				"CourseService: No UserCourse found for UserID %d, CourseID %d. Creating.",
				userID,
				courseID,
			)
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
