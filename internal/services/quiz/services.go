package quiz

import (
	"errors"
	"fmt"
	"github.com/2000ostd/enssi-tel-bot/internal/models"
	"gorm.io/gorm"
	"log"
	"math/rand"
	"strings"
	"time"
)

// Service implements the QuizService interface.
type Service struct {
	db *gorm.DB
	// No courseService dependency here
}

// NewService creates a new instance of the quiz Service.
func NewService(db *gorm.DB) *Service {
	return &Service{
		db: db,
	}
}

// Ensure Service implements QuizService interface
var _ QuizService = (*Service)(nil)

// --- Private Helper Methods for Quiz Creation ---

func (s *Service) findActiveQuizAttemptForBlockInternal(userID uint, courseID uint, progressAtBlockEnd uint) (*models.QuizAttempt, error) {
	var attempt models.QuizAttempt
	err := s.db.Joins("JOIN quizzes ON quizzes.id = quiz_attempts.quiz_id").
		Where("quiz_attempts.user_id = ? AND quizzes.course_id = ? AND quizzes.trigger_progress = ? AND quiz_attempts.is_completed = ?",
			userID, courseID, progressAtBlockEnd, false).
		Order("quiz_attempts.created_at DESC").
		First(&attempt).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("db error finding active quiz attempt for block: %w", err)
	}
	return &attempt, nil
}

func (s *Service) createQuestionInternal(tx *gorm.DB, quizID uint, courseID uint, currentWordCourseIndex uint, wordPoolCourseIndices []uint) error {
	localRand := rand.New(rand.NewSource(time.Now().UnixNano()))

	var cw models.CourseWord
	if err := tx.Where("course_id = ? AND index = ?", courseID, currentWordCourseIndex).First(&cw).Error; err != nil {
		return fmt.Errorf("createQuestion: failed to fetch course word (courseID: %d, index: %d): %w", courseID, currentWordCourseIndex, err)
	}

	var questionWord models.Word
	if err := tx.First(&questionWord, cw.WordID).Error; err != nil {
		return fmt.Errorf("createQuestion: failed to fetch word (ID: %d) for question: %w", cw.WordID, err)
	}

	var questionWS models.WordSource
	if err := tx.Where("word_id = ?", questionWord.ID).
		Preload("PartsOfSpeeches.Meanings").
		First(&questionWS).Error; err != nil {
		return fmt.Errorf("createQuestion: failed to fetch word source for word ID %d: %w", questionWord.ID, err)
	}

	var allMeanings []string
	for _, pos := range questionWS.PartsOfSpeeches {
		for _, m := range pos.Meanings {
			if strings.TrimSpace(m.Title) != "" {
				allMeanings = append(allMeanings, m.Title)
			}
		}
	}
	if len(allMeanings) == 0 {
		if strings.TrimSpace(questionWS.DefPrimary) != "" {
			allMeanings = append(allMeanings, questionWS.DefPrimary)
		} else {
			return fmt.Errorf("word (ID: %d, title: %s) has no usable meanings for question", questionWord.ID, questionWord.Title)
		}
	}
	correctMeaningText := allMeanings[localRand.Intn(len(allMeanings))]

	newQuestion := models.QuizQuestion{QuizID: quizID, Text: questionWord.Title}
	if err := tx.Create(&newQuestion).Error; err != nil {
		return fmt.Errorf("failed to create quiz question entry: %w", err)
	}

	correctOption := models.QuizQuestionOption{QuizQuestionID: newQuestion.ID, Text: correctMeaningText, IsCorrect: true}
	if err := tx.Create(&correctOption).Error; err != nil {
		return fmt.Errorf("failed to create correct quiz option: %w", err)
	}

	numDistractors := 3
	var potentialDistractorMeanings []string
	for _, poolWordIdx := range wordPoolCourseIndices {
		if poolWordIdx == currentWordCourseIndex {
			continue
		}
		var poolCW models.CourseWord
		if err := tx.Where("course_id = ? AND index = ?", courseID, poolWordIdx).First(&poolCW).Error; err == nil {
			var poolWord models.Word
			if err := tx.First(&poolWord, poolCW.WordID).Error; err == nil {
				var poolWS models.WordSource
				if err := tx.Where("word_id = ?", poolWord.ID).Preload("PartsOfSpeeches.Meanings").First(&poolWS).Error; err == nil {
					for _, pos := range poolWS.PartsOfSpeeches {
						for _, m := range pos.Meanings {
							trimmed := strings.TrimSpace(m.Title)
							if trimmed != "" && trimmed != correctMeaningText {
								potentialDistractorMeanings = append(potentialDistractorMeanings, trimmed)
							}
						}
					}
				}
			}
		}
	}

	localRand.Shuffle(len(potentialDistractorMeanings), func(i, j int) {
		potentialDistractorMeanings[i], potentialDistractorMeanings[j] = potentialDistractorMeanings[j], potentialDistractorMeanings[i]
	})

	seenDistractors := make(map[string]bool)
	seenDistractors[correctMeaningText] = true
	distractorsCreated := 0
	for _, meaning := range potentialDistractorMeanings {
		if distractorsCreated >= numDistractors {
			break
		}
		if !seenDistractors[meaning] {
			distractorOption := models.QuizQuestionOption{QuizQuestionID: newQuestion.ID, Text: meaning, IsCorrect: false}
			if err := tx.Create(&distractorOption).Error; err != nil {
				log.Printf("Warning: failed to create distractor option '%s': %v", meaning, err)
			} else {
				distractorsCreated++
				seenDistractors[meaning] = true
			}
		}
	}
	if distractorsCreated < numDistractors {
		log.Printf("Warning: created only %d/%d distractors for question '%s' (QID %d)", distractorsCreated, numDistractors, questionWord.Title, newQuestion.ID)
	}
	return nil
}

