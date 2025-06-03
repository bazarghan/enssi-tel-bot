package handlers

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/2000ostd/enssi-tel-bot/internal/bot/formatters"
	"github.com/2000ostd/enssi-tel-bot/internal/bot/keyboards"
	"github.com/2000ostd/enssi-tel-bot/internal/services"
	"github.com/2000ostd/enssi-tel-bot/internal/services/course"
	"github.com/2000ostd/enssi-tel-bot/internal/services/quiz"
	"github.com/2000ostd/enssi-tel-bot/internal/services/word"
	"gopkg.in/telebot.v4"
)

// sendServiceError sends a generic error message to the user.
func sendServiceError(c telebot.Context, operation string, err error) error {
	log.Printf("[Service ERROR] UserID %d performing %s: %v", c.Sender().ID, operation, err)
	return c.Send("متاسفانه در انجام عملیات مشکلی پیش آمد. لطفا دوباره تلاش کنید.", keyboards.MainMenu)
}

// sendLearningContext processes the LearningContext DTO and sends the appropriate content (word or quiz).
func sendLearningContext(c telebot.Context, lc *course.LearningContext, appServices *services.AppServices) error {
	if lc == nil {
		log.Printf("[ViewHelper ERROR] UserID %d: Received nil LearningContext", c.Sender().ID)
		return c.Send("محتوایی برای نمایش یافت نشد. لطفا به منوی اصلی بازگردید.", keyboards.MainMenu)
	}

	var menuState string
	if lc.IsQuizDue && lc.QuizState != nil {
		menuState = InQuizMenuState(lc.QuizState.AttemptID)
	} else if lc.WordToDisplay != nil {
		menuState = InCourseMenuState(lc.CourseID)
	} else if lc.IsCourseEnded {
		menuState = StateCourseDetailsBase // Or MenuStateCourseList
	} else {
		menuState = StateMain // Fallback
	}

	currentUser, err := appServices.User().GetOrCreateUserByTelegramID(c.Sender().ID, c.Sender().Username, c.Sender().FirstName, c.Sender().LastName)
	if err != nil {
		return sendServiceError(c, "updating menu state (fetch user)", err)
	}
	err = appServices.User().UpdateUserLastMenu(currentUser.ID, menuState)
	if err != nil {
		log.Printf("[ViewHelper WARN] UserID %d: Failed to update LastMenu to '%s': %v", lc.UserID, menuState, err)
	}

	if lc.IsCourseEnded {
		// c.Send here is fine
		return c.Send("🎉 "+formatters.EscapeMarkdownV2(lc.MessageToUser), keyboards.CourseCompletedKeyboard())
	}

	if lc.IsQuizDue {
		if lc.QuizState == nil {
			log.Printf("[ViewHelper ERROR] UserID %d: IsQuizDue is true but QuizState is nil.", lc.UserID)
			// c.Send here is fine
			return c.Send("آماده سازی آزمون با مشکل مواجه شد.", keyboards.MainMenu)
		}

		quizTimeReplyKeyboard := &telebot.ReplyMarkup{ResizeKeyboard: true}
		quizTimeReplyKeyboard.Reply(quizTimeReplyKeyboard.Row(keyboards.BtnReturnToMainMenu))

		if err := c.Send(lc.MessageToUser, quizTimeReplyKeyboard); err != nil {
			log.Printf("[ViewHelper WARN] UserID %d: Failed to send Quiz Intro Message with quizTimeReplyKeyboard: %v", lc.UserID, err)
		}

		return sendQuizQuestion(c, lc.QuizState, appServices.Quiz())
	}

	if lc.WordToDisplay != nil {
		return sendWordDisplay(c, lc.WordToDisplay, appServices.Word())
	}

	log.Printf("[ViewHelper INFO] UserID %d: No specific content in LearningContext (not ended, no quiz, no word). Progress: %d", lc.UserID, lc.UserProgress)
	// c.Send here is fine
	return c.Send("محتوای بعدی در دسترس نیست. لطفا به منوی اصلی بروید.", keyboards.MainMenu)
}

