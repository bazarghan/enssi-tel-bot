package course

import (
	"errors"
	"fmt"

	"github.com/2000ostd/enssi-tel-bot/internal/models"

	"gorm.io/gorm"
	"log"
)

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