func (s *Service) createQuizStructureInternal(tx *gorm.DB, courseID uint, userProgressAtTrigger uint) (*models.Quiz, error) {
	questionCountTarget := quizDefaultQuestionCount
	lastWordIndexInBlock := userProgressAtTrigger
	firstWordIndexInBlock := lastWordIndexInBlock - uint(wordsPerQuizBlock) + 1
	if firstWordIndexInBlock <= 0 {
		firstWordIndexInBlock = 1
	}

	var courseWordsInBlock []models.CourseWord
	err := tx.Where("course_id = ? AND index >= ? AND index <= ?", courseID, firstWordIndexInBlock, lastWordIndexInBlock).
		Order("index ASC").Find(&courseWordsInBlock).Error
	if err != nil {
		return nil, fmt.Errorf("failed to fetch course words for quiz block: %w", err)
	}

	if len(courseWordsInBlock) == 0 {
		return nil, fmt.Errorf("%w: course %d, progress %d to %d", ErrNoWordsForQuiz, courseID, firstWordIndexInBlock, lastWordIndexInBlock)
	}

	numQuestions := len(courseWordsInBlock)
	if numQuestions > questionCountTarget {
		numQuestions = questionCountTarget
	}

	// Ensure we only use available words for question selection
	if numQuestions > len(courseWordsInBlock) {
		numQuestions = len(courseWordsInBlock)
	}
	if numQuestions == 0 { // Should be caught by len(courseWordsInBlock) == 0 earlier
		return nil, fmt.Errorf("%w: no words available to form questions for course %d, block ending at %d", ErrNoWordsForQuiz, courseID, userProgressAtTrigger)
	}

	rand.New(rand.NewSource(time.Now().UnixNano())).Shuffle(len(courseWordsInBlock), func(i, j int) {
		courseWordsInBlock[i], courseWordsInBlock[j] = courseWordsInBlock[j], courseWordsInBlock[i]
	})

	questionWordIndices := make([]uint, 0, numQuestions)
	for i := 0; i < numQuestions; i++ { // No need for && i < len(courseWordsInBlock) due to earlier check
		questionWordIndices = append(questionWordIndices, courseWordsInBlock[i].Index)
	}

	distractorPoolIndices := make([]uint, len(courseWordsInBlock))
	for i, cw := range courseWordsInBlock {
		distractorPoolIndices[i] = cw.Index
	}

	newQuiz := models.Quiz{
		CourseID:        courseID,
		Type:            "multi-option",
		QuestionCount:   uint(numQuestions),
		TriggerProgress: userProgressAtTrigger,
	}
	if err := tx.Create(&newQuiz).Error; err != nil {
		return nil, fmt.Errorf("failed to create quiz DB entry: %w", err)
	}

	questionsCreatedSuccessfully := 0
	for _, wordIdx := range questionWordIndices {
		if err := s.createQuestionInternal(tx, newQuiz.ID, courseID, wordIdx, distractorPoolIndices); err != nil {
			log.Printf("Error creating question for course word index %d (QuizID: %d): %v. Skipping.", wordIdx, newQuiz.ID, err)
		} else {
			questionsCreatedSuccessfully++
		}
	}

	if questionsCreatedSuccessfully == 0 {
		return nil, fmt.Errorf("%w: no questions could be generated for quiz", ErrQuizCreation)
	}

	if uint(questionsCreatedSuccessfully) != newQuiz.QuestionCount {
		newQuiz.QuestionCount = uint(questionsCreatedSuccessfully)
		if err := tx.Model(&newQuiz).Update("question_count", newQuiz.QuestionCount).Error; err != nil {
			log.Printf("Warning: Failed to update actual question count for QuizID %d: %v", newQuiz.ID, err)
		}
	}
	return &newQuiz, nil
}

