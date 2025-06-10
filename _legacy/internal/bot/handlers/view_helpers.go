package handlers

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time" // Added for CheckAndInitiateReview

	"github.com/2000ostd/enssi-tel-bot/internal/bot/formatters"
	"github.com/2000ostd/enssi-tel-bot/internal/bot/keyboards"
	"github.com/2000ostd/enssi-tel-bot/internal/models" // Added for models.User
	"github.com/2000ostd/enssi-tel-bot/internal/services"
	"github.com/2000ostd/enssi-tel-bot/internal/services/course"
	"github.com/2000ostd/enssi-tel-bot/internal/services/quiz"
	"github.com/2000ostd/enssi-tel-bot/internal/services/word"
	"gopkg.in/telebot.v4"
)

// sendServiceError sends a generic error message to the user.
func SendServiceError(c telebot.Context, operation string, err error) error {
	log.Printf("[Service ERROR] UserID %d performing %s: %v", c.Sender().ID, operation, err)
	if cb := c.Callback(); cb != nil {
		respErr := c.Respond(&telebot.CallbackResponse{
			Text:      "متاسفانه مشکلی پیش آمده.",
			ShowAlert: true,
		})
		if respErr != nil {
			log.Printf("[Service ERROR] Also failed to respond to callback for user %d: %v", c.Sender().ID, respErr)
		}
		return nil
	}
	return c.Send("متاسفانه در انجام عملیات مشکلی پیش آمد. لطفا دوباره تلاش کنید.", keyboards.MainMenu)
}

// sendLearningContext processes the LearningContext DTO and sends the appropriate content (word or quiz).
func sendLearningContext(c telebot.Context, lc *course.LearningContext, appServices *services.AppServices) error {
	if lc == nil {
		log.Printf("[ViewHelper ERROR] UserID %d: Received nil LearningContext", c.Sender().ID)
		return c.Send("محتوایی برای نمایش یافت نشد. لطفا به منوی اصلی بازگردید.", keyboards.MainMenu)
	}

	currentUser, err := appServices.User().GetOrCreateUserByTelegramID(c.Sender().ID, c.Sender().Username, c.Sender().FirstName, c.Sender().LastName)
	if err != nil {
		return SendServiceError(c, "updating menu state (fetch user)", err)
	}

	var menuState string
	if lc.IsQuizDue && lc.QuizState != nil {
		menuState = InQuizMenuState(lc.QuizState.AttemptID)
	} else if lc.WordToDisplay != nil {
		menuState = InCourseMenuState(lc.CourseID)
	} else if lc.IsCourseEnded {
		menuState = CourseDetailsMenuState(lc.CourseID)
	} else {
		menuState = StateMain
	}

	err = appServices.User().UpdateUserLastMenu(currentUser.ID, menuState)
	if err != nil {
		log.Printf("[ViewHelper WARN] UserID %d: Failed to update LastMenu to '%s': %v", currentUser.ID, menuState, err)
	}

	if lc.IsCourseEnded {
		endMessage := "🎉 " + formatters.EscapeMarkdownV2(lc.MessageToUser)
		// keyboards.CourseCompletedKeyboard definition likely does not take arguments.
		return c.Send(endMessage, keyboards.CourseCompletedKeyboard(), telebot.ModeMarkdownV2)
	}

	if lc.IsQuizDue {
		if lc.QuizState == nil {
			log.Printf("[ViewHelper ERROR] UserID %d: IsQuizDue is true but QuizState is nil.", currentUser.ID)
			return c.Send("آماده سازی آزمون با مشکل مواجه شد.", keyboards.MainMenu)
		}
		quizTimeReplyKeyboard := keyboards.BackToMainMenuKeyboard()
		if lc.MessageToUser != "" {
			if err := c.Send(formatters.EscapeMarkdownV2(lc.MessageToUser), quizTimeReplyKeyboard, telebot.ModeMarkdownV2); err != nil {
				log.Printf("[ViewHelper WARN] UserID %d: Failed to send Quiz Intro Message: %v", currentUser.ID, err)
			}
		}
		return sendQuizQuestion(c, lc.QuizState, appServices.Quiz())
	}

	if lc.WordToDisplay != nil {
		// keyboards.InCourseNavigationKeyboard definition likely does not take arguments.
		return sendWordDisplay(c, lc.WordToDisplay, appServices.Word())
	}

	log.Printf("[ViewHelper INFO] UserID %d: No specific content in LearningContext (not ended, no quiz, no word). Progress: %d", currentUser.ID, lc.UserProgress)
	return c.Send("محتوای بعدی در دسترس نیست. لطفا به منوی اصلی بروید.", keyboards.MainMenu)
}

