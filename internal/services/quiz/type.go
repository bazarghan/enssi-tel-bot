package quiz

import (
	"errors"
	"github.com/2000ostd/enssi-tel-bot/internal/models"
)

// --- Service-Specific Errors ---
var (
	ErrQuizNotFound            = errors.New("quiz service: quiz not found")
	ErrAttemptNotFound         = errors.New("quiz service: quiz attempt not found")
	ErrAttemptAlreadyCompleted = errors.New("quiz service: quiz attempt already completed")
	ErrUserMismatch            = errors.New("quiz service: user does not own this quiz attempt")
	ErrInvalidOption           = errors.New("quiz service: invalid option selected")
	ErrNoWordsForQuiz          = errors.New("quiz service: not enough words available to create a quiz for this block")
	ErrQuizCreation            = errors.New("quiz service: failed to create quiz structure or questions")
	ErrQuestionLoad            = errors.New("quiz service: failed to load quiz question")
	ErrAnswerProcessing        = errors.New("quiz service: failed to process answer")
	ErrFinalization            = errors.New("quiz service: failed to finalize quiz attempt")
	ErrCourseInteraction       = errors.New("quiz service: failed to interact with course service for progress update")
	ErrInvalidInput            = errors.New("quiz service: invalid input provided")
)

// --- Quiz Specific Constants ---

const (
	// These should ideally come from a configuration shared or app config.
	quizPassThresholdCorrectAnswers = 9
	wordsPerQuizBlock               = 12 // Used to calculate reset progress
	quizDefaultQuestionCount        = 12

	// User Messages
	msgResumeActiveQuiz     = "شما یک آزمون نیمه‌تمام برای این بخش دارید. ادامه می‌دهیم..."
	msgQuizTime             = "🎉 زمان آزمون! (برای کلمات تا شماره %d)"
	msgQuizNoQuestions      = "این آزمون سوالی ندارد. لطفا به ادمین اطلاع دهید."
	msgQuizAlreadyCompleted = "این آزمون قبلا تکمیل شده است."
	msgInvalidQuestionNum   = "خطا در شماره سوال، آزمون پایان می‌یابد."
	msgStartNewQuizDefault  = "🎉 شروع آزمون جدید! (برای کلمات تا %d)"
	msgResumeQuizDefault    = "ادامه آزمون..."
	msgQuizPassed           = "\n🎉 تبریک! شما آزمون را با موفقیت گذراندید."
	msgQuizFailed           = "\n😔 متاسفانه حد نصاب قبولی (%d پاسخ صحیح) را کسب نکردید."
	msgQuizFinished         = "🏁 آزمون تمام شد!\n\nشما به %d سوال از %d سوال پاسخ صحیح دادید."
	msgReviewHeader         = "\n\n📝 مرور سوالات:\n"
	msgReviewQuestion       = "\n%d. سوال: %s\n"
	msgReviewUserAnswer     = "   شما پاسخ دادید: %s (%s)\n"
	msgReviewCorrectAnswer  = "   پاسخ صحیح: %s\n"
	msgReviewNotAnswered    = "پاسخ نداده"
	msgCorrectMarker        = "✅"
	msgIncorrectMarker      = "❌"
)

// --- Request/Response Structs for Service Methods ---

// QuizState represents the current state of a quiz to be presented to the user.
type QuizState struct {
	AttemptID                uint
	QuizID                   uint
	UserID                   uint
	CourseID                 uint
	IsCompleted              bool // True if this call determined the quiz is already complete
	CurrentQuestionNum       int  // 0-indexed
	TotalQuestionsInQuiz     uint
	QuestionModelID          uint // The ID of the models.QuizQuestion
	QuestionText             string
	Options                  []models.QuizQuestionOption // Shuffled options for the current question
	MessageToUser            string                      // Initial message like "Starting quiz..." or "Resuming quiz..."
	CurrentQuestionMessageID int                         // Telegram message ID of the current question, if any
}

// AnswerSubmissionResult represents the outcome of submitting an answer.
type AnswerSubmissionResult struct {
	AttemptID                uint
	QuizID                   uint
	UserID                   uint
	CourseID                 uint
	IsQuizNowCompleted       bool
	NextQuestionState        *QuizState  // Populated if quiz is not completed
	FinalQuizResult          *QuizResult // Populated if quiz is completed
	CurrentQuestionMessageID int         // The message ID that was just handled (e.g., the one with the buttons)
}

type QuizResult struct {
	AttemptID            uint
	QuizID               uint
	UserID               uint
	CourseID             uint // Course ID is important here
	Score                int
	TotalQuestions       int
	Passed               bool
	ReviewText           string
	ResultMessage        string
	ShouldResetProgress  bool // True if user failed and progress should be reset
	SuggestedNewProgress uint // The progress value the user should be reset to
}
