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

func (s *Service) createQuestionInternal(
	tx *gorm.DB,
	quizID uint,
	wordID uint,
	questionTextSeed string,
	courseID uint,
	distractorSourceIndices []uint,
) error {
	localRand := rand.New(rand.NewSource(time.Now().UnixNano()))
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
			if strings.TrimSpace(m.Title) != "" && (m.Lang == "Fa" || m.Lang == "En" || m.Lang == "") {
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
		Text:   questionTextSeed,
		WordID: wordID,
	}
	if err := tx.Create(&newQuestion).Error; err != nil {
		return fmt.Errorf("failed to create quiz question entry: %w", err)
	}

	correctOption := models.QuizQuestionOption{QuizQuestionID: newQuestion.ID, Text: correctMeaningText, IsCorrect: true}
	if err := tx.Create(&correctOption).Error; err != nil {
		return fmt.Errorf("failed to create correct quiz option: %w", err)
	}

	numDistractorsNeeded := reviewQuizMaxOptions - 1
	var potentialDistractorMeanings []string
	if courseID != 0 && len(distractorSourceIndices) > 0 {
		for _, poolWordIdx := range distractorSourceIndices {
			var poolCW models.CourseWord
			if err := tx.Where("course_id = ? AND index = ?", courseID, poolWordIdx).First(&poolCW).Error; err == nil {
				if poolCW.WordID == wordID {
					continue
				}
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
	}

	if len(potentialDistractorMeanings) < numDistractorsNeeded {
		var randomWords []models.Word
		err := tx.Not("id = ?", wordID).Order(gorm.Expr("RANDOM()")).Limit(numDistractorsNeeded * 3).Find(&randomWords).Error // Fetch more for better variety
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
	seenDistractors[correctMeaningText] = true
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
	}
	return nil
}

func (s *Service) createCourseBlockQuizStructureInternal(tx *gorm.DB, courseID uint, userProgressAtTrigger uint) (*models.Quiz, error) {
	questionCountTarget := quizDefaultQuestionCount
	lastWordIndexInBlock := userProgressAtTrigger
	firstWordIndexInBlock := uint(1)
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

	localRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	localRand.Shuffle(len(courseWordsInBlock), func(i, j int) {
		courseWordsInBlock[i], courseWordsInBlock[j] = courseWordsInBlock[j], courseWordsInBlock[i]
	})

	questionWordsFromBlock := courseWordsInBlock[:numQuestionsToGenerate]
	distractorPoolIndices := make([]uint, len(courseWordsInBlock))
	for i, cw := range courseWordsInBlock {
		distractorPoolIndices[i] = cw.Index
	}

	newQuiz := models.Quiz{
		CourseID:        courseID,
		Type:            models.QuizTypeCourseBlock,
		QuestionCount:   uint(numQuestionsToGenerate),
		TriggerProgress: userProgressAtTrigger,
	}
	if err := tx.Create(&newQuiz).Error; err != nil {
		return nil, fmt.Errorf("failed to create quiz DB entry for course block: %w", err)
	}

	questionsCreatedSuccessfully := 0
	for _, cw := range questionWordsFromBlock {
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
		tx.Delete(&newQuiz)
		return nil, fmt.Errorf("%w: no questions could be generated for course block quiz", ErrQuizCreation)
	}

	if uint(questionsCreatedSuccessfully) != newQuiz.QuestionCount {
		newQuiz.QuestionCount = uint(questionsCreatedSuccessfully)
		if err := tx.Model(&newQuiz).Update("question_count", newQuiz.QuestionCount).Error; err != nil {
			log.Printf("Warning: Failed to update actual question count for CourseBlock QuizID %d: %v", newQuiz.ID, err)
		}
	}
	// IMPORTANT: newQuiz struct here does NOT have QuizQuestions preloaded.
	return &newQuiz, nil
}

// createQuizAndAttemptInternal is for COURSE_BLOCK quizzes
func (s *Service) createQuizAndAttemptInternal(tx *gorm.DB, userID uint, courseID uint, userProgressAtTrigger uint) (*models.QuizAttempt, *models.Quiz, error) {
	quizStruct, err := s.createCourseBlockQuizStructureInternal(tx, courseID, userProgressAtTrigger)
	if err != nil {
		return nil, nil, fmt.Errorf("creating course block quiz structure: %w", err)
	}
	// quizStruct returned here does not have QuizQuestions loaded in the Go struct itself.
	if quizStruct == nil || quizStruct.QuestionCount == 0 { // Rely on QuestionCount field.
		return nil, nil, fmt.Errorf("%w: course block quiz structure indicates no questions or is nil after creation attempt", ErrQuizCreation)
	}

	quizAttempt := models.QuizAttempt{
		QuizID:             quizStruct.ID, // Use ID from the created quiz
		UserID:             userID,
		Score:              0,
		IsCompleted:        false,
		CurrentQuestionNum: 0,
	}
	if err := tx.Create(&quizAttempt).Error; err != nil {
		return nil, quizStruct, fmt.Errorf("failed to create quiz attempt for course block quiz: %w", err)
	}
	return &quizAttempt, quizStruct, nil
}

// --- Public Service Methods ---

func (s *Service) StartOrResumeQuiz(userID uint, courseID uint, userCourseProgress uint) (*QuizState, error) {
	log.Printf("QuizService: StartOrResumeQuiz (COURSE_BLOCK) called for UserID: %d, CourseID: %d, ProgressTrigger: %d", userID, courseID, userCourseProgress)

	var currentAttempt *models.QuizAttempt
	// Removed: var newlyCreatedQuizStruct *models.Quiz
	var messageToUser string
	isNewAttempt := false

	// Transaction for finding or creating the attempt
	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		var activeAttempt models.QuizAttempt
		err := tx.Joins("JOIN quizzes ON quizzes.id = quiz_attempts.quiz_id").
			Where("quiz_attempts.user_id = ? AND quizzes.course_id = ? AND quizzes.trigger_progress = ? AND quiz_attempts.is_completed = ? AND quizzes.type = ?",
				userID, courseID, userCourseProgress, false, models.QuizTypeCourseBlock).
			Order("quiz_attempts.created_at DESC").
			First(&activeAttempt).Error

		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				log.Printf("QuizService: No active COURSE_BLOCK quiz. Creating new one for progress trigger %d.", userCourseProgress)
				// Pass nil for the *models.Quiz return as it's not used by the caller of createQuizAndAttemptInternal directly in this path
				newAttempt, _, createErr := s.createQuizAndAttemptInternal(tx, userID, courseID, userCourseProgress)
				if createErr != nil {
					return fmt.Errorf("from createQuizAndAttemptInternal: %w", createErr)
				}
				currentAttempt = newAttempt
				messageToUser = fmt.Sprintf(msgQuizTime, userCourseProgress)
				isNewAttempt = true
			} else {
				return fmt.Errorf("db error finding active course block quiz attempt: %w", err)
			}
		} else {
			log.Printf("QuizService: Resuming active COURSE_BLOCK quiz attempt %d.", activeAttempt.ID)
			currentAttempt = &activeAttempt
			messageToUser = msgResumeActiveQuiz
		}
		return nil
	})

	if txErr != nil {
		return nil, txErr
	}
	if currentAttempt == nil {
		return nil, errors.New("internal error: quiz attempt not initialized after transaction")
	}

	var quizForAttempt models.Quiz
	if errDb := s.db.Preload("QuizQuesetions.Options").First(&quizForAttempt, currentAttempt.QuizID).Error; errDb != nil {
		log.Printf("QuizService: Critical error - Failed to load Quiz (ID: %d) for Attempt (ID: %d): %v", currentAttempt.QuizID, currentAttempt.ID, errDb)
		return nil, fmt.Errorf("%w: loading quiz (ID %d) details for attempt %d: %w", ErrQuizNotFound, currentAttempt.QuizID, currentAttempt.ID, errDb)
	}

	if quizForAttempt.QuestionCount == 0 {
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

	if currentAttempt.CurrentQuestionNum < 0 || currentAttempt.CurrentQuestionNum >= len(quizForAttempt.QuizQuesetions) {
		log.Printf("QuizService: Invalid CurrentQuestionNum %d for COURSE_BLOCK attempt %d (Total questions in slice: %d). Finalizing.",
			currentAttempt.CurrentQuestionNum, currentAttempt.ID, len(quizForAttempt.QuizQuesetions))
		s.db.Model(&currentAttempt).Updates(map[string]interface{}{"is_completed": true, "score": currentAttempt.Score})
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
		CourseID:                 quizForAttempt.CourseID,
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
		createdQuizStruct := models.Quiz{
			Type:          models.QuizTypeReview,
			QuestionCount: uint(len(wordsToReview)),
		}
		if err := tx.Create(&createdQuizStruct).Error; err != nil {
			return fmt.Errorf("failed to create REVIEW Quiz DB entry: %w", err)
		}

		questionsCreatedSuccessfully := 0
		for _, wordView := range wordsToReview {
			if err := s.createQuestionInternal(tx, createdQuizStruct.ID, wordView.WordID, wordView.WordTitle, 0, nil); err != nil {
				log.Printf("Error creating REVIEW question for WordID %d (QuizID: %d): %v. Skipping.", wordView.WordID, createdQuizStruct.ID, err)
			} else {
				questionsCreatedSuccessfully++
			}
		}

		if questionsCreatedSuccessfully == 0 {
			tx.Delete(&createdQuizStruct)
			return fmt.Errorf("%w: no questions could be generated for review quiz", ErrQuizCreation)
		}
		if uint(questionsCreatedSuccessfully) != createdQuizStruct.QuestionCount {
			createdQuizStruct.QuestionCount = uint(questionsCreatedSuccessfully)
			if err := tx.Model(&createdQuizStruct).Update("question_count", createdQuizStruct.QuestionCount).Error; err != nil {
				log.Printf("Warning: Failed to update actual question count for REVIEW QuizID %d: %v", createdQuizStruct.ID, err)
			}
		}

		if err := tx.Preload("QuizQuesetions.Options").First(&newQuiz, createdQuizStruct.ID).Error; err != nil {
			return fmt.Errorf("failed to reload review quiz (ID: %d) with questions: %w", createdQuizStruct.ID, err)
		}

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
	if newQuiz.QuestionCount == 0 || len(newQuiz.QuizQuesetions) == 0 {
		log.Printf("QuizService: Review quiz (ID %d) created with no questions, though QuestionCount field is %d.", newQuiz.ID, newQuiz.QuestionCount)
		s.db.Model(&currentAttempt).Update("is_completed", true)
		return &QuizState{UserID: userID, AttemptID: currentAttempt.ID, QuizID: newQuiz.ID, IsCompleted: true, MessageToUser: msgQuizNoQuestions, QuizType: models.QuizTypeReview}, nil
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
		CourseID:                 0,
		QuizType:                 models.QuizTypeReview,
		IsCompleted:              false,
		CurrentQuestionNum:       currentAttempt.CurrentQuestionNum,
		TotalQuestionsInQuiz:     newQuiz.QuestionCount,
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

		if attempt.CurrentQuestionNum < 0 || attempt.CurrentQuestionNum >= len(quizForAttempt.QuizQuesetions) {
			log.Printf("QuizService: Invalid CurrentQuestionNum %d for attempt %d. Total questions: %d. Finalizing.", attempt.CurrentQuestionNum, attempt.ID, len(quizForAttempt.QuizQuesetions))
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

		if quizForAttempt.Type == models.QuizTypeReview {
			if currentQuestionModel.WordID == 0 {
				log.Printf("Error: Review quiz question (ID: %d) has no WordID associated. Cannot update SRS.", currentQuestionModel.ID)
			} else {
				errSRS := s.wordService.UpdateWordReviewSchedule(userID, currentQuestionModel.WordID, quizAnswer.IsCorrect)
				if errSRS != nil {
					log.Printf("Error updating SRS for WordID %d after review answer (AttemptID %d): %v", currentQuestionModel.WordID, attemptID, errSRS)
				}
			}
		}

		attempt.CurrentQuestionNum++
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
			if attempt.CurrentQuestionNum >= len(quizForAttempt.QuizQuesetions) {
				log.Printf("QuizService: CurrentQuestionNum (%d) became out of bounds after increment for attempt %d (len: %d). Finalizing.", attempt.CurrentQuestionNum, attempt.ID, len(quizForAttempt.QuizQuesetions))
				finalResults, err := s.getQuizResultsInternal(tx, &attempt, &quizForAttempt)
				if err != nil {
					return fmt.Errorf("finalizing quiz attempt %d (bounds error): %w", attemptID, err)
				}
				submissionResult = &AnswerSubmissionResult{
					AttemptID: attempt.ID, QuizID: attempt.QuizID, UserID: userID, CourseID: quizForAttempt.CourseID, QuizType: quizForAttempt.Type,
					IsQuizNowCompleted: true, FinalQuizResult: finalResults, CurrentQuestionMessageID: attempt.CurrentQuestionMessageID,
				}
				return nil
			}

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
	if quizInfo.Type == models.QuizTypeCourseBlock && len(quizInfo.QuizQuesetions) > 0 {
		reviewBuilder.WriteString(msgReviewHeader)
		var answersForReview []models.QuizAnswer
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
				suggestedNewProgress = quizInfo.TriggerProgress - wordsPerQuizBlock
			} else {
				suggestedNewProgress = 0
			}
			if suggestedNewProgress <= 0 {
				suggestedNewProgress = 0
			}
		}
	} else if quizInfo.Type == models.QuizTypeReview {
		passed = true
		resultMessage = fmt.Sprintf(msgReviewQuizFinished, attempt.Score, totalQuestions)
		if totalQuestions > 0 && attempt.Score == totalQuestions {
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
		log.Printf("QuizService: UpdateQuizAttemptMessageID - No rows affected for attempt %d. Attempt may not exist.", attemptID)
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
