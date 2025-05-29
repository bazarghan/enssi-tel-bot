package handlers

import (
	"errors"
	"fmt"
	"github.com/2000ostd/enssi-tel-bot/internal/models"
	"gopkg.in/telebot.v4"
	"gorm.io/gorm"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

const QuizPassThresholdCorrectAnswers = 9 // e.g., 9 out of 12 for 75%
const WordsPerQuizBlock = 12
const QuizDefaultQuestionCount = 12

// sendCourseWord presents the word details comprehensively.
// The parameter 'idx' is the UserCourse.Progress.
func sendCourseWord(ctx telebot.Context, db *gorm.DB, courseID uint, idx uint) error {
	// 1. Fetch Core Data
	cw, err := fetchCourseWordByCourseIDAndIndex(db, courseID, idx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// This specific error message will be checked by the caller (handleNextWord)
			return errors.New("کلمه مورد نظر یافت نشد، ممکن است به انتهای دوره رسیده باشید یا دوره کلمه‌ای با این شماره نداشته باشد.")
		}
		log.Printf("Error fetching CourseWord (CourseID: %d, Index: %d): %v", courseID, idx, err)
		return ctx.Send("مشکلی در بارگذاری اطلاعات کلمه پیش آمد.")
	}

	word, err := fetchWord(db, cw.WordID)
	if err != nil {
		log.Printf("Error fetching Word (ID: %d): %v", cw.WordID, err)
		return ctx.Send("مشکلی در یافتن اطلاعات پایه کلمه پیش آمد.")
	}

	var ws models.WordSource
	if err := db.
		Where("word_id = ?", word.ID).
		Preload("Phonetics").
		Preload("Pronunciations").
		Preload("PartsOfSpeeches.Meanings"). // Eager load PartsOfSpeeches and their Meanings
		First(&ws).Error; err != nil {
		log.Printf("Error fetching WordSource with preloads for WordID %d: %v", word.ID, err)
		return ctx.Send("خطا در بارگذاری جزئیات کامل کلمه (تلفظ، معانی و غیره).")
	}

	// 2. Send Images (if available)
	if cw.TelgramImageID != "" {
		photo := &telebot.Photo{File: telebot.File{FileID: cw.TelgramImageID}}
		if err := ctx.Send(photo); err != nil {
			log.Printf("Error sending photo (FileID: %s) for WordID %d: %v", cw.TelgramImageID, word.ID, err)
			// Don't return yet, try sending document or text content
		}
	} else if cw.TelgramImageDocID != "" { // Use 'else if' if you prefer only one image, or remove 'else' to send both if both IDs exist
		doc := &telebot.Document{File: telebot.File{FileID: cw.TelgramImageDocID}}
		if err := ctx.Send(doc); err != nil {
			log.Printf("Error sending document (FileID: %s) for WordID %d: %v", cw.TelgramImageDocID, word.ID, err)
		}
	}

	// 3. Construct and Send Main Textual Information
	var messageBuilder strings.Builder

	// Word Title
	messageBuilder.WriteString(fmt.Sprintf("🔤 **%s**\n", strings.ToUpper(word.Title))) // Make title more prominent

	// Phonetics
	if len(ws.Phonetics) > 0 {
		messageBuilder.WriteString("\n🗣️ **تلفظ / IPA:**\n")
		for _, p := range ws.Phonetics {
			if p.Title != "" { // Only add if IPA text exists
				langTag := ""
				if p.Lang != "" {
					langTag = fmt.Sprintf(" (%s)", p.Lang)
				}
				messageBuilder.WriteString(fmt.Sprintf("  ▪️ %s%s\n", p.Title, langTag))
			}
		}
	}

	// Meanings by Part of Speech
	if len(ws.PartsOfSpeeches) > 0 {
		messageBuilder.WriteString("\n📜 **معانی و کاربردها:**\n")
		for _, pos := range ws.PartsOfSpeeches {
			if len(pos.Meanings) > 0 {
				messageBuilder.WriteString(fmt.Sprintf("\n  🏷️ **%s:**\n", pos.Title)) // Part of Speech Title
				for _, meaning := range pos.Meanings {
					langTag := ""
					if meaning.Lang != "" && meaning.Lang != "en" { // Optionally show lang if not default
						langTag = fmt.Sprintf(" (%s)", meaning.Lang)
					}
					messageBuilder.WriteString(fmt.Sprintf("    💡 %s%s\n", meaning.Title, langTag))
				}
			}
		}
	} else {
		// Fallback if no structured PartsOfSpeeches, but DefPrimary/Secondary exist
		if ws.DefPrimary != "" || ws.DefSecondary != "" {
			messageBuilder.WriteString("\n📜 **تعاریف اصلی:**\n")
			if ws.DefPrimary != "" {
				messageBuilder.WriteString(fmt.Sprintf("  💡 %s\n", ws.DefPrimary))
			}
			if ws.DefSecondary != "" {
				messageBuilder.WriteString(fmt.Sprintf("  💡 %s\n", ws.DefSecondary))
			}
		}
	}

	if messageBuilder.Len() > 0 {
		// Send the constructed message. Use MarkdownV2 if using ** for bold.
		// Ensure your text is properly escaped if using MarkdownV2. For simplicity, using default mode.
		// If using Markdown, escape special characters in word.Title, p.Title, pos.Title, meaning.Title.
		// For now, sending as plain text to avoid complex escaping.
		// If you want Markdown, you'll need to escape: _, *, [, ], (, ), ~, `, >, #, +, -, =, |, {, }, ., !
		if err := ctx.Send(messageBuilder.String()); err != nil {
			log.Printf("Error sending main text message for WordID %d: %v", word.ID, err)
			return ctx.Send("خطا در نمایش اطلاعات متنی کلمه.")
		}
	}

	// 4. Handle Voice Pronunciation
	// Try to send an existing voice first
	for _, pron := range ws.Pronunciations {
		if pron.TelgramVoiceID != "" {
			voice := &telebot.Voice{File: telebot.File{FileID: pron.TelgramVoiceID}}
			if err := ctx.Send(voice); err == nil {
				log.Printf("Sent existing voice (FileID: %s) for WordID %d.", pron.TelgramVoiceID, word.ID)
				return nil // Voice sent successfully
			} else {
				log.Printf("Error sending existing voice (FileID: %s) for WordID %d: %v. Will check for URLs.", pron.TelgramVoiceID, word.ID, err)
				// Don't return, try other pronunciations or download logic
			}
		}
	}

	// If no existing TelgramVoiceID was successfully sent, try to download and upload
	var usPronunciation *models.Pronunciation
	var ukPronunciation *models.Pronunciation // Fallback

	for i := range ws.Pronunciations {
		p := &ws.Pronunciations[i] // Use pointer to modify if needed (though not modifying p directly here)
		if p.URL != "" {
			if strings.ToUpper(p.Region) == "US" {
				usPronunciation = p
				break // Prefer US
			}
			if strings.ToUpper(p.Region) == "UK" && ukPronunciation == nil { // Take first UK as fallback
				ukPronunciation = p
			}
		}
	}

	pronToDownload := usPronunciation
	if pronToDownload == nil {
		pronToDownload = ukPronunciation
	}

	if pronToDownload != nil && pronToDownload.URL != "" {
		log.Printf("Attempting to download voice from URL: %s for WordID %d (PronunciationID: %d)", pronToDownload.URL, word.ID, pronToDownload.ID)

		// Download the audio file
		// Consider adding a timeout to http.Get
		// client := http.Client{Timeout: 10 * time.Second}
		// resp, err := client.Get(pronToDownload.URL)
		resp, err := http.Get(pronToDownload.URL)
		if err != nil {
			log.Printf("Error downloading audio from %s: %v", pronToDownload.URL, err)
			return nil // Silently fail, don't send error to user about voice
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("Error downloading audio: status code %d from %s", resp.StatusCode, pronToDownload.URL)
			return nil // Silently fail
		}

		// Send the downloaded audio as a voice message
		// Using telebot.FromReader to stream directly if possible, or save to temp file if needed by library/Telegram.
		// Telebot's Voice struct can take an io.Reader via telebot.FromReader.
		voiceFile := telebot.FromReader(resp.Body)
		sentVoiceMsg, err := ctx.Bot().Send(ctx.Chat(), &telebot.Voice{File: voiceFile})
		if err != nil {
			log.Printf("Error sending downloaded voice to Telegram for WordID %d (URL: %s): %v", word.ID, pronToDownload.URL, err)
			return nil // Silently fail
		}

		// If successful, update the database with the new TelgramVoiceID
		if sentVoiceMsg.Voice != nil && sentVoiceMsg.Voice.FileID != "" {
			newFileID := sentVoiceMsg.Voice.FileID
			log.Printf("Successfully sent downloaded voice. New FileID: %s for PronunciationID: %d", newFileID, pronToDownload.ID)

			// Update the specific pronunciation record
			if errDbUpdate := db.Model(&models.Pronunciation{}).Where("id = ?", pronToDownload.ID).Update("telgram_voice_id", newFileID).Error; errDbUpdate != nil {
				log.Printf("CRITICAL: Failed to update TelgramVoiceID in DB for PronunciationID %d (New FileID: %s): %v", pronToDownload.ID, newFileID, errDbUpdate)
				// The voice was sent, but DB update failed. This is an internal issue.
			} else {
				log.Printf("Successfully updated TelgramVoiceID in DB for PronunciationID %d.", pronToDownload.ID)
			}
		} else {
			log.Printf("Sent voice message for WordID %d, but Voice or FileID was empty in response.", word.ID)
		}
	} else {
		log.Printf("No suitable pronunciation URL found to download for WordID %d.", word.ID)
	}

	return nil // Successfully sent word info, voice handling was best-effort
}