// sendWordDisplay sends the word details (text, image, audio).
// Removed courseID parameter as InCourseNavigationKeyboard likely doesn't use it.
func sendWordDisplay(c telebot.Context, wordData *word.WordDisplayData, wordService word.WordService) error {
	if strings.TrimSpace(wordData.FormattedText) != "" {
		if err := c.Send(wordData.FormattedText, keyboards.InCourseNavigationKeyboard(), telebot.ModeMarkdownV2); err != nil {
			log.Printf("[ViewHelper ERROR] UserID %d, WordID %d: Sending MarkdownV2 text failed: %v. Retrying plain.", c.Sender().ID, wordData.WordID, err)
			return c.Send("خطا در نمایش اطلاعات کلمه.", keyboards.InCourseNavigationKeyboard())
		}
	} else {
		if err := c.Send("کلمه بارگذاری شد، اما محتوایی برای نمایش نبود.", keyboards.InCourseNavigationKeyboard()); err != nil {
			log.Printf("[ViewHelper ERROR] UserID %d, WordID %d: Failed to send placeholder for empty word text: %v", c.Sender().ID, wordData.WordID, err)
		}
	}

	var imageFileIDToCache, imageDocFileIDToCache string
	// Removed unused: courseWordEntryForImageCache *models.CourseWord

	if wordData.TelegramImageID != "" {
		if err := c.Send(&telebot.Photo{File: telebot.File{FileID: wordData.TelegramImageID}}); err != nil {
			log.Printf("[ViewHelper WARN] UserID %d, WordID %d: Sending cached image photo (FileID: %s) failed: %v. Will try URL.", c.Sender().ID, wordData.WordID, wordData.TelegramImageID, err)
			if wordData.ImageURL != "" {
				sentMsg, errURL := c.Bot().Send(c.Chat(), &telebot.Photo{File: telebot.FromURL(wordData.ImageURL)})
				if errURL != nil {
					log.Printf("[ViewHelper WARN] UserID %d, WordID %d: Sending image by URL (%s) failed: %v", c.Sender().ID, wordData.WordID, wordData.ImageURL, errURL)
				} else if sentMsg != nil && sentMsg.Photo != nil {
					imageFileIDToCache = sentMsg.Photo.FileID
				}
			}
		}
	} else if wordData.ImageURL != "" {
		sentMsg, err := c.Bot().Send(c.Chat(), &telebot.Photo{File: telebot.FromURL(wordData.ImageURL)})
		if err != nil {
			log.Printf("[ViewHelper WARN] UserID %d, WordID %d: Sending image by URL (%s) failed: %v", c.Sender().ID, wordData.WordID, wordData.ImageURL, err)
		} else if sentMsg != nil && sentMsg.Photo != nil {
			imageFileIDToCache = sentMsg.Photo.FileID
		}
	}

	// ---> ADD THIS NEW BLOCK FOR SENDING THE IMAGE AS A DOCUMENT <---
	if wordData.TelegramImageDocID != "" {
		log.Printf("[ViewHelper INFO] UserID %d, WordID %d: Attempting to send image as document using FileID: %s", c.Sender().ID, wordData.WordID, wordData.TelegramImageDocID)
		// Send the image as a document
		docToSend := &telebot.Document{
			File: telebot.File{FileID: wordData.TelegramImageDocID},
			// You can add a Caption here if you want, e.g., wordData.Title
		}
		if err := c.Send(docToSend); err != nil {
			log.Printf("[ViewHelper WARN] UserID %d, WordID %d: Sending cached image document (FileID: %s) failed: %v.", c.Sender().ID, wordData.WordID, wordData.TelegramImageDocID, err)
			// Optional: Fallback logic if you also have a URL specifically for the document version
			// and want to send it via URL if the FileID fails, then cache the new FileID.
			// For now, we'll just handle sending by existing FileID.
		} else {
			log.Printf("[ViewHelper INFO] UserID %d, WordID %d: Successfully sent image document with FileID: %s", c.Sender().ID, wordData.WordID, wordData.TelegramImageDocID)
		}
	} else {
		log.Printf("[ViewHelper INFO] UserID %d, WordID %d: No TelegramImageDocID available to send image as document.", c.Sender().ID, wordData.WordID)
	}
	// ---> END OF NEW BLOCK <---

	if imageFileIDToCache != "" || imageDocFileIDToCache != "" {
		// The interface for CacheTelegramFileIDForCourseWordImage is (courseWordID uint, imageFileID string, imageDocFileID string)
		// Assuming courseWordID here refers to the actual WordID for the purpose of this cache operation.
		// The service implementation needs to handle how to find the CourseWord record using this WordID and potentially CourseID from wordData.
		// This call matches the current interface signature.
		errCache := wordService.CacheTelegramFileIDForCourseWordImage(wordData.WordID, imageFileIDToCache, imageDocFileIDToCache)
		if errCache != nil {
			log.Printf("[ViewHelper WARN] UserID %d, WordID %d: Caching image FileIDs failed: %v", c.Sender().ID, wordData.WordID, errCache)
		}
	}

	for _, pron := range wordData.Pronunciations {

		if pron.TelegramVoiceID != "" {

			_, errSend := c.Bot().Send(c.Chat(), &telebot.Voice{File: telebot.File{FileID: pron.TelegramVoiceID}, Caption: pron.Region})

			if errSend != nil {
				log.Printf(
					"[ViewHelper WARN] UserID %d, PronunciationID %d: Sending cached voice (FileID: %s) failed: %v. Will try URL.",
					c.Sender().ID,
					pron.ID,
					pron.TelegramVoiceID,
					errSend,
				)
				if pron.AudioURL != "" {
					_, errSend = c.Bot().Send(c.Chat(), &telebot.Voice{File: telebot.FromURL(pron.AudioURL), Caption: pron.Region})
				}
			}
		}
	}
	return nil
}

