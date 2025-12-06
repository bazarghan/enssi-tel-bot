package postgres

import (
	"context"
	"errors"

	"github.com/bazarghan/enssi-tel-bot/internal/domain/course"
	"gorm.io/gorm"
)

// CourseRepository is the GORM implementation of the course repository port.
type CourseRepository struct {
	db *gorm.DB
}

// NewCourseRepository creates a new course repository.
func NewCourseRepository(db *gorm.DB) *CourseRepository {
	return &CourseRepository{db: db}
}

// GetOrCreateUserCourse ensures a user course record exists and returns its progress state.
func (r *CourseRepository) GetOrCreateUserCourse(ctx context.Context, userID uint, courseID uint) (course.UserProgress, error) {
	var uc UserCourseModel
	err := r.db.WithContext(ctx).Where("user_id = ? AND course_id = ?", userID, courseID).First(&uc).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Create a new record because the user is starting the course.
		uc = UserCourseModel{UserID: userID, CourseID: courseID, Progress: 0}
		if createErr := r.db.WithContext(ctx).Create(&uc).Error; createErr != nil {
			return course.UserProgress{}, createErr
		}
	} else if err != nil {
		return course.UserProgress{}, course.ErrProgressQuery
	}

	// After getting or creating, calculate the progress percentage.
	// This duplicates GetUserProgress logic, suggesting a potential internal refactor later.
	totalWords, err := r.GetTotalWords(ctx, courseID)
	if err != nil {
		return course.UserProgress{}, err
	}

	progress := calculateProgress(uc, totalWords)
	return progress, nil

}

func calculateProgress(uc UserCourseModel, totalWords int) course.UserProgress {
	progress := course.UserProgress{
		WordsCompleted: int(uc.Progress),
	}
	if uc.ID != 0 {
		progress.IsStarted = true
	}

	if totalWords > 0 {
		progress.ProgressPercentage = int((float64(uc.Progress) / float64(totalWords)) * 100)
		if uc.Progress >= uint(totalWords) {
			progress.IsCompleted = true
			progress.ProgressPercentage = 100
		}
	} else if uc.Progress > 0 { // Case for courses with progress but somehow 0 words
		progress.IsCompleted = true
		progress.ProgressPercentage = 100
	}

	return progress
}

// FindAll retrieves all available courses from the database.
func (r *CourseRepository) FindAll(ctx context.Context) ([]course.Course, error) {
	var models []CourseModel
	if err := r.db.WithContext(ctx).Order("id ASC").Find(&models).Error; err != nil {
		return nil, course.ErrFetchFailed
	}

	courses := make([]course.Course, 0, len(models))
	for _, m := range models {
		totalWords, _ := r.GetTotalWords(ctx, m.ID)
		courses = append(courses, toDomainCourse(m, totalWords))
	}
	return courses, nil
}

// FindByID retrieves a single course by its ID.
func (r *CourseRepository) FindByID(ctx context.Context, courseID uint) (course.Course, error) {
	var model CourseModel
	if err := r.db.WithContext(ctx).First(&model, courseID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return course.Course{}, course.ErrNotFound
		}
		return course.Course{}, err
	}
	totalWords, _ := r.GetTotalWords(ctx, model.ID)
	return toDomainCourse(model, totalWords), nil
}

// FindByPersianTitle retrieves a single course by its Persian title.
func (r *CourseRepository) FindByPersianTitle(ctx context.Context, title string) (course.Course, error) {
	var model CourseModel
	// The title on the keyboard may have progress percentage, so we need a partial match.
	if err := r.db.WithContext(ctx).Where("persian_title LIKE ?", title+"%").First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return course.Course{}, course.ErrNotFound
		}
		return course.Course{}, err
	}
	totalWords, _ := r.GetTotalWords(ctx, model.ID)
	return toDomainCourse(model, totalWords), nil
}

// GetUserProgress retrieves a specific user's progress for a given course.
func (r *CourseRepository) GetUserProgress(ctx context.Context, userID uint, courseID uint) (course.UserProgress, error) {
	var uc UserCourseModel
	err := r.db.WithContext(ctx).Where("user_id = ? AND course_id = ?", userID, courseID).First(&uc).Error

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return course.UserProgress{}, course.ErrProgressQuery
	}

	totalWords, err := r.GetTotalWords(ctx, courseID)
	if err != nil {
		return course.UserProgress{}, err
	}

	progress := course.UserProgress{}
	if uc.ID != 0 {
		progress.IsStarted = true
		if totalWords > 0 {
			progress.ProgressPercentage = int((float64(uc.Progress) / float64(totalWords)) * 100)
			if uc.Progress >= uint(totalWords) {
				progress.IsCompleted = true
				progress.ProgressPercentage = 100
			}
		}
	}
	return progress, nil
}

// IncrementProgress advances a user's progress in a course by one.
func (r *CourseRepository) IncrementProgress(ctx context.Context, userID, courseID uint) (course.UserProgress, error) {
	var uc UserCourseModel
	err := r.db.WithContext(ctx).Where("user_id = ? AND course_id = ?", userID, courseID).First(&uc).Error
	if err != nil {
		// Should not happen if user is in a course, but handle defensively.
		return course.UserProgress{}, course.ErrProgressQuery
	}

	uc.Progress++
	if err := r.db.WithContext(ctx).Save(&uc).Error; err != nil {
		return course.UserProgress{}, err
	}

	totalWords, _ := r.GetTotalWords(ctx, courseID)
	return calculateProgress(uc, totalWords), nil
}

// GetTotalWords retrieves the number of words in a course.
func (r *CourseRepository) GetTotalWords(ctx context.Context, courseID uint) (int, error) {
	var totalWords int64
	err := r.db.WithContext(ctx).Model(&CourseWordModel{}).Where("course_id = ?", courseID).Count(&totalWords).Error
	return int(totalWords), err
}

// Add this new function to the file
func (r *CourseRepository) SetProgress(ctx context.Context, userID, courseID, newProgress uint) error {
	result := r.db.WithContext(ctx).Model(&UserCourseModel{}).
		Where("user_id = ? AND course_id = ?", userID, courseID).
		Update("progress", newProgress)

	return result.Error
}

// CountActive counts the number of active courses for a user.
func (r *CourseRepository) CountActive(ctx context.Context, userID uint) (int, error) {
	var count int64

	err := r.db.WithContext(ctx).Model(&UserCourseModel{}).
		Where("user_id = ?", userID).
		Count(&count).Error
	return int(count), err
}
