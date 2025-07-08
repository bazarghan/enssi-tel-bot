package message

import (
	"context"
	"fmt"

	"github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/keyboards"
	sc "github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/stateconstants"
	ui "github.com/2000ostd/enssi-tel-bot/internal/adapter/telegram/uiconstants"
	"github.com/2000ostd/enssi-tel-bot/internal/domain/user"
	"github.com/2000ostd/enssi-tel-bot/internal/platform/observability/logger"
	adminCmd "github.com/2000ostd/enssi-tel-bot/internal/usecase/commands/admin"
	adminQueries "github.com/2000ostd/enssi-tel-bot/internal/usecase/queries/admin"
	"gopkg.in/telebot.v4"
)

// AdminHandler holds dependencies for admin-related message handlers.
type AdminHandler struct {
	logger    logger.Logger
	getStats  adminQueries.GetStatsHandler
	Broadcast adminCmd.BroadcastHandler
	userRepo  user.Repository
}

// NewAdminHandler creates a new admin handler.
func NewAdminHandler(
	appLogger logger.Logger,
	getStats adminQueries.GetStatsHandler,
	userRepo user.Repository,
) *AdminHandler {
	return &AdminHandler{
		logger:   appLogger,
		getStats: getStats,
		userRepo: userRepo,
	}
}

// Handle routes incoming admin-related messages based on user state.
func (h *AdminHandler) Handle(c telebot.Context, u user.User, userInput string) error {
	switch u.LastMenu {
	case sc.StateAdminPanel:
		if userInput == ui.BtnAdminStatsText {
			return h.handleGetStats(c)
		}
		if userInput == ui.BtnAdminBroadcastText {
			return h.handleInitiateAdminBroadcast(c, u)
		}
	case sc.StateAdminBroadcast:
		return h.handleAdminBroadcast(c, u, userInput)
	}
	return nil // Should not happen if routed correctly
}

func (h *AdminHandler) handleGetStats(c telebot.Context) error {
	stats, err := h.getStats.Handle(context.Background())
	if err != nil {
		h.logger.Error("Failed to get admin stats", "error", err)
		return c.Send("خطا در دریافت آمار.")
	}

	// Format the message and send it back to the admin
	statsMsg := fmt.Sprintf(
		"📊 *آمار ربات*\n\nتعداد کل کاربران: *%d*\nتعداد کل کلمات مطالعه شده: *%d*",
		stats.TotalUsers,
		stats.TotalWordsStudied, // <-- ADD THIS
	)
	return c.Send(statsMsg, telebot.ModeMarkdownV2)
}

func (h *AdminHandler) handleInitiateAdminBroadcast(c telebot.Context, u user.User) error {

	h.userRepo.UpdateLastMenu(context.Background(), u.ID, sc.StateAdminBroadcast)
	return c.Send("لطفا پیامی که میخواهید برای همه کاربران ارسال شود را وارد کنید:")

}

func (h *AdminHandler) handleAdminBroadcast(c telebot.Context, u user.User, userInput string) error {

	// Any text received in this state is the message to be broadcast
	broadcastCmd := adminCmd.BroadcastCommand{Message: userInput} // Assuming adminCmd alias

	recipients, err := h.Broadcast.Handle(context.Background(), broadcastCmd) // Assuming dependency is `broadcast`
	if err != nil {
		h.logger.Error("Broadcast failed", "error", err)
		return c.Send("ارسال پیام همگانی با خطا مواجه شد.")
	}

	// Send confirmation to the admin and return them to the admin panel
	confirmationMsg := fmt.Sprintf("✅ پیام شما برای %d کاربر ارسال شد.", recipients)
	c.Send(confirmationMsg)

	h.userRepo.UpdateLastMenu(context.Background(), u.ID, sc.StateAdminPanel)
	return c.Send("به پنل ادمین بازگشتید.", keyboards.AdminPanelKeyboard())

}