// sendQuizQuestion sends a single quiz question to the user.
func sendQuizQuestion(c telebot.Context, qs *quiz.QuizState, quizService quiz.QuizService) error {
	if qs == nil {
		log.Printf("[ViewHelper ERROR] UserID %d: sendQuizQuestion called with nil QuizState", c.Sender().ID)
		return c.Send("خطا در بارگذاری سوال آزمون.", keyboards.MainMenu)
	}
	if qs.QuestionText == "" || len(qs.Options) == 0 {
		log.Printf("[ViewHelper ERROR] UserID %d, AttemptID %d: QuizState has no question text or options.", qs.UserID, qs.AttemptID)
		return c.Send("سوال آزمون به درستی بارگذاری نشد.", keyboards.MainMenu)
	}

	// ---> MODIFICATION HERE <---
	// Build the final formatted message text here.
	var textBuilder strings.Builder

	textBuilder.WriteString(fmt.Sprintf("\n%s", qs.QuestionText))
	textBuilder.WriteString("\n\n") // Add spacing

	for i, opt := range qs.Options {
		textBuilder.WriteString(fmt.Sprintf(">%d\\. %s\n", i+1, formatters.EscapeMarkdownV2(opt.Text)))

		seperatorText := "─────────────────────────"
		textBuilder.WriteString(fmt.Sprintf(">%s\n", formatters.EscapeMarkdownV2(seperatorText)))

	}

	questionText := textBuilder.String()
	// ---> END OF MODIFICATION <---

	optionsMarkup := keyboards.QuizQuestionOptionsKeyboard(qs.Options, qs.AttemptID)

	sentMsg, err := c.Bot().Send(
		c.Chat(),
		questionText,
		optionsMarkup,
		telebot.ModeMarkdownV2,
	)

	if err != nil {
		log.Printf("[ViewHelper ERROR] UserID %d, AttemptID %d: Sending quiz question failed (MDv2): %v. Retrying plain.", qs.UserID, qs.AttemptID, err)
		sentMsg, err = c.Bot().Send(c.Chat(), questionText, optionsMarkup)
		if err != nil {
			log.Printf("[ViewHelper ERROR] UserID %d, AttemptID %d: Sending plain text quiz question failed: %v", qs.UserID, qs.AttemptID, err)
			return c.Send("خطا در نمایش سوال آزمون.")
		}
	}

	if sentMsg != nil {
		errUpdate := quizService.UpdateQuizAttemptMessageID(qs.AttemptID, sentMsg.ID)
		if errUpdate != nil {
			log.Printf("[ViewHelper WARN] UserID %d, AttemptID %d: Failed to update message ID for quiz question (MsgID: %d): %v", qs.UserID, qs.AttemptID, sentMsg.ID, errUpdate)
		}
	}
	return nil
}