func (s *Service) createQuizAndAttemptInternal(tx *gorm.DB, userID uint, courseID uint, userProgressAtTrigger uint) (*models.QuizAttempt, error) {
	quiz, err := s.createQuizStructureInternal(tx, courseID, userProgressAtTrigger)
	if err != nil {
		return nil, fmt.Errorf("creating quiz structure: %w", err)
	}
	if quiz == nil || quiz.QuestionCount == 0 {
		return nil, fmt.Errorf("%w: quiz structure is empty or nil after creation attempt", ErrQuizCreation)
	}

	quizAttempt := models.QuizAttempt{
		QuizID:             quiz.ID,
		UserID:             userID,
		Score:              0,
		IsCompleted:        false,
		CurrentQuestionNum: 0,
	}
	if err := tx.Create(&quizAttempt).Error; err != nil {
		return nil, fmt.Errorf("failed to create quiz attempt: %w", err)
	}
	return &quizAttempt, nil
}

// --- Public Service Methods ---

func (s *Service) StartOrResumeQuiz(userID uint, courseID uint, userCourseProgress uint) (*QuizState, error) {
	log.Printf("QuizService: StartOrResumeQuiz called for UserID: %d, CourseID: %d, ProgressTrigger: %d", userID, courseID, userCourseProgress)

	var currentAttempt *models.QuizAttempt
	var messageToUser string
	isNewAttempt := false // To help construct initial message

	// Transaction for finding and potentially creating quiz attempt
	err := s.db.Transaction(func(tx *gorm.DB) error {
		activeAttempt, findErr := s.findActiveQuizAttemptForBlockInternal(userID, courseID, userCourseProgress)
		if findErr != nil { // findActiveQuizAttemptForBlockInternal returns nil, nil if not found
			return fmt.Errorf("checking for active quiz: %w", findErr)
		}

		if activeAttempt != nil {
			log.Printf("QuizService: Resuming active quiz attempt %d.", activeAttempt.ID)
			currentAttempt = activeAttempt
			messageToUser = msgResumeActiveQuiz
		} else {
			log.Printf("QuizService: No active quiz. Creating new one for progress trigger %d.", userCourseProgress)
			newAttempt, createErr := s.createQuizAndAttemptInternal(tx, userID, courseID, userCourseProgress)
			if createErr != nil {
				return fmt.Errorf("%w: %w", ErrQuizCreation, createErr) // Wrap with a specific error
			}
			currentAttempt = newAttempt
			messageToUser = fmt.Sprintf(msgQuizTime, userCourseProgress)
			isNewAttempt = true
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	if currentAttempt == nil {
		return nil, errors.New("internal error: quiz attempt not initialized after transaction")
	}

	var quizForAttempt models.Quiz
	if errDb := s.db.Preload("QuizQuesetions.Options").First(&quizForAttempt, currentAttempt.QuizID).Error; errDb != nil {
		return nil, fmt.Errorf("%w: loading quiz details for attempt %d: %w", ErrQuizNotFound, currentAttempt.ID, errDb)
	}

	if quizForAttempt.QuestionCount == 0 {
		return &QuizState{
			AttemptID:     currentAttempt.ID,
			QuizID:        currentAttempt.QuizID,
			UserID:        userID,
			CourseID:      courseID,
			IsCompleted:   true,
			MessageToUser: msgQuizNoQuestions,
		}, nil
	}
	if currentAttempt.IsCompleted {
		return &QuizState{
			AttemptID:     currentAttempt.ID,
			QuizID:        currentAttempt.QuizID,
			UserID:        userID,
			CourseID:      courseID,
			IsCompleted:   true,
			MessageToUser: msgQuizAlreadyCompleted,
		}, nil
	}
	if currentAttempt.CurrentQuestionNum < 0 || currentAttempt.CurrentQuestionNum >= int(quizForAttempt.QuestionCount) {
		log.Printf("QuizService: Invalid CurrentQuestionNum %d for attempt %d. Signalling for finalization.", currentAttempt.CurrentQuestionNum, currentAttempt.ID)
		return &QuizState{
			AttemptID:                currentAttempt.ID,
			QuizID:                   currentAttempt.QuizID,
			UserID:                   userID,
			CourseID:                 courseID,
			IsCompleted:              true,
			MessageToUser:            msgInvalidQuestionNum, // Handler will fetch full results
			CurrentQuestionMessageID: currentAttempt.CurrentQuestionMessageID,
		}, nil
	}

	currentQuestionModel := quizForAttempt.QuizQuesetions[currentAttempt.CurrentQuestionNum]
	shuffledOptions := make([]models.QuizQuestionOption, len(currentQuestionModel.Options))
	copy(shuffledOptions, currentQuestionModel.Options)
	rand.New(rand.NewSource(time.Now().UnixNano())).Shuffle(len(shuffledOptions), func(i, j int) {
		shuffledOptions[i], shuffledOptions[j] = shuffledOptions[j], shuffledOptions[i]
	})

	questionText := fmt.Sprintf("سوال %d از %d:\n\n%s", currentAttempt.CurrentQuestionNum+1, quizForAttempt.QuestionCount, currentQuestionModel.Text)
	if isNewAttempt && messageToUser == "" { // Ensure messageToUser has a value if new attempt
		messageToUser = fmt.Sprintf(msgStartNewQuizDefault, userCourseProgress)
	} else if messageToUser == "" { // Resuming, but no specific message was set (should not happen)
		messageToUser = msgResumeQuizDefault
	}

	return &QuizState{
		AttemptID:                currentAttempt.ID,
		QuizID:                   currentAttempt.QuizID,
		UserID:                   userID,
		CourseID:                 quizForAttempt.CourseID,
		IsCompleted:              false,
		CurrentQuestionNum:       currentAttempt.CurrentQuestionNum,
		TotalQuestionsInQuiz:     quizForAttempt.QuestionCount,
		QuestionModelID:          currentQuestionModel.ID,
		QuestionText:             questionText,
		Options:                  shuffledOptions,
		MessageToUser:            messageToUser,
		CurrentQuestionMessageID: currentAttempt.CurrentQuestionMessageID,
	}, nil
}

func (s *Service) SubmitAnswer(attemptID uint, chosenOptionID uint, userID uint) (*AnswerSubmissionResult, error) {
	log.Printf("QuizService: SubmitAnswer called for AttemptID: %d, OptionID: %d, UserID: %d", attemptID, chosenOptionID, userID)

	var submissionResult *AnswerSubmissionResult
	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		var attempt models.QuizAttempt
		if err := tx.First(&attempt, attemptID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrAttemptNotFound
			}
			return fmt.Errorf("fetching attempt %d: %w", attemptID, err)
		}
		if attempt.UserID != userID {
			return ErrUserMismatch
		}

		var quizForAttempt models.Quiz
		if err := tx.
			Preload("QuizQuesetions.Options").
			First(&quizForAttempt, attempt.QuizID).
			Error; err != nil {
			return fmt.Errorf("loading quiz %d for attempt %d: %w", attempt.QuizID, attempt.ID, err)
		}

		if attempt.IsCompleted {
			finalResults, err := s.getQuizResultsInternal(tx, &attempt, &quizForAttempt)
			if err != nil {
				return fmt.Errorf("getting results for completed attempt %d: %w", attemptID, err)
			}
			submissionResult = &AnswerSubmissionResult{
				AttemptID: attempt.ID, QuizID: attempt.QuizID, UserID: userID, CourseID: quizForAttempt.CourseID,
				IsQuizNowCompleted: true, FinalQuizResult: finalResults, CurrentQuestionMessageID: attempt.CurrentQuestionMessageID,
			}
			return nil
		}

		var chosenOption models.QuizQuestionOption
		if err := tx.First(&chosenOption, chosenOptionID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrInvalidOption
			}
			return fmt.Errorf("fetching option %d: %w", chosenOptionID, err)
		}

		if attempt.CurrentQuestionNum < 0 || attempt.CurrentQuestionNum >= int(quizForAttempt.QuestionCount) {
			finalResults, err := s.getQuizResultsInternal(tx, &attempt, &quizForAttempt)
			if err != nil {
				return fmt.Errorf("finalizing attempt %d with invalid q_num: %w", attemptID, err)
			}
			submissionResult = &AnswerSubmissionResult{
				AttemptID: attempt.ID, QuizID: attempt.QuizID, UserID: userID, CourseID: quizForAttempt.CourseID,
				IsQuizNowCompleted: true, FinalQuizResult: finalResults, CurrentQuestionMessageID: attempt.CurrentQuestionMessageID,
			}
			return nil
		}
		expectedQuestionID := quizForAttempt.QuizQuesetions[attempt.CurrentQuestionNum].ID
		if chosenOption.QuizQuestionID != expectedQuestionID {
			log.Printf("Option mismatch for attempt %d. Expected QID %d, got for QID %d", attemptID, expectedQuestionID, chosenOption.QuizQuestionID)
			return fmt.Errorf("%w: option chosen for question %d, but current is implicitly %d", ErrInvalidOption, chosenOption.QuizQuestionID, expectedQuestionID)
		}

		quizAnswer := models.QuizAnswer{
			QuizAttemptID:        attempt.ID,
			QuizQuestionID:       chosenOption.QuizQuestionID,
			QuizQuestionOptionID: chosenOption.ID,
			IsCorrect:            chosenOption.IsCorrect,
		}
		if err := tx.Create(&quizAnswer).Error; err != nil {
			return fmt.Errorf("saving quiz answer for attempt %d: %w", attempt.ID, err)
		}

		attempt.CurrentQuestionNum++
		if err := tx.Model(&attempt).Update("current_question_num", attempt.CurrentQuestionNum).Error; err != nil { // Only update one field
			return fmt.Errorf("updating CurrentQuestionNum for attempt %d: %w", attempt.ID, err)
		}

		if attempt.CurrentQuestionNum >= int(quizForAttempt.QuestionCount) {
			finalResults, err := s.getQuizResultsInternal(tx, &attempt, &quizForAttempt)
			if err != nil {
				return fmt.Errorf("finalizing quiz attempt %d: %w", attemptID, err)
			}
			submissionResult = &AnswerSubmissionResult{
				AttemptID:                attempt.ID,
				QuizID:                   attempt.QuizID,
				UserID:                   userID,
				CourseID:                 quizForAttempt.CourseID,
				IsQuizNowCompleted:       true,
				FinalQuizResult:          finalResults,
				CurrentQuestionMessageID: attempt.CurrentQuestionMessageID,
			}
		} else {
			currentQuestionModel := quizForAttempt.QuizQuesetions[attempt.CurrentQuestionNum]
			shuffledOptions := make([]models.QuizQuestionOption, len(currentQuestionModel.Options))
			copy(shuffledOptions, currentQuestionModel.Options)
			rand.New(rand.NewSource(time.Now().UnixNano())).Shuffle(len(shuffledOptions), func(i, j int) {
				shuffledOptions[i], shuffledOptions[j] = shuffledOptions[j], shuffledOptions[i]
			})
			questionText := fmt.Sprintf("سوال %d از %d:\n\n%s", attempt.CurrentQuestionNum+1, quizForAttempt.QuestionCount, currentQuestionModel.Text)

			submissionResult = &AnswerSubmissionResult{
				AttemptID:          attempt.ID,
				QuizID:             attempt.QuizID,
				UserID:             userID,
				CourseID:           quizForAttempt.CourseID,
				IsQuizNowCompleted: false,
				NextQuestionState: &QuizState{
					AttemptID:                attempt.ID,
					QuizID:                   attempt.QuizID,
					UserID:                   userID,
					CourseID:                 quizForAttempt.CourseID,
					IsCompleted:              false,
					CurrentQuestionNum:       attempt.CurrentQuestionNum,
					TotalQuestionsInQuiz:     quizForAttempt.QuestionCount,
					QuestionModelID:          currentQuestionModel.ID,
					QuestionText:             questionText,
					Options:                  shuffledOptions,
					CurrentQuestionMessageID: attempt.CurrentQuestionMessageID,
				},
				CurrentQuestionMessageID: attempt.CurrentQuestionMessageID,
			}
		}
		return nil
	})

	if txErr != nil {
		return nil, fmt.Errorf("%w: %w", ErrAnswerProcessing, txErr)
	}
	return submissionResult, nil
}

