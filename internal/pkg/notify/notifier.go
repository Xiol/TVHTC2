package notify

type Notifier interface {
	Fire() error
}
