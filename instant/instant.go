package instant

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"github.com/pkg/errors"

	"github.com/pinheirolucas/discord_instants_player/fsutil"
)

const instantsURLPrefix = "https://www.myinstants.com/media/sounds/"

type Instant struct {
	Exists  bool
	Content string
}

func GetPlayableInstant(link string) (*Instant, error) {
	if !IsLinkValid(link) {
		return nil, errors.Errorf("invalid link: %s", link)
	}

	w := new(bytes.Buffer)
	instant := new(Instant)

	f, err := fsutil.GetFromCache(link)
	switch err {
	case nil:
		// continue
	case fsutil.ErrNotFound:
		return instant, nil
	default:
		return nil, err
	}
	defer f.Close()

	enc := base64.NewEncoder(base64.StdEncoding, w)
	if _, err := io.Copy(enc, f); err != nil {
		return nil, errors.Wrap(err, "genarating base64 hash")
	}

	instant.Exists = true
	instant.Content = fmt.Sprintf("data:audio/mp3;base64,%s", w.String())

	return instant, nil
}

func IsLinkValid(link string) bool {
	return strings.HasPrefix(link, instantsURLPrefix)
}
