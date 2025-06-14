package quiz

type QuizType string

const (
	CourseBlock       QuizType = "COURSE_BLOCK"
	Review            QuizType = "REVIEW"
	QuizPassThreshold          = 9
)

// Question represents a single quiz question and its possible options.
type Question struct {
	ID      uint
	Text    string
	WordID  uint
	Options []Option
}

// Option is one possible answer to a question.
type Option struct {
	ID        uint
	Text      string
	IsCorrect bool
}

// Attempt represents a single user's attempt to complete a quiz.
type Attempt struct {
	ID                       uint
	UserID                   uint
	QuizID                   uint
	Score                    int
	IsCompleted              bool
	CurrentQuestionIndex     int
	CurrentQuestionMessageID int
	Questions                []Question
	Type                     QuizType
	CourseID                 uint // May be 0 for review quizzes
	UserAnswers              map[uint]uint
}

// Result holds the final outcome of a completed quiz attempt.
type Result struct {
	Score                int
	TotalQuestions       int
	Passed               bool
	ShouldResetProgress  bool // For course block quizzes
	SuggestedNewProgress uint // For course block quizzes
	CourseID             uint // Added field
	TriggerProgress      uint // Added field
}
