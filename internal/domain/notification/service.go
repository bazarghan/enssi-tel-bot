package notification

type Notifier interface {
	Notify(userID uint, message string) error
	NotifyWithMainMenu(userID uint, message string) error
}
