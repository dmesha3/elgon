package elgon

// Plugin defines an extension hook that can configure the app at startup.
type Plugin interface {
	Name() string
	Init(*App) error
}