func createQuestion(db *gorm.DB, quizID uint, courseID uint, currentWordCourseIndex uint, wordPoolIndices []uint) error {

	// Initialize random number generator for this question creation session
	localRand := rand.New(rand.NewSource(time.Now().UnixNano()))

	// 1. Fetch the CourseWord for the current question
	cw, err := fetchCourseWordByCourseIDAndIndex(db, courseID, currentWordCourseIndex)
	if err != nil {
		return fmt.Errorf("createQuestion: failed to fetch course word (courseID: %d, index: %d): %w", courseID, currentWordCourseIndex, err)
	}

	// 2. Fetch the Word model for the question
	questionWord, err := fetchWord(db, cw.WordID)
	if err != nil {
		return fmt.Errorf("createQuestion: failed to fetch word (ID: %d) for question: %w", cw.WordID, err)
	}

	// 3. Fetch WordSource and its Meanings for the question word to select a correct answer
	var questionWS models.WordSource
	if err := db.Where("word_id = ?", questionWord.ID).
		Preload("PartsOfSpeeches.Meanings").
		First(&questionWS).Error; err != nil {
		return fmt.Errorf("createQuestion: failed to fetch word source with meanings for word ID %d (title: %s): %w", questionWord.ID, questionWord.Title, err)
	}

	var questionWordAllMeanings []string
	for _, pos := range questionWS.PartsOfSpeeches {
		for _, m := range pos.Meanings {
			if strings.TrimSpace(m.Title) != "" { // Ensure meaning is not empty or just whitespace
				questionWordAllMeanings = append(questionWordAllMeanings, m.Title)
			}
		}
	}

	if len(questionWordAllMeanings) == 0 {
		// Fallback to DefPrimary if no granular meanings are found, or error out
		if strings.TrimSpace(questionWS.DefPrimary) != "" {
			questionWordAllMeanings = append(questionWordAllMeanings, questionWS.DefPrimary)
		} else {
			// If still no meaning, then we cannot create a question.
			return fmt.Errorf("createQuestion: word (ID: %d, title: %s) has no usable meanings (neither granular nor DefPrimary), cannot create question", questionWord.ID, questionWord.Title)
		}
	}

	// Randomly select one meaning as the correct answer
	correctMeaningText := questionWordAllMeanings[localRand.Intn(len(questionWordAllMeanings))]

	// 4. Create the QuizQuestion database entry
	newQuestion := models.QuizQuestion{
		QuizID: quizID,
		Text:   questionWord.Title, // The question text is the word itself
	}
	if err := db.Create(&newQuestion).Error; err != nil {
		return fmt.Errorf("createQuestion: failed to create quiz question DB entry: %w", err)
	}

	// 5. Create the Correct QuizQuestionOption
	correctOption := models.QuizQuestionOption{
		QuizQuestionID: newQuestion.ID,
		Text:           correctMeaningText,
		IsCorrect:      true,
	}
	if err := db.Create(&correctOption).Error; err != nil {
		return fmt.Errorf("createQuestion: failed to create correct quiz option: %w", err)
	}

	// 6. Prepare distractor options
	numDistractors := 3 // Typically 3 distractors for a 4-option question
	var allPotentialDistractorMeanings []string

	// Collect meanings from the word pool
	for _, poolWordIndex := range wordPoolIndices {
		// Skip if this pool word is the same as the current question word
		// here is the warning make sure to check it the indexing should be correct
		if poolWordIndex+1 == currentWordCourseIndex {
			continue
		}

		poolCW, err := fetchCourseWordByCourseIDAndIndex(db, courseID, poolWordIndex+1)
		if err != nil {
			fmt.Printf("Warning: createQuestion: failed to fetch pool course word (index %d) for distractors: %v. Skipping this word.\n", poolWordIndex, err)
			continue
		}
		poolWord, err := fetchWord(db, poolCW.WordID)
		if err != nil {
			fmt.Printf("Warning: createQuestion: failed to fetch pool word (ID %d) for distractors: %v. Skipping this word.\n", poolCW.WordID, err)
			continue
		}

		var poolWS models.WordSource
		if err := db.Where("word_id = ?", poolWord.ID).
			Preload("PartsOfSpeeches.Meanings"). // Preload meanings for distractor pool words
			First(&poolWS).Error; err != nil {
			fmt.Printf("Warning: createQuestion: failed to fetch word source for pool word ID %d (title: %s) for distractors: %v. Skipping.\n", poolWord.ID, poolWord.Title, err)
			continue
		}

		for _, pos := range poolWS.PartsOfSpeeches {
			for _, m := range pos.Meanings {
				trimmedMeaning := strings.TrimSpace(m.Title)
				// Ensure distractor is not empty and not the same as the correct answer's meaning
				// for future is should similiarity check this
				if trimmedMeaning != "" && trimmedMeaning != correctMeaningText {
					allPotentialDistractorMeanings = append(allPotentialDistractorMeanings, trimmedMeaning)
				}
			}
		}

	}

	// Shuffle and pick unique distractors
	localRand.Shuffle(len(allPotentialDistractorMeanings), func(i, j int) {
		allPotentialDistractorMeanings[i], allPotentialDistractorMeanings[j] = allPotentialDistractorMeanings[j], allPotentialDistractorMeanings[i]
	})

	finalDistractorOptions := []string{}
	seenDistractors := make(map[string]bool)
	seenDistractors[correctMeaningText] = true // Don't pick the correct meaning as a distractor

	for _, meaning := range allPotentialDistractorMeanings {
		if len(finalDistractorOptions) >= numDistractors {
			break
		}
		if !seenDistractors[meaning] {
			finalDistractorOptions = append(finalDistractorOptions, meaning)
			seenDistractors[meaning] = true
		}
	}

	// Create distractor options in the database
	for _, distractorText := range finalDistractorOptions {
		distractorOption := models.QuizQuestionOption{
			QuizQuestionID: newQuestion.ID,
			Text:           distractorText,
			IsCorrect:      false,
		}
		if err := db.Create(&distractorOption).Error; err != nil {
			// Log or handle error; maybe continue to create other options
			fmt.Printf("Warning: createQuestion: failed to create distractor option ('%s'): %v\n", distractorText, err)
		}
	}

	if len(finalDistractorOptions) < numDistractors {
		// This means we couldn't find enough unique distractors from the pool.
		fmt.Printf("Warning: createQuestion: created only %d out of %d required distractors for question '%s' (QuizQuestionID: %d). WordPool might be too small or meanings too similar.\n", len(finalDistractorOptions), numDistractors, questionWord.Title, newQuestion.ID)

	}

	return nil
}

