// Package i18n is a small, hand-rolled locale catalog for the backend's own
// user-facing text: the API's error messages and the Discord bot's command
// descriptions. Not a framework — with this few strings and no plural forms,
// x/text's language matching is all the negotiation this needs.
package i18n

import "golang.org/x/text/language"

// Supported lists this app's two locales, English first: language.NewMatcher
// falls back to index 0 for anything it can't confidently match, and English
// is the right default for a log line or for any client that isn't this
// app's own UI.
var Supported = []language.Tag{
	language.AmericanEnglish,
	language.BrazilianPortuguese,
}

var matcher = language.NewMatcher(Supported)

// Match resolves an arbitrary locale string — a Discord guild's
// PreferredLocale, a bot.locale config value — to one of the two locales
// this app ships. Unparseable or unrecognized input falls back to English.
func Match(locale string) language.Tag {
	tag, err := language.Parse(locale)
	if err != nil {
		return Supported[0]
	}

	// The index, not the returned tag: matcher.Match decorates its result
	// with a region extension recalling the input (e.g. matching "pt-PT"
	// against Supported's "pt-BR" comes back "pt-BR-u-rg-ptzzzz"), which
	// would never compare equal to the bare Supported tags Text keys off of.
	_, index, _ := matcher.Match(tag)
	return Supported[index]
}

// MatchAcceptLanguage is Match for an HTTP Accept-Language header, which can
// carry several weighted tags rather than one.
func MatchAcceptLanguage(header string) language.Tag {
	tags, _, err := language.ParseAcceptLanguage(header)
	if err != nil || len(tags) == 0 {
		return Supported[0]
	}

	_, index, _ := matcher.Match(tags...)
	return Supported[index]
}

var catalogs = map[language.Tag]map[string]string{
	language.AmericanEnglish:     enUS,
	language.BrazilianPortuguese: ptBR,
}

// Text resolves key in tag's catalog, falling back to English and finally to
// key itself, so a typo or a not-yet-cataloged key surfaces as an odd string
// instead of a blank response.
func Text(tag language.Tag, key string) string {
	if text, ok := catalogs[tag][key]; ok {
		return text
	}
	if text, ok := catalogs[Supported[0]][key]; ok {
		return text
	}
	return key
}