// sendWordDisplay sends the word details (text, image, audio).
// Removed userService and courseID as they weren't directly used here after changes.
func sendWordDisplay(c telebot.Context, wordData *word.WordDisplayData, wordService word.WordService) error {
	// 1. Send Formatted Text (already MarkdownV2 from service)
	// c.Send here is fine
	if strings.TrimSpace(wordData.FormattedText) != "" {
		if err := c.Send(wordData.FormattedText, telebot.ModeMarkdownV2, keyboards.InCourseNavigationKeyboard()); err != nil {
			log.Printf("[ViewHelper ERROR] UserID %d, WordID %d: Sending MarkdownV2 text failed: %v. Retrying plain.", c.Sender().ID, wordData.WordID, err)
			if errPlain := c.Send(formatters.EscapeMarkdownV2(wordData.FormattedText), keyboards.InCourseNavigationKeyboard()); errPlain != nil {
				log.Printf("[ViewHelper ERROR] UserID %d, WordID %d: Sending plain text fallback failed: %v", c.Sender().ID, wordData.WordID, errPlain)
				return c.Send("خطا در نمایش اطلاعات کلمه.")
			}
		}
	} else {
		if err := c.Send("کلمه بارگذاری شد.", keyboards.InCourseNavigationKeyboard()); err != nil {
			log.Printf("[ViewHelper ERROR] UserID %d, WordID %d: Failed to send placeholder for empty word text: %v", c.Sender().ID, wordData.WordID, err)
		}
	}

	var imageFileIDToCache, imageDocFileIDToCache string

	// 2. Send Image using c.Bot().Send to get the message if sent by URL
	if wordData.TelegramImageID != "" {
		// Use c.Bot().Send if you needed the message, but for just sending, c.Send is simpler if error is enough
		if err := c.Send(&telebot.Photo{File: telebot.File{FileID: wordData.TelegramImageID}}); err != nil {
			log.Printf("[ViewHelper WARN] UserID %d, WordID %d: Sending cached image photo (FileID: %s) failed: %v. Will try URL.", c.Sender().ID, wordData.WordID, wordData.TelegramImageID, err)
			if wordData.ImageURL != "" {
				sentMsg, errURL := c.Bot().Send(c.Chat(), &telebot.Photo{File: telebot.FromURL(wordData.ImageURL)}) // CORRECTED
				if errURL != nil {
					log.Printf("[ViewHelper WARN] UserID %d, WordID %d: Sending image by URL (%s) failed: %v", c.Sender().ID, wordData.WordID, wordData.ImageURL, errURL)
				} else if sentMsg != nil && sentMsg.Photo != nil {
					imageFileIDToCache = sentMsg.Photo.FileID
				}
			}
		}
	} else if wordData.ImageURL != "" {
		sentMsg, err := c.Bot().Send(c.Chat(), &telebot.Photo{File: telebot.FromURL(wordData.ImageURL)}) // CORRECTED
		if err != nil {
			log.Printf("[ViewHelper WARN] UserID %d, WordID %d: Sending image by URL (%s) failed: %v", c.Sender().ID, wordData.WordID, wordData.ImageURL, err)
		} else if sentMsg != nil && sentMsg.Photo != nil {
			imageFileIDToCache = sentMsg.Photo.FileID
		}
	}

	if wordData.TelegramImageDocID != "" {
		if err := c.Send(&telebot.Document{File: telebot.File{FileID: wordData.TelegramImageDocID}}); err != nil { // c.Send fine here
			log.Printf("[ViewHelper WARN] UserID %d, WordID %d: Sending cached image document (FileID: %s) failed: %v.", c.Sender().ID, wordData.WordID, wordData.TelegramImageDocID, err)
		}
	}

	if imageFileIDToCache != "" || imageDocFileIDToCache != "" {
		log.Printf("[ViewHelper INFO] Attempting to cache image FileIDs. Using WordID %d as placeholder for courseWordID. ImageFileID: '%s', ImageDocFileID: '%s'", wordData.WordID, imageFileIDToCache, imageDocFileIDToCache)
		errCache := wordService.CacheTelegramFileIDForCourseWordImage(wordData.WordID, imageFileIDToCache, imageDocFileIDToCache)
		if errCache != nil {
			log.Printf("[ViewHelper WARN] UserID %d, WordID %d: Caching image FileIDs failed: %v", c.Sender().ID, wordData.WordID, errCache)
		}
	}

	// 3. Send Audio Pronunciations using c.Bot().Send
	for _, pron := range wordData.Pronunciations {
		var sentAudioMsg *telebot.Message
		var pronFileIDToCache string
		var errSend error

		if pron.TelegramVoiceID != "" {
			sentAudioMsg, errSend = c.Bot().Send(c.Chat(), &telebot.Voice{File: telebot.File{FileID: pron.TelegramVoiceID}, Caption: pron.Region}) // CORRECTED
			if errSend != nil {
				log.Printf("[ViewHelper WARN] UserID %d, PronunciationID %d: Sending cached voice (FileID: %s) failed: %v. Will try URL.", c.Sender().ID, pron.ID, pron.TelegramVoiceID, errSend)
				if pron.AudioURL != "" {
					sentAudioMsg, errSend = c.Bot().Send(c.Chat(), &telebot.Voice{File: telebot.FromURL(pron.AudioURL), Caption: pron.Region}) // CORRECTED
					if errSend != nil {
						log.Printf("[ViewHelper WARN] UserID %d, PronunciationID %d: Sending voice by URL (%s) failed: %v", c.Sender().ID, pron.ID, pron.AudioURL, errSend)
						continue
					}
				} else {
					continue
				}
			}
		} else if pron.AudioURL != "" {
			sentAudioMsg, errSend = c.Bot().Send(c.Chat(), &telebot.Voice{File: telebot.FromURL(pron.AudioURL), Caption: pron.Region}) // CORRECTED
			if errSend != nil {
				log.Printf("[ViewHelper WARN] UserID %d, PronunciationID %d: Sending voice by URL (%s) failed: %v", c.Sender().ID, pron.ID, pron.AudioURL, errSend)
				continue
			}
		}

		if errSend == nil && sentAudioMsg != nil && sentAudioMsg.Voice != nil && sentAudioMsg.Voice.FileID != "" && sentAudioMsg.Voice.FileID != pron.TelegramVoiceID {
			pronFileIDToCache = sentAudioMsg.Voice.FileID
			errCache := wordService.CacheTelegramFileIDForPronunciation(pron.ID, pronFileIDToCache)
			if errCache != nil {
				log.Printf("[ViewHelper WARN] UserID %d, PronunciationID %d: Caching voice FileID (%s) failed: %v", c.Sender().ID, pron.ID, pronFileIDToCache, errCache)
			}
		}
		break
	}
	return nil
}