// getQuizResultsInternal finalizes an attempt and prepares results.
// It now takes quizInfo as a parameter because SubmitAnswer already loaded it.
func (s *Service) getQuizResultsInternal(tx *gorm.DB, attempt *models.QuizAttempt, quizInfo *models.Quiz) (*QuizResult, error) {
	if !attempt.IsCompleted {
		var quizAnswers []models.QuizAnswer
		// Fetch answers within the transaction
		if err := tx.Where("quiz_attempt_id = ?", attempt.ID).Find(&quizAnswers).Error; err != nil {
			return nil, fmt.Errorf("fetching answers for attempt %d: %w", attempt.ID, err)
		}
		correctAnswers := 0
		for _, ans := range quizAnswers {
			if ans.IsCorrect {
				correctAnswers++
			}
		}
		attempt.Score = correctAnswers
		attempt.IsCompleted = true
		updates := map[string]interface{}{"score": attempt.Score, "is_completed": true}
		if err := tx.Model(attempt).Updates(updates).Error; err != nil {
			return nil, fmt.Errorf("saving final attempt details for attempt %d: %w", attempt.ID, err)
		}
	}

	totalQuestions := int(quizInfo.QuestionCount)
	var reviewBuilder strings.Builder
	reviewBuilder.WriteString(msgReviewHeader)

	// Fetch answers and options for review. Ensure QuizQuestions are preloaded on quizInfo.
	// For brevity, assuming quizInfo.QuizQuesetions.Options are loaded.
	var answersForReview []models.QuizAnswer // Re-fetch or ensure they are on 'attempt' struct if preloaded by caller
	tx.Where("quiz_attempt_id = ?", attempt.ID).Find(&answersForReview)

	userAnswersMap := make(map[uint]models.QuizAnswer)
	var allChosenOptionIDs []uint
	for _, ans := range answersForReview {
		userAnswersMap[ans.QuizQuestionID] = ans
		allChosenOptionIDs = append(allChosenOptionIDs, ans.QuizQuestionOptionID)
	}
	chosenOptionsTextMap := make(map[uint]string)
	if len(allChosenOptionIDs) > 0 {
		var fetchedChosenOptions []models.QuizQuestionOption
		err := tx.Where("id IN ?", allChosenOptionIDs).Find(&fetchedChosenOptions).Error
		if err == nil {
			for _, opt := range fetchedChosenOptions {
				chosenOptionsTextMap[opt.ID] = opt.Text
			}
		}

	}

	for i, q := range quizInfo.QuizQuesetions {
		reviewBuilder.WriteString(fmt.Sprintf(msgReviewQuestion, i+1, q.Text))
		userChosenText := msgReviewNotAnswered
		userCorrectStatus := msgIncorrectMarker
		if userAnswer, found := userAnswersMap[q.ID]; found {
			if text, ok := chosenOptionsTextMap[userAnswer.QuizQuestionOptionID]; ok {
				userChosenText = text
			}
			if userAnswer.IsCorrect {
				userCorrectStatus = msgCorrectMarker
			}
		}
		reviewBuilder.WriteString(fmt.Sprintf(msgReviewUserAnswer, userChosenText, userCorrectStatus))
		var correctOptTexts []string
		for _, opt := range q.Options {
			if opt.IsCorrect {
				correctOptTexts = append(correctOptTexts, opt.Text)
			}
		}
		if len(correctOptTexts) > 0 {
			reviewBuilder.WriteString(fmt.Sprintf(msgReviewCorrectAnswer, strings.Join(correctOptTexts, " / ")))
		}
	}
	reviewText := reviewBuilder.String()

	passed := attempt.Score >= quizPassThresholdCorrectAnswers
	resultMessage := fmt.Sprintf(msgQuizFinished, attempt.Score, totalQuestions)

	shouldResetProgress := false
	suggestedNewProgress := uint(0)

	if passed {
		resultMessage += msgQuizPassed
	} else {
		resultMessage += fmt.Sprintf(msgQuizFailed, quizPassThresholdCorrectAnswers)
		shouldResetProgress = true
		suggestedNewProgress = quizInfo.TriggerProgress - uint(wordsPerQuizBlock) + 1
		if quizInfo.TriggerProgress < uint(wordsPerQuizBlock) {
			suggestedNewProgress = 1
		}
		if suggestedNewProgress <= 0 {
			suggestedNewProgress = 1
		}
		// Message about progress reset will be handled by CourseService/handler
	}

	return &QuizResult{
		AttemptID: attempt.ID, QuizID: quizInfo.ID, UserID: attempt.UserID, CourseID: quizInfo.CourseID,
		Score: attempt.Score, TotalQuestions: totalQuestions, Passed: passed, ReviewText: reviewText, ResultMessage: resultMessage,
		ShouldResetProgress:  shouldResetProgress,
		SuggestedNewProgress: suggestedNewProgress,
	}, nil
}

