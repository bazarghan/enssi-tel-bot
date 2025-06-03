package handlers

import (
	"fmt"
	"strconv"
	"strings"
)

// Callback Data Prefixes
const (
	CourseSelectCallbackPrefix = "cs:"       // cs:<course_id>
	QuizAnswerCallbackPrefix   = "quiz_ans:" // quiz_ans:<attempt_ID>:<option_ID>
)

// User state values for `LastMenu` (base states)
const (
	StateMain              = "main"
	StateCourseList        = "course_list"
	StateCourseDetailsBase = "course_details" // Base for "course_details:<id>"
	StateInCourseBase      = "in_course"      // Base for "in_course:<id>"
	StateInQuizBase        = "in_quiz"        // Base for "in_quiz:<id>"
)

// Prefixes for parsing states with IDs
const (
	StateCourseDetailsPrefix = StateCourseDetailsBase + ":"
	StateInCoursePrefix      = StateInCourseBase + ":"
	StateInQuizPrefix        = StateInQuizBase + ":"
)

// Functions to generate full state strings
func CourseDetailsMenuState(courseID uint) string {
	return StateCourseDetailsPrefix + strconv.FormatUint(uint64(courseID), 10)
}

func InCourseMenuState(courseID uint) string {
	return StateInCoursePrefix + strconv.FormatUint(uint64(courseID), 10)
}

func InQuizMenuState(attemptID uint) string {
	return StateInQuizPrefix + strconv.FormatUint(uint64(attemptID), 10)
}

// ParseIDFromState extracts an ID from a menu state string given a prefix.
// Example: ParseIDFromState("in_course:123", StateInCoursePrefix) returns 123.
func ParseIDFromState(menuState string, prefix string) (uint, error) {
	if !strings.HasPrefix(menuState, prefix) {
		return 0, fmt.Errorf("invalid menu state format: expected prefix '%s', got '%s'", prefix, menuState)
	}
	idStr := strings.TrimPrefix(menuState, prefix)
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid ID in menu state '%s': %w", menuState, err)
	}
	return uint(id), nil
}
