package handlers

import (
	"github.com/2000ostd/enssi-tel-bot/internal/bot/keyboards"
	"gopkg.in/telebot.v4"
)

func RegisterHandlers(bot *telebot.Bot) {
	bot.Handle("/start", handleStart)
	bot.Handle(&keyboards.BtnReturnToMainMenu, handleBtnReturnToMainMenuClicked)
	bot.Handle(&keyboards.BtnStartLearning, handleBtnStartLearningClicked)
	bot.Handle(&keyboards.Btn504, handleBtn504Clicked)
	bot.Handle(&keyboards.Btn1100, handleBtn1100Clicked)
	bot.Handle(&keyboards.BtnStartCourse, handleBtnStartCourseClicked)
	bot.Handle(&keyboards.BtnNextWord, handleBtnNextWordClicked)
	bot.Handle(&keyboards.BtnPrevMenuCourseDetails, handleBtnPrevMenuCourseDetailsClicked)
	bot.Handle(&keyboards.BtnPrevMenuCourse, handleBtnPrevMenuCourseClicked)
}

func handleStart(ctx telebot.Context) error {
	menu := keyboards.Main()

	desciption := "به ربات زبان *اِنسی* خوش اومدی 🙌\n\nواسه یاد گرفتن زبان انگیزه نداری ؟\nاز کلاسا و دوره های مختلف نتیجه نگرفتی ؟\nبرنامه ریزی واسه خوندن و مرور کلمات سخته ؟\nکلماتی که می خونی رو مدام یادت می ره ؟\n\n اگه اینطوره پس جای درستی اومدی، اینجا ما با روشای مختلف مثل گیمیفیکیشن، مرورای دوره ای و زمان بندی شده، کوییزای متنوع، رقابت و \\.\\.\\. بهت انگیزه می دیم و کمک می کنیم به هدفت تو یادگیری زبان برسی و تا نرسیدی ول کنتم نیستیم 😅\n\n پس اگه آماده ای بزن بریممم 🤝"

	return ctx.Send(desciption, menu, telebot.ModeMarkdownV2)
}

func handleBtnStartLearningClicked(ctx telebot.Context) error {
	menu := keyboards.SelectCourse()

	desciption := "حالا که تصمیم گرفتی یادگیری رو شروع کنی تو این مرحله باید مجموعه ای که می خوای رو انتخاب کنی و ادامه بدی 🙂"

	return ctx.Send(desciption, menu)
}

func handleBtnReturnToMainMenuClicked(ctx telebot.Context) error {
	menu := keyboards.Main()

	return ctx.Send("Please continue with an option", menu)
}