// createQuizStructure creates the Quiz DB entry and its questions.
// userProgressAtTrigger is the UserCourse.Progress value for which this quiz is being created (e.g., 12, 24).
func createQuizStructure(db *gorm.DB, courseID uint, userProgressAtTrigger uint) (*models.Quiz, int, error) {
	questionCountTarget := QuizDefaultQuestionCount

	// Word pool is for the block ending at userProgressAtTrigger
	lastLearnedIndexInCourse := userProgressAtTrigger
	if lastLearnedIndexInCourse < uint(questionCountTarget) {
		questionCountTarget = int(lastLearnedIndexInCourse)
		if questionCountTarget == 0 {
			return nil, 0, fmt.Errorf("createQuizStructure: not enough words learned (progress: %d) to form any questions", userProgressAtTrigger)
		}
	}

	wordPoolIndices := make([]uint, 0, questionCountTarget)
	firstIndexInPool := lastLearnedIndexInCourse - uint(questionCountTarget) + 1
	if firstIndexInPool <= 0 { // Guard against progress < questionCountTarget leading to 0 or negative
		firstIndexInPool = 1
	}

	for i := 0; i < questionCountTarget; i++ {
		currentIndexToConsider := firstIndexInPool + uint(i)
		if currentIndexToConsider > 0 && currentIndexToConsider <= lastLearnedIndexInCourse {
			wordPoolIndices = append(wordPoolIndices, currentIndexToConsider)
		}
	}

	if len(wordPoolIndices) == 0 {
		return nil, 0, fmt.Errorf("createQuizStructure: no word indices could be determined. ProgressAtTrigger: %d", userProgressAtTrigger)
	}
	actualQuestionCountFromPool := len(wordPoolIndices)

	newQuiz := models.Quiz{
		CourseID:        courseID,
		Type:            "multi-option",
		QuestionCount:   uint(actualQuestionCountFromPool), // Tentative
		TriggerProgress: userProgressAtTrigger,             // Set the TriggerProgress
	}
	if err := db.Create(&newQuiz).Error; err != nil {
		return nil, 0, fmt.Errorf("createQuizStructure: failed to create quiz entry: %w", err)
	}

	questionsCreatedSuccessfully := 0
	for _, wordIndexForQuestion := range wordPoolIndices {
		err := createQuestion(db, newQuiz.ID, courseID, wordIndexForQuestion, wordPoolIndices)
		if err != nil {
			fmt.Printf("Error creating question for course word index %d (QuizID: %d): %v. Skipping.\n", wordIndexForQuestion, newQuiz.ID, err)
		} else {
			questionsCreatedSuccessfully++
		}
	}

	if questionsCreatedSuccessfully == 0 {
		db.Delete(&newQuiz)
		return nil, 0, fmt.Errorf("createQuizStructure: no questions could be created")
	}

	if uint(questionsCreatedSuccessfully) != newQuiz.QuestionCount {
		newQuiz.QuestionCount = uint(questionsCreatedSuccessfully)
		if err := db.Model(&newQuiz).Update("question_count", newQuiz.QuestionCount).Error; err != nil {
			fmt.Printf("Warning: Failed to update actual question count for QuizID %d: %v\n", newQuiz.ID, err)
		}
	}

	if err := db.Preload("QuizQuesetions.Options").First(&newQuiz, newQuiz.ID).Error; err != nil {
		return nil, 0, fmt.Errorf("createQuizStructure: failed to reload quiz with questions: %w", err)
	}

	return &newQuiz, questionsCreatedSuccessfully, nil
}

// createQuizAndAttemptForUser now uses userProgressAtTrigger for createQuizStructure
func createQuizAndAttemptForUser(db *gorm.DB, userID uint, courseID uint, userProgressAtTrigger uint) (*models.QuizAttempt, error) {
	quiz, questionsCount, err := createQuizStructure(db, courseID, userProgressAtTrigger)
	if err != nil {
		return nil, fmt.Errorf("createQuizAndAttemptForUser: failed to create quiz structure: %w", err)
	}
	if questionsCount == 0 || quiz == nil {
		return nil, fmt.Errorf("createQuizAndAttemptForUser: quiz created with no questions")
	}

	quizAttempt := models.QuizAttempt{
		QuizID:             quiz.ID,
		UserID:             userID,
		Score:              0,
		IsCompleted:        false,
		CurrentQuestionNum: 0, // Start at the first question
	}
	if err := db.Create(&quizAttempt).Error; err != nil {
		return nil, fmt.Errorf("createQuizAndAttemptForUser: failed to create quiz attempt: %w", err)
	}
	return &quizAttempt, nil
}