// editPreviousMessage edits a previously sent message.
// Changed newMarkup type to *telebot.ReplyMarkup
func editPreviousMessage(
	c telebot.Context,
	chatID int64,
	messageIDStr string,
	newText string, // Caller should ensure this is MarkdownV2 escaped if sending as Markdown
	newReplyMarkup *telebot.ReplyMarkup,
) {

	if messageIDStr == "" || chatID == 0 {
		log.Printf("[ViewHelper WARN] editPreviousMessage: messageIDStr or chatID is empty/zero. Cannot edit.")
		return
	}

	messageID, err := strconv.Atoi(messageIDStr)
	if err != nil {
		log.Printf("[ViewHelper WARN] Invalid messageIDStr for editing: %s. Error: %v", messageIDStr, err)
		return
	}

	editable := &telebot.Message{ID: messageID, Chat: &telebot.Chat{ID: chatID}}

	if newReplyMarkup != nil {
		// Case 1: A newReplyMarkup is provided.
		_, errMd := c.Bot().Edit(editable, newText, telebot.ModeMarkdownV2, newReplyMarkup)
		if errMd != nil {
			log.Printf("[ViewHelper WARN] Failed to edit message with MarkdownV2 (and new markup) (ChatID: %d, MsgID: %d). Error: %v. Text: %s. Trying plain text.", chatID, messageID, errMd, newText)
			// Fallback: Try to edit with plain text and the newReplyMarkup.
			// If newText was MarkdownV2 escaped, it will be sent as is here.
			// Consider unescaping newText if you want a cleaner plain text fallback.
			if _, errPlain := c.Bot().Edit(editable, newText, newReplyMarkup); errPlain != nil {
				log.Printf("[ViewHelper WARN] Failed to edit message with plain text (and new markup) (ChatID: %d, MsgID: %d). Error: %v.", chatID, messageID, errPlain)
			}
		}
	} else {
		// Case 2: newReplyMarkup is nil.
		_, errMd := c.Bot().Edit(editable, newText, telebot.ModeMarkdownV2)
		if errMd != nil {
			log.Printf("[ViewHelper WARN] Failed to edit message with MarkdownV2 (text-only) (ChatID: %d, MsgID: %d). Error: %v. Text: %s. Trying plain text.", chatID, messageID, errMd, newText)
			// Fallback: Try to edit with plain text only.
			// If newText was MarkdownV2 escaped, it will be sent as is here.
			if _, errPlain := c.Bot().Edit(editable, newText); errPlain != nil {
				log.Printf("[ViewHelper WARN] Failed to edit message with plain text (text-only) (ChatID: %d, MsgID: %d). Error: %v.", chatID, messageID, errPlain)
			}
		}
	}
}

