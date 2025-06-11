package notification

// Notifier defines a port for sending outbound notifications to users.
type Notifier interface {
	// Notify sends a message to a user identified by their internal application ID.
	Notify(userID uint, message string) error
}
