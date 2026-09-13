package bot

type Option func(*Bot)

func WithOwner(owner string) Option {
	return func(b *Bot) {
		b.owner = owner
	}
}

// WithLocale fixes the bot's response language regardless of the invoking
// guild's own PreferredLocale — for a single-owner bot whose owner wants a
// language other than their server's. Empty (the default) leaves per-guild
// detection in place.
func WithLocale(locale string) Option {
	return func(b *Bot) {
		b.locale = locale
	}
}
