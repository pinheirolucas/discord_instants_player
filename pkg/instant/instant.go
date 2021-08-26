package instant

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/url"

	"github.com/pkg/errors"

	"github.com/pinheirolucas/discord_instants_player/pkg/fsutil"
)

type Instant struct {
	Exists  bool   `json:"exists,omitempty"`
	Content string `json:"content,omitempty"`
}

func GetPlayable(link string) (*Instant, error) {
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
	_, err := url.ParseRequestURI(link)
	if err != nil {
		return false
	}

	u, err := url.Parse(link)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}

	return true
}