// CheckAndInitiateReview checks if a daily review is due and starts it.
func CheckAndInitiateReview(c telebot.Context, appServices *services.AppServices, dbUser *models.User) (reviewHandled bool, err error) {
	if dbUser == nil {
		log.Println("[CheckAndInitiateReview] Received nil dbUser.")
		return false, errors.New("user not provided for review check")
	}

	now := time.Now()
	isNewDayForReview := true
	if !dbUser.LastReviewSessionCompletedAt.IsZero() {
		loc := now.Location()
		lastReviewDate := dbUser.LastReviewSessionCompletedAt.In(loc)
		currentDate := now.In(loc)
		if lastReviewDate.Year() == currentDate.Year() && lastReviewDate.YearDay() == currentDate.YearDay() {
			isNewDayForReview = false
		}
	}

	if !isNewDayForReview {
		log.Printf("[CheckAndInitiateReview] UserID %d: Review already completed today or not a new day. Last review: %s", dbUser.ID, dbUser.LastReviewSessionCompletedAt.Format(time.RFC3339))
		return false, nil
	}

	wordsStudiedDue, err := appServices.Word().GetWordsDueForReview(dbUser.ID, now)
	if err != nil {
		log.Printf("[CheckAndInitiateReview] UserID %d: Error getting words due for review: %v", dbUser.ID, err)
		return false, fmt.Errorf("failed to fetch words for review: %w", err)
	}

	if len(wordsStudiedDue) == 0 {
		log.Printf("[CheckAndInitiateReview] UserID %d: No words due for review today.", dbUser.ID)
		if err := appServices.User().MarkReviewSessionCompleted(dbUser.ID, now); err != nil {
			log.Printf("[CheckAndInitiateReview] UserID %d: Failed to mark review session as completed (no words due): %v", dbUser.ID, err)
		}
		return false, nil
	}

	log.Printf("[CheckAndInitiateReview] UserID %d: %d words due for review. Initiating review quiz.", dbUser.ID, len(wordsStudiedDue))
	// Send an initial message before creating the quiz
	// Use a simple reply keyboard, or no keyboard if the quiz itself will have one.
	c.Send(formatters.EscapeMarkdownV2(fmt.Sprintf("⏳ آماده‌سازی جلسه مرور روزانه برای %d کلمه...", len(wordsStudiedDue))),
		telebot.ModeMarkdownV2, &telebot.ReplyMarkup{RemoveKeyboard: true})

	reviewQuizState, errQuiz := appServices.Quiz().CreateReviewQuiz(dbUser.ID, wordsStudiedDue)
	if errQuiz != nil || reviewQuizState == nil || reviewQuizState.QuestionText == "" {
		log.Printf("[CheckAndInitiateReview] UserID %d: Error creating review quiz: %v. QuizState: %+v", dbUser.ID, errQuiz, reviewQuizState)
		SendServiceError(c, "creating review quiz", fmt.Errorf("could not start your review session: %v", errQuiz))
		if markErr := appServices.User().MarkReviewSessionCompleted(dbUser.ID, now); markErr != nil {
			log.Printf("[CheckAndInitiateReview] UserID %d: Also failed to mark review session completed after quiz creation error: %v", dbUser.ID, markErr)
		}
		return true, nil
	}

	if err := appServices.User().UpdateUserLastMenu(dbUser.ID, InReviewQuizMenuState(reviewQuizState.AttemptID)); err != nil {
		log.Printf("[CheckAndInitiateReview] UserID %d: Failed to update LastMenu to 'in_review_quiz:%d': %v", dbUser.ID, reviewQuizState.AttemptID, err)
	}

	if err := sendQuizQuestion(c, reviewQuizState, appServices.Quiz()); err != nil {
		log.Printf("[CheckAndInitiateReview] UserID %d: Error sending first review quiz question: %v", dbUser.ID, err)
		SendServiceError(c, "sending review quiz question", err)
		if markErr := appServices.User().MarkReviewSessionCompleted(dbUser.ID, now); markErr != nil {
			log.Printf("[CheckAndInitiateReview] UserID %d: Also failed to mark review session completed after send question error: %v", dbUser.ID, markErr)
		}
		return true, nil
	}
	return true, nil
}
