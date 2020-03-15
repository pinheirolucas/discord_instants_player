package bot

type Option func(*Bot)

func WithOwner(owner string) Option {
	return func(b *Bot) {
		b.owner = owner
	}
}
