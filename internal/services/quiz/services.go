package quiz

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/2000ostd/enssi-tel-bot/internal/models"
	"github.com/2000ostd/enssi-tel-bot/internal/services/word" // Added for WordService interaction
	"gorm.io/gorm"
)

// Service implements the QuizService interface.
type Service struct {
	db          *gorm.DB
	wordService word.WordService // Added WordService dependency
}

// NewService creates a new instance of the quiz Service.
func NewService(db *gorm.DB, ws word.WordService) *Service { // Added WordService parameter
	return &Service{
		db:          db,
		wordService: ws,
	}
}

// Ensure Service implements QuizService interface
var _ QuizService = (*Service)(nil)

// --- Private Helper Methods for Quiz Creation ---

// createQuestionInternal generates a question for a given word (from course or review).
// For review quizzes, wordToReview is non-nil. For course quizzes, courseID and wordCourseIndex are used.
func (s *Service) createQuestionInternal(
	tx *gorm.DB,
	quizID uint,
	wordID uint, // The ID of the word this question is about
	questionTextSeed string, // Typically the word's title
	courseID uint, // Optional: 0 if not a course-specific context (e.g. for global distractors)
	distractorSourceIndices []uint, // Optional: indices of words in a course block for distractors
) error {
	localRand := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Fetch the primary word for the question
	var questionWord models.Word
	if err := tx.First(&questionWord, wordID).Error; err != nil {
		return fmt.Errorf("createQuestion: failed to fetch word (ID: %d) for question: %w", wordID, err)
	}

	var questionWS models.WordSource
	if err := tx.
		Where("word_id = ?", questionWord.ID).
		Preload("PartsOfSpeeches.Meanings").
		First(&questionWS).Error; err != nil {
		return fmt.Errorf("createQuestion: failed to fetch word source for word ID %d: %w", questionWord.ID, err)
	}

	var allMeanings []string
	for _, pos := range questionWS.PartsOfSpeeches {
		for _, m := range pos.Meanings {
			// Prefer Persian meanings if available, otherwise English
			if strings.TrimSpace(m.Title) != "" && (m.Lang == "fa" || m.Lang == "en" || m.Lang == "") {
				allMeanings = append(allMeanings, m.Title)
			}
		}
	}
	if len(allMeanings) == 0 {
		if strings.TrimSpace(questionWS.DefPrimary) != "" {
			allMeanings = append(allMeanings, questionWS.DefPrimary)
		} else {
			log.Printf("Warning: Word (ID: %d, title: %s) has no usable meanings for question. Skipping question.", questionWord.ID, questionWord.Title)
			return fmt.Errorf("word (ID: %d, title: %s) has no usable meanings for question", questionWord.ID, questionWord.Title)
		}
	}
	correctMeaningText := allMeanings[localRand.Intn(len(allMeanings))]

	newQuestion := models.QuizQuestion{
		QuizID: quizID,
		Text:   questionTextSeed, // Use the provided seed (e.g., word title)
		WordID: wordID,           // Link question to the word
	}
	if err := tx.Create(&newQuestion).Error; err != nil {
		return fmt.Errorf("failed to create quiz question entry: %w", err)
	}

	correctOption := models.QuizQuestionOption{QuizQuestionID: newQuestion.ID, Text: correctMeaningText, IsCorrect: true}
	if err := tx.Create(&correctOption).Error; err != nil {
		return fmt.Errorf("failed to create correct quiz option: %w", err)
	}

	// --- Distractor Generation ---
	numDistractorsNeeded := reviewQuizMaxOptions - 1 // e.g., 3
	var potentialDistractorMeanings []string

	// Strategy 1: Use distractorSourceIndices if provided (for course quizzes)
	if courseID != 0 && len(distractorSourceIndices) > 0 {
		for _, poolWordIdx := range distractorSourceIndices {
			// Skip if it's the current question's word index (though wordID is the primary check now)
			// This logic might need adjustment if distractorSourceIndices refers to actual WordIDs
			var poolCW models.CourseWord
			if err := tx.Where("course_id = ? AND index = ?", courseID, poolWordIdx).First(&poolCW).Error; err == nil {
				if poolCW.WordID == wordID { // Don't use the same word for distractors
					continue
				}
				var poolWord models.Word
				if err := tx.First(&poolWord, poolCW.WordID).Error; err == nil {
					var poolWS models.WordSource
					if err := tx.Where("word_id = ?", poolWord.ID).Preload("PartsOfSpeeches.Meanings").First(&poolWS).Error; err == nil {
						for _, pos := range poolWS.PartsOfSpeeches {
							for _, m := range pos.Meanings {
								trimmed := strings.TrimSpace(m.Title)
								// Add if it's a valid meaning and not the correct answer for the current question
								if trimmed != "" && trimmed != correctMeaningText {
									potentialDistractorMeanings = append(potentialDistractorMeanings, trimmed)
								}
							}
						}
					}
				}
			}
		}
	}

	// Strategy 2: If not enough distractors, fetch random words from DB (primarily for review quizzes)
	if len(potentialDistractorMeanings) < numDistractorsNeeded {
		var randomWords []models.Word
		// Fetch more words than needed to increase chances of getting unique meanings
		// Exclude the current question's word ID
		err := tx.Not("id = ?", wordID).Order(gorm.Expr("RANDOM()")).Limit(numDistractorsNeeded * 2).Find(&randomWords).Error
		if err == nil {
			for _, rWord := range randomWords {
				var rWS models.WordSource
				if err := tx.Where("word_id = ?", rWord.ID).Preload("PartsOfSpeeches.Meanings").First(&rWS).Error; err == nil {
					for _, pos := range rWS.PartsOfSpeeches {
						for _, m := range pos.Meanings {
							trimmed := strings.TrimSpace(m.Title)
							if trimmed != "" && trimmed != correctMeaningText {
								potentialDistractorMeanings = append(potentialDistractorMeanings, trimmed)
							}
						}
					}
				}
			}
		} else {
			log.Printf("Warning: Failed to fetch random words for distractors: %v", err)
		}
	}

	localRand.Shuffle(len(potentialDistractorMeanings), func(i, j int) {
		potentialDistractorMeanings[i], potentialDistractorMeanings[j] = potentialDistractorMeanings[j], potentialDistractorMeanings[i]
	})

	seenDistractors := make(map[string]bool)
	seenDistractors[correctMeaningText] = true // Don't pick the correct answer as a distractor
	distractorsCreated := 0
	for _, meaning := range potentialDistractorMeanings {
		if distractorsCreated >= numDistractorsNeeded {
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

	if distractorsCreated < numDistractorsNeeded {
		log.Printf("Warning: created only %d/%d distractors for question '%s' (WordID %d, QID %d)", distractorsCreated, numDistractorsNeeded, questionTextSeed, wordID, newQuestion.ID)
		// Optionally, fill with generic distractors if absolutely needed, e.g., "None of the above" - but this is usually not great.
	}
	return nil
}

// createCourseBlockQuizStructureInternal creates the quiz structure for a course block.
func (s *Service) createCourseBlockQuizStructureInternal(tx *gorm.DB, courseID uint, userProgressAtTrigger uint) (*models.Quiz, error) {
	questionCountTarget := quizDefaultQuestionCount
	lastWordIndexInBlock := userProgressAtTrigger
	firstWordIndexInBlock := uint(1) // Default to 1
	if lastWordIndexInBlock >= wordsPerQuizBlock {
		firstWordIndexInBlock = lastWordIndexInBlock - wordsPerQuizBlock + 1
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

	numQuestionsToGenerate := len(courseWordsInBlock)
	if numQuestionsToGenerate > questionCountTarget {
		numQuestionsToGenerate = questionCountTarget
	}
	if numQuestionsToGenerate == 0 {
		return nil, fmt.Errorf("%w: no words available to form questions for course %d, block ending at %d", ErrNoWordsForQuiz, courseID, userProgressAtTrigger)
	}

	// Shuffle course words to pick from for questions
	localRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	localRand.Shuffle(len(courseWordsInBlock), func(i, j int) {
		courseWordsInBlock[i], courseWordsInBlock[j] = courseWordsInBlock[j], courseWordsInBlock[i]
	})

	// Words to be used for questions
	questionWordsFromBlock := courseWordsInBlock[:numQuestionsToGenerate]
	// Indices of all words in the block, for distractor pool
	distractorPoolIndices := make([]uint, len(courseWordsInBlock))
	for i, cw := range courseWordsInBlock {
		distractorPoolIndices[i] = cw.Index
	}

	newQuiz := models.Quiz{
		CourseID:        courseID,
		Type:            models.QuizTypeCourseBlock,
		QuestionCount:   uint(numQuestionsToGenerate), // Tentative, will be updated
		TriggerProgress: userProgressAtTrigger,
	}
	if err := tx.Create(&newQuiz).Error; err != nil {
		return nil, fmt.Errorf("failed to create quiz DB entry for course block: %w", err)
	}

	questionsCreatedSuccessfully := 0
	for _, cw := range questionWordsFromBlock {
		// Fetch the word title to use as question text seed
		var wordForQuestion models.Word
		if err := tx.First(&wordForQuestion, cw.WordID).Error; err != nil {
			log.Printf("Error fetching word (ID %d) for question text seed: %v. Skipping question.", cw.WordID, err)
			continue
		}

		if err := s.createQuestionInternal(tx, newQuiz.ID, cw.WordID, wordForQuestion.Title, courseID, distractorPoolIndices); err != nil {
			log.Printf("Error creating course block question for CourseWord (Index %d, WordID %d, QuizID: %d): %v. Skipping.", cw.Index, cw.WordID, newQuiz.ID, err)
		} else {
			questionsCreatedSuccessfully++
		}
	}

	if questionsCreatedSuccessfully == 0 {
		// Attempt to delete the quiz if no questions were made? Or leave it empty?
		// For now, returning an error is safer.
		tx.Delete(&newQuiz) // Rollback will handle this if transaction fails, but explicit delete if this path is taken.
		return nil, fmt.Errorf("%w: no questions could be generated for course block quiz", ErrQuizCreation)
	}

	if uint(questionsCreatedSuccessfully) != newQuiz.QuestionCount {
		newQuiz.QuestionCount = uint(questionsCreatedSuccessfully)
		if err := tx.Model(&newQuiz).Update("question_count", newQuiz.QuestionCount).Error; err != nil {
			log.Printf("Warning: Failed to update actual question count for CourseBlock QuizID %d: %v", newQuiz.ID, err)
			// Non-fatal, proceed with the questions made.
		}
	}
	return &newQuiz, nil
}

// createQuizAndAttemptInternal is for COURSE_BLOCK quizzes
func (s *Service) createQuizAndAttemptInternal(tx *gorm.DB, userID uint, courseID uint, userProgressAtTrigger uint) (*models.QuizAttempt, *models.Quiz, error) {
	quiz, err := s.createCourseBlockQuizStructureInternal(tx, courseID, userProgressAtTrigger)
	if err != nil {
		return nil, nil, fmt.Errorf("creating course block quiz structure: %w", err)
	}
	if quiz == nil || quiz.QuestionCount == 0 {
		return nil, nil, fmt.Errorf("%w: course block quiz structure is empty or nil after creation attempt", ErrQuizCreation)
	}

	quizAttempt := models.QuizAttempt{
		QuizID:             quiz.ID,
		UserID:             userID,
		Score:              0,
		IsCompleted:        false,
		CurrentQuestionNum: 0,
	}
	if err := tx.Create(&quizAttempt).Error; err != nil {
		return nil, quiz, fmt.Errorf("failed to create quiz attempt for course block quiz: %w", err)
	}
	return &quizAttempt, quiz, nil
}

// --- Public Service Methods ---

func (s *Service) StartOrResumeQuiz(userID uint, courseID uint, userCourseProgress uint) (*QuizState, error) {
	log.Printf("QuizService: StartOrResumeQuiz (COURSE_BLOCK) called for UserID: %d, CourseID: %d, ProgressTrigger: %d", userID, courseID, userCourseProgress)

	var currentAttempt *models.QuizAttempt
	var quizForAttempt models.Quiz // To store the quiz associated with the attempt
	var messageToUser string
	isNewAttempt := false

	err := s.db.Transaction(func(tx *gorm.DB) error {
		var activeAttempt models.QuizAttempt
		// Find active attempt for THIS specific course block quiz trigger
		err := tx.Joins("JOIN quizzes ON quizzes.id = quiz_attempts.quiz_id").
			Where("quiz_attempts.user_id = ? AND quizzes.course_id = ? AND quizzes.trigger_progress = ? AND quiz_attempts.is_completed = ? AND quizzes.type = ?",
				userID, courseID, userCourseProgress, false, models.QuizTypeCourseBlock).
			Order("quiz_attempts.created_at DESC").
			Preload("Quiz"). // Preload the Quiz model
			First(&activeAttempt).Error

		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// No active attempt, create new one
				log.Printf("QuizService: No active COURSE_BLOCK quiz. Creating new one for progress trigger %d.", userCourseProgress)
				newAttempt, newQuiz, createErr := s.createQuizAndAttemptInternal(tx, userID, courseID, userCourseProgress)
				if createErr != nil {
					return fmt.Errorf("%w: %w", ErrQuizCreation, createErr)
				}
				currentAttempt = newAttempt
				quizForAttempt = *newQuiz // Store the newly created quiz
				messageToUser = fmt.Sprintf(msgQuizTime, userCourseProgress)
				isNewAttempt = true
			} else {
				return fmt.Errorf("db error finding active course block quiz attempt: %w", err)
			}
		} else {
			log.Printf("QuizService: Resuming active COURSE_BLOCK quiz attempt %d.", activeAttempt.ID)
			currentAttempt = &activeAttempt
			// Fetch the quiz details if not preloaded correctly or if needed separately
			if err := tx.Preload("QuizQuesetions.Options").First(&quizForAttempt, currentAttempt.QuizID).Error; err != nil {
				return fmt.Errorf("%w: loading quiz details for active attempt %d: %w", ErrQuizNotFound, currentAttempt.ID, err)
			}
			messageToUser = msgResumeActiveQuiz
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	if currentAttempt == nil { // Should be caught by transaction error handling
		return nil, errors.New("internal error: quiz attempt not initialized after transaction for course block quiz")
	}
	if quizForAttempt.ID == 0 { // Ensure quizForAttempt is populated
		if errDb := s.db.Preload("QuizQuesetions.Options").First(&quizForAttempt, currentAttempt.QuizID).Error; errDb != nil {
			return nil, fmt.Errorf("%w: loading quiz details for attempt %d: %w", ErrQuizNotFound, currentAttempt.ID, errDb)
		}
	}

	if quizForAttempt.QuestionCount == 0 {
		// Attempt to mark as complete if it's empty to avoid loops
		s.db.Model(&currentAttempt).Update("is_completed", true)
		return &QuizState{
			AttemptID: currentAttempt.ID, QuizID: currentAttempt.QuizID, UserID: userID, CourseID: courseID, QuizType: models.QuizTypeCourseBlock,
			IsCompleted: true, MessageToUser: msgQuizNoQuestions, CurrentQuestionMessageID: currentAttempt.CurrentQuestionMessageID,
		}, nil
	}
	if currentAttempt.IsCompleted {
		return &QuizState{
			AttemptID: currentAttempt.ID, QuizID: currentAttempt.QuizID, UserID: userID, CourseID: courseID, QuizType: models.QuizTypeCourseBlock,
			IsCompleted: true, MessageToUser: msgQuizAlreadyCompleted, CurrentQuestionMessageID: currentAttempt.CurrentQuestionMessageID,
		}, nil
	}
	if currentAttempt.CurrentQuestionNum < 0 || currentAttempt.CurrentQuestionNum >= int(quizForAttempt.QuestionCount) {
		log.Printf("QuizService: Invalid CurrentQuestionNum %d for COURSE_BLOCK attempt %d. Finalizing.", currentAttempt.CurrentQuestionNum, currentAttempt.ID)
		// This state signals to the handler to fetch full results.
		s.db.Model(&currentAttempt).Updates(map[string]interface{}{"is_completed": true, "score": currentAttempt.Score}) // Ensure it's marked completed
		return &QuizState{
			AttemptID: currentAttempt.ID, QuizID: currentAttempt.QuizID, UserID: userID, CourseID: courseID, QuizType: models.QuizTypeCourseBlock,
			IsCompleted: true, MessageToUser: msgInvalidQuestionNum, CurrentQuestionMessageID: currentAttempt.CurrentQuestionMessageID,
		}, nil
	}

	currentQuestionModel := quizForAttempt.QuizQuesetions[currentAttempt.CurrentQuestionNum]
	shuffledOptions := make([]models.QuizQuestionOption, len(currentQuestionModel.Options))
	copy(shuffledOptions, currentQuestionModel.Options)
	rand.New(rand.NewSource(time.Now().UnixNano())).Shuffle(len(shuffledOptions), func(i, j int) {
		shuffledOptions[i], shuffledOptions[j] = shuffledOptions[j], shuffledOptions[i]
	})

	questionText := fmt.Sprintf("سوال %d از %d:\n\n%s", currentAttempt.CurrentQuestionNum+1, quizForAttempt.QuestionCount, currentQuestionModel.Text)
	if isNewAttempt && messageToUser == "" {
		messageToUser = fmt.Sprintf(msgStartNewQuizDefault, userCourseProgress)
	} else if messageToUser == "" {
		messageToUser = msgResumeQuizDefault
	}

	return &QuizState{
		AttemptID:                currentAttempt.ID,
		QuizID:                   currentAttempt.QuizID,
		UserID:                   userID,
		CourseID:                 quizForAttempt.CourseID, // From the quiz model
		QuizType:                 models.QuizTypeCourseBlock,
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

func (s *Service) CreateReviewQuiz(userID uint, wordsToReview []word.WordStudiedView) (*QuizState, error) {
	log.Printf("QuizService: CreateReviewQuiz called for UserID: %d with %d words.", userID, len(wordsToReview))
	if len(wordsToReview) == 0 {
		return &QuizState{UserID: userID, IsCompleted: true, MessageToUser: "کلمه‌ای برای مرور وجود ندارد.", QuizType: models.QuizTypeReview}, nil
	}

	var newQuiz models.Quiz
	var currentAttempt models.QuizAttempt
	var messageToUser string = msgStartNewReviewQuiz

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Create the Quiz model entry
		newQuiz = models.Quiz{
			Type:          models.QuizTypeReview,
			QuestionCount: uint(len(wordsToReview)), // Tentative
			// CourseID is not set for general review quizzes
		}
		if err := tx.Create(&newQuiz).Error; err != nil {
			return fmt.Errorf("failed to create REVIEW Quiz DB entry: %w", err)
		}

		questionsCreatedSuccessfully := 0
		for _, wordView := range wordsToReview {
			// For review quizzes, distractorSourceIndices is nil/empty as distractors are fetched globally/randomly.
			// CourseID is also 0 for createQuestionInternal in this context if distractors are global.
			if err := s.createQuestionInternal(tx, newQuiz.ID, wordView.WordID, wordView.WordTitle, 0, nil); err != nil {
				log.Printf("Error creating REVIEW question for WordID %d (QuizID: %d): %v. Skipping.", wordView.WordID, newQuiz.ID, err)
			} else {
				questionsCreatedSuccessfully++
			}
		}

		if questionsCreatedSuccessfully == 0 {
			tx.Delete(&newQuiz) // Clean up quiz if no questions
			return fmt.Errorf("%w: no questions could be generated for review quiz", ErrQuizCreation)
		}
		if uint(questionsCreatedSuccessfully) != newQuiz.QuestionCount {
			newQuiz.QuestionCount = uint(questionsCreatedSuccessfully)
			if err := tx.Model(&newQuiz).Update("question_count", newQuiz.QuestionCount).Error; err != nil {
				log.Printf("Warning: Failed to update actual question count for REVIEW QuizID %d: %v", newQuiz.ID, err)
			}
		}

		// Reload newQuiz to get all its associations, especially QuizQuestions
		if err := tx.Preload("QuizQuesetions.Options").First(&newQuiz, newQuiz.ID).Error; err != nil {
			return fmt.Errorf("failed to reload review quiz with questions: %w", err)
		}

		// Create the QuizAttempt
		currentAttempt = models.QuizAttempt{
			QuizID:             newQuiz.ID,
			UserID:             userID,
			Score:              0,
			IsCompleted:        false,
			CurrentQuestionNum: 0,
		}
		if err := tx.Create(&currentAttempt).Error; err != nil {
			return fmt.Errorf("failed to create REVIEW QuizAttempt: %w", err)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	if newQuiz.QuestionCount == 0 { // Should be caught by transaction error
		return &QuizState{UserID: userID, IsCompleted: true, MessageToUser: msgQuizNoQuestions, QuizType: models.QuizTypeReview}, nil
	}

	currentQuestionModel := newQuiz.QuizQuesetions[currentAttempt.CurrentQuestionNum]
	shuffledOptions := make([]models.QuizQuestionOption, len(currentQuestionModel.Options))
	copy(shuffledOptions, currentQuestionModel.Options)
	rand.New(rand.NewSource(time.Now().UnixNano())).Shuffle(len(shuffledOptions), func(i, j int) {
		shuffledOptions[i], shuffledOptions[j] = shuffledOptions[j], shuffledOptions[i]
	})

	questionText := fmt.Sprintf("سوال %d از %d:\n\n%s", currentAttempt.CurrentQuestionNum+1, newQuiz.QuestionCount, currentQuestionModel.Text)

	return &QuizState{
		AttemptID:                currentAttempt.ID,
		QuizID:                   currentAttempt.QuizID,
		UserID:                   userID,
		CourseID:                 0, // No specific course for a general review quiz
		QuizType:                 models.QuizTypeReview,
		IsCompleted:              false,
		CurrentQuestionNum:       currentAttempt.CurrentQuestionNum,
		TotalQuestionsInQuiz:     newQuiz.QuestionCount,
		QuestionModelID:          currentQuestionModel.ID,
		QuestionText:             questionText,
		Options:                  shuffledOptions,
		MessageToUser:            messageToUser,
		CurrentQuestionMessageID: currentAttempt.CurrentQuestionMessageID, // Will be 0 initially
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
			Preload("QuizQuesetions.Options"). // Preload questions and their options
			First(&quizForAttempt, attempt.QuizID).
			Error; err != nil {
			return fmt.Errorf("loading quiz %d for attempt %d: %w", attempt.QuizID, attempt.ID, err)
		}

		if attempt.IsCompleted {
			finalResults, err := s.getQuizResultsInternal(tx, &attempt, &quizForAttempt) // Pass loaded quizForAttempt
			if err != nil {
				return fmt.Errorf("getting results for completed attempt %d: %w", attemptID, err)
			}
			submissionResult = &AnswerSubmissionResult{
				AttemptID: attempt.ID, QuizID: attempt.QuizID, UserID: userID, CourseID: quizForAttempt.CourseID, QuizType: quizForAttempt.Type,
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

		// Ensure question number is valid before accessing QuizQuestions slice
		if attempt.CurrentQuestionNum < 0 || attempt.CurrentQuestionNum >= int(quizForAttempt.QuestionCount) || len(quizForAttempt.QuizQuesetions) <= attempt.CurrentQuestionNum {
			log.Printf("QuizService: Invalid CurrentQuestionNum %d for attempt %d. Total questions: %d. Finalizing.", attempt.CurrentQuestionNum, attempt.ID, quizForAttempt.QuestionCount)
			finalResults, err := s.getQuizResultsInternal(tx, &attempt, &quizForAttempt)
			if err != nil {
				return fmt.Errorf("finalizing attempt %d with invalid q_num: %w", attemptID, err)
			}
			submissionResult = &AnswerSubmissionResult{
				AttemptID: attempt.ID, QuizID: attempt.QuizID, UserID: userID, CourseID: quizForAttempt.CourseID, QuizType: quizForAttempt.Type,
				IsQuizNowCompleted: true, FinalQuizResult: finalResults, CurrentQuestionMessageID: attempt.CurrentQuestionMessageID,
			}
			return nil
		}
		currentQuestionModel := quizForAttempt.QuizQuesetions[attempt.CurrentQuestionNum]
		expectedQuestionID := currentQuestionModel.ID
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

		// If it's a REVIEW quiz, update SRS schedule for the answered word
		if quizForAttempt.Type == models.QuizTypeReview {
			if currentQuestionModel.WordID == 0 {
				log.Printf("Error: Review quiz question (ID: %d) has no WordID associated. Cannot update SRS.", currentQuestionModel.ID)
			} else {
				errSRS := s.wordService.UpdateWordReviewSchedule(userID, currentQuestionModel.WordID, quizAnswer.IsCorrect)
				if errSRS != nil {
					log.Printf("Error updating SRS for WordID %d after review answer (AttemptID %d): %v", currentQuestionModel.WordID, attemptID, errSRS)
					// Non-fatal for quiz flow, but log it.
					// Potentially return a wrapped error if SRS update failure should halt things.
				}
			}
		}

		attempt.CurrentQuestionNum++ // Increment after processing current question
		if err := tx.Model(&attempt).Update("current_question_num", attempt.CurrentQuestionNum).Error; err != nil {
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
				QuizType:                 quizForAttempt.Type,
				IsQuizNowCompleted:       true,
				FinalQuizResult:          finalResults,
				CurrentQuestionMessageID: attempt.CurrentQuestionMessageID,
			}
		} else {
			nextQuestionModel := quizForAttempt.QuizQuesetions[attempt.CurrentQuestionNum]
			shuffledOptions := make([]models.QuizQuestionOption, len(nextQuestionModel.Options))
			copy(shuffledOptions, nextQuestionModel.Options)
			rand.New(rand.NewSource(time.Now().UnixNano())).Shuffle(len(shuffledOptions), func(i, j int) {
				shuffledOptions[i], shuffledOptions[j] = shuffledOptions[j], shuffledOptions[i]
			})
			questionText := fmt.Sprintf("سوال %d از %d:\n\n%s", attempt.CurrentQuestionNum+1, quizForAttempt.QuestionCount, nextQuestionModel.Text)

			submissionResult = &AnswerSubmissionResult{
				AttemptID:          attempt.ID,
				QuizID:             attempt.QuizID,
				UserID:             userID,
				CourseID:           quizForAttempt.CourseID,
				QuizType:           quizForAttempt.Type,
				IsQuizNowCompleted: false,
				NextQuestionState: &QuizState{
					AttemptID:                attempt.ID,
					QuizID:                   attempt.QuizID,
					UserID:                   userID,
					CourseID:                 quizForAttempt.CourseID,
					QuizType:                 quizForAttempt.Type,
					IsCompleted:              false,
					CurrentQuestionNum:       attempt.CurrentQuestionNum,
					TotalQuestionsInQuiz:     quizForAttempt.QuestionCount,
					QuestionModelID:          nextQuestionModel.ID,
					QuestionText:             questionText,
					Options:                  shuffledOptions,
					CurrentQuestionMessageID: attempt.CurrentQuestionMessageID, // This is the ID of the *previous* question message
				},
				CurrentQuestionMessageID: attempt.CurrentQuestionMessageID,
			}
		}
		return nil
	})

	if txErr != nil {
		// Wrap specific GORM errors if needed, or just return the processed error
		return nil, fmt.Errorf("%w: %w", ErrAnswerProcessing, txErr)
	}
	return submissionResult, nil
}

// getQuizResultsInternal finalizes an attempt and prepares results.
func (s *Service) getQuizResultsInternal(tx *gorm.DB, attempt *models.QuizAttempt, quizInfo *models.Quiz) (*QuizResult, error) {
	if !attempt.IsCompleted {
		var quizAnswers []models.QuizAnswer
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
	if quizInfo.Type == models.QuizTypeCourseBlock { // Only build detailed review for course block quizzes for now
		reviewBuilder.WriteString(msgReviewHeader)
		var answersForReview []models.QuizAnswer
		tx.Where("quiz_attempt_id = ?", attempt.ID).Find(&answersForReview) // Re-fetch for safety if not on attempt

		userAnswersMap := make(map[uint]models.QuizAnswer)
		var allChosenOptionIDs []uint
		for _, ans := range answersForReview {
			userAnswersMap[ans.QuizQuestionID] = ans
			allChosenOptionIDs = append(allChosenOptionIDs, ans.QuizQuestionOptionID)
		}
		chosenOptionsTextMap := make(map[uint]string)
		if len(allChosenOptionIDs) > 0 {
			var fetchedChosenOptions []models.QuizQuestionOption
			if err := tx.Where("id IN ?", allChosenOptionIDs).Find(&fetchedChosenOptions).Error; err == nil {
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
	}
	reviewText := reviewBuilder.String()

	passed := false
	resultMessage := ""
	shouldResetProgress := false
	suggestedNewProgress := uint(0)

	if quizInfo.Type == models.QuizTypeCourseBlock {
		passed = attempt.Score >= quizPassThresholdCorrectAnswers
		resultMessage = fmt.Sprintf(msgQuizFinished, attempt.Score, totalQuestions)
		if passed {
			resultMessage += msgQuizPassed
		} else {
			resultMessage += fmt.Sprintf(msgQuizFailed, quizPassThresholdCorrectAnswers)
			shouldResetProgress = true
			if quizInfo.TriggerProgress >= wordsPerQuizBlock {
				suggestedNewProgress = quizInfo.TriggerProgress - wordsPerQuizBlock + 1
			} else {
				suggestedNewProgress = 1
			}
			if suggestedNewProgress <= 0 { // Should not happen if logic above is correct
				suggestedNewProgress = 1
			}
		}
	} else if quizInfo.Type == models.QuizTypeReview {
		passed = true // For review, "passed" means completed the session. SRS handles word status.
		resultMessage = fmt.Sprintf(msgReviewQuizFinished, attempt.Score, totalQuestions)
		// Add a generic positive message or based on score.
		if attempt.Score == totalQuestions && totalQuestions > 0 {
			resultMessage += "\n🎉 عالی بود! همه رو درست جواب دادی!"
		} else if attempt.Score > 0 {
			resultMessage += "\n👍 خوب بود! به مرور ادامه بده."
		} else {
			resultMessage += "\nاشکالی نداره، دفعه بعد بهتر میشه!"
		}
	}

	return &QuizResult{
		AttemptID:            attempt.ID,
		QuizID:               quizInfo.ID,
		UserID:               attempt.UserID,
		CourseID:             quizInfo.CourseID,
		QuizType:             quizInfo.Type,
		Score:                attempt.Score,
		TotalQuestions:       totalQuestions,
		Passed:               passed,
		ReviewText:           reviewText,
		ResultMessage:        resultMessage,
		ShouldResetProgress:  shouldResetProgress,
		SuggestedNewProgress: suggestedNewProgress,
	}, nil
}

func (s *Service) GetQuizResults(attemptID uint, userID uint) (*QuizResult, error) {
	log.Printf("QuizService: GetQuizResults called for AttemptID: %d, UserID: %d", attemptID, userID)
	var finalResults *QuizResult
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

		var quizInfo models.Quiz
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
			return nil, nil // No active attempt is not an error in this context
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
	// Ensure messageID is not 0 if we are setting it, to avoid clearing a valid ID with an uninitialized one.
	// However, Telegram message IDs can be 0 if a message was deleted or never sent.
	// The handler should be responsible for providing a valid messageID.
	result := s.db.Model(&models.QuizAttempt{}).Where("id = ?", attemptID).Update("current_question_message_id", messageID)
	if result.Error != nil {
		return fmt.Errorf("failed to update message_id for attempt %d: %w", attemptID, result.Error)
	}
	if result.RowsAffected == 0 {
		// This might happen if the attemptID is invalid.
		log.Printf("QuizService: UpdateQuizAttemptMessageID - No rows affected for attempt %d. Attempt may not exist.", attemptID)
		// Consider returning ErrAttemptNotFound or a similar error if strictness is required.
	}
	return nil
}

func (s *Service) GetQuizType(quizID uint) (models.QuizType, error) {
	log.Printf("QuizService: GetQuizType called for QuizID: %d", quizID)
	var quiz models.Quiz
	if err := s.db.Select("type").First(&quiz, quizID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", fmt.Errorf("%w: quiz with ID %d not found", ErrQuizNotFound, quizID)
		}
		return "", fmt.Errorf("failed to fetch quiz type for QuizID %d: %w", quizID, err)
	}
	return quiz.Type, nil
}