func (s *Service) GetQuizResults(attemptID uint, userID uint) (*QuizResult, error) {
	log.Printf("QuizService: GetQuizResults called for AttemptID: %d, UserID: %d", attemptID, userID)
	var finalResults *QuizResult
	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		var attempt models.QuizAttempt
		if err := tx.First(&attempt, attemptID).Error; err != nil { // Removed Preload("QuizAnswers")
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrAttemptNotFound
			}
			return fmt.Errorf("fetching attempt %d: %w", attemptID, err)
		}
		if attempt.UserID != userID {
			return ErrUserMismatch
		}

		var quizInfo models.Quiz // Load quizInfo needed by getQuizResultsInternal
		if err := tx.Preload("QuizQuesetions.Options").First(&quizInfo, attempt.QuizID).Error; err != nil {
			return fmt.Errorf("loading quiz info %d for attempt %d: %w", attempt.QuizID, attempt.ID, err)
		}

		var err error
		finalResults, err = s.getQuizResultsInternal(tx, &attempt, &quizInfo)
		return err
	})
	if txErr != nil {
		return nil, fmt.Errorf("%w: %w", ErrFinalization, txErr)
	}
	return finalResults, nil
}

func (s *Service) FindAnyActiveQuizAttempt(userID uint) (*models.QuizAttempt, error) {
	log.Printf("QuizService: FindAnyActiveQuizAttempt called for UserID: %d", userID)
	var attempt models.QuizAttempt
	err := s.db.Where("user_id = ? AND is_completed = ?", userID, false).
		Order("created_at DESC").First(&attempt).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("db error finding active quiz: %w", err)
	}
	return &attempt, nil
}

func (s *Service) UpdateQuizAttemptMessageID(attemptID uint, messageID int) error {
	log.Printf("QuizService: UpdateQuizAttemptMessageID called for AttemptID: %d, MessageID: %d", attemptID, messageID)
	if attemptID == 0 {
		return fmt.Errorf("%w: attemptID cannot be zero", ErrInvalidInput)
	}
	result := s.db.Model(&models.QuizAttempt{}).Where("id = ?", attemptID).Update("current_question_message_id", messageID)
	if result.Error != nil {
		return fmt.Errorf("failed to update message_id for attempt %d: %w", attemptID, result.Error)
	}
	if result.RowsAffected == 0 {
		log.Printf("QuizService: UpdateQuizAttemptMessageID - No rows affected for attempt %d.", attemptID)
	}
	return nil
}