// sendQuizQuestion sends a single quiz question to the user.
func sendQuizQuestion(c telebot.Context, qs *quiz.QuizState, quizService quiz.QuizService) error {
	if qs.QuestionText == "" || len(qs.Options) == 0 {
		log.Printf("[ViewHelper ERROR] UserID %d, AttemptID %d: QuizState has no question text or options.", qs.UserID, qs.AttemptID)
		return c.Send("سوال آزمون به درستی بارگذاری نشد.", keyboards.InCourseNavigationKeyboard())
	}

	if qs.MessageToUser != "" {
		if err := c.Send(qs.MessageToUser); err != nil { // c.Send fine here
			log.Printf("[ViewHelper WARN] UserID %d: Failed to send QuizState.MessageToUser: %v", qs.UserID, err)
		}
	}

	questionText := qs.QuestionText
	optionsMarkup := keyboards.QuizQuestionOptionsKeyboard(qs.Options, qs.AttemptID)

	// Use c.Bot().Send to get the message ID
	sentMsg, err := c.Bot().Send(
		c.Chat(),
		questionText,
		optionsMarkup,
		telebot.ModeMarkdownV2,
	) // CORRECTED

	if err != nil {
		log.Printf("[ViewHelper ERROR] UserID %d, AttemptID %d: Sending quiz question failed (MDv2): %v. Retrying plain.", qs.UserID, qs.AttemptID, err)
		sentMsg, err = c.Bot().Send(
			c.Chat(),
			questionText,
			optionsMarkup,
		)

		if err != nil {
			log.Printf("[ViewHelper ERROR] UserID %d, AttemptID %d: Sending plain text quiz question failed: %v", qs.UserID, qs.AttemptID, err)
			return c.Send("خطا در نمایش سوال آزمون.") // c.Send fine here
		}
	}

	if sentMsg != nil { // Check if message was successfully sent
		errUpdate := quizService.UpdateQuizAttemptMessageID(qs.AttemptID, sentMsg.ID)
		if errUpdate != nil {
			log.Printf("[ViewHelper WARN] UserID %d, AttemptID %d: Failed to update message ID for quiz question (MsgID: %d): %v", qs.UserID, qs.AttemptID, sentMsg.ID, errUpdate)
		}
	}
	return nil
}

