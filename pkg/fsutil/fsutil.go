package fsutil

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/h2non/filetype"
	"github.com/h2non/filetype/matchers"
	"github.com/pkg/errors"
)

var (
	ErrNotFound              = errors.New("resource not found")
	ErrUnsuportedAudioFormat = errors.New("unduported audio format")
)

// Cache resolves instant links to files on disk, downloading them on first use.
// Client and Dir exist so tests can point it at an httptest.Server and a
// temporary directory; both fall back to the production defaults when unset.
type Cache struct {
	Client *http.Client
	Dir    string
}

// Default backs the package-level functions the rest of the app calls.
var Default = &Cache{}

func GetFromCache(link string) (*os.File, error) {
	return Default.Get(link)
}

func GetCacheDirOrCreate() (string, error) {
	return Default.DirOrCreate()
}

func (c *Cache) client() *http.Client {
	if c.Client != nil {
		return c.Client
	}

	return http.DefaultClient
}

func (c *Cache) Get(link string) (*os.File, error) {
	cdr, err := c.DirOrCreate()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cache dir")
	}

	fname := filepath.Join(cdr, fmt.Sprintf("%x.mp3", md5.Sum(([]byte(link)))))
	ifile, err := os.Open(fname)
	switch {
	case err == nil:
		return ifile, nil
	case os.IsNotExist(err):
		// continue
	default:
		return nil, err
	}

	fr, err := c.client().Get(link)
	if err != nil {
		return nil, err
	}
	defer fr.Body.Close()

	switch fr.StatusCode {
	case http.StatusOK:
		// continue
	case http.StatusNotFound:
		return nil, ErrNotFound
	default:
		return nil, errors.Errorf("failed to fetch instant: %d", fr.StatusCode)
	}

	// filetype.MatchReader reads up to 8KB off the reader to sniff the type and
	// does not put it back, so copying fr.Body afterwards writes only whatever
	// was left — nothing at all for a clip smaller than the sniff buffer. Read
	// the head ourselves and stitch it back on before copying.
	head := make([]byte, 8192)
	n, err := io.ReadFull(fr.Body, head)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return nil, errors.Wrap(err, "failed to read instant")
	}
	head = head[:n]

	fileKind, err := filetype.Match(head)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get instant info")
	}

	if fileKind != matchers.TypeMp3 {
		return nil, ErrUnsuportedAudioFormat
	}

	file, err := os.Create(fname)
	if err != nil {
		return nil, err
	}

	if _, err := io.Copy(file, io.MultiReader(bytes.NewReader(head), fr.Body)); err != nil {
		return nil, err
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	return file, nil
}

func (c *Cache) DirOrCreate() (string, error) {
	cdr := c.Dir
	if cdr == "" {
		h, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}

		cdr = filepath.Join(h, ".instants")
	}

	if err := os.MkdirAll(cdr, os.ModePerm); err != nil {
		return "", err
	}

	return cdr, nil
}