// sendQuizResult and editPreviousMessage remain unchanged as they were either using c.Send correctly
// or c.Bot().Edit correctly.
// ... (rest of the file: sendQuizResult, editPreviousMessage - should be okay as they are)
func sendQuizResult(c telebot.Context, qr *quiz.QuizResult, appServices *services.AppServices) error {
	if qr == nil {
		log.Printf("[ViewHelper ERROR] UserID %d: Received nil QuizResult", c.Sender().ID)
		return c.Send("نتیجه آزمون یافت نشد.", keyboards.MainMenu)
	}

	resultMessage := formatters.EscapeMarkdownV2(qr.ResultMessage)
	reviewText := formatters.EscapeMarkdownV2(qr.ReviewText)

	fullMessage := resultMessage
	if reviewText != "" {
		fullMessage = fmt.Sprintf("%s\n\n%s", resultMessage, reviewText)
	}

	if err := c.Send(fullMessage, telebot.ModeMarkdownV2); err != nil { // c.Send is fine
		log.Printf("[ViewHelper WARN] UserID %d, AttemptID %d: Failed to send formatted quiz result. Sending plain. Error: %v", qr.UserID, qr.AttemptID, err)
		if errPlain := c.Send(fullMessage); errPlain != nil {
			log.Printf("[ViewHelper ERROR] UserID %d, AttemptID %d: Failed to send plain quiz result: %v", qr.UserID, qr.AttemptID, errPlain)
			return c.Send("خطا در نمایش نتیجه آزمون.")
		}
	}

	nextLearningContext, errCourseCont := appServices.Course().HandleQuizCompletion(
		qr.UserID,
		qr.CourseID,
		qr,
	)

	if errCourseCont != nil {
		log.Printf("[ViewHelper ERROR] UserID %d, CourseID %d: Error handling quiz completion in course service: %v", qr.UserID, qr.CourseID, errCourseCont)
		return c.Send("مشکلی در ادامه دادن پس از آزمون پیش آمد. لطفا از منوی اصلی دوباره شروع کنید.", keyboards.MainMenu)
	}

	return sendLearningContext(c, nextLearningContext, appServices)
}

func editPreviousMessage(c telebot.Context, chatID int64, messageIDStr string, newText string, newMarkup ...*telebot.ReplyMarkup) {
	if messageIDStr == "" || chatID == 0 {
		return
	}

	messageID, err := strconv.Atoi(messageIDStr)
	if err != nil {
		log.Printf("[ViewHelper WARN] Invalid messageIDStr for editing: %s", messageIDStr)
		return
	}

	editable := &telebot.Message{ID: messageID, Chat: &telebot.Chat{ID: chatID}}
	var opts []interface{}
	if len(newMarkup) > 0 && newMarkup[0] != nil {
		opts = append(opts, newMarkup[0])
	} else {
		opts = append(opts, &telebot.ReplyMarkup{})
	}

	// c.Bot().Edit returns (*telebot.Message, error)
	if _, err := c.Bot().Edit(editable, newText, opts...); err != nil {
		log.Printf("[ViewHelper WARN] Failed to edit message (ChatID: %d, MsgID: %d): %v", chatID, messageID, err)
	}
}
