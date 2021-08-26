package fsutil

import (
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/h2non/filetype"
	"github.com/h2non/filetype/matchers"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
)

var (
	ErrNotFound              = errors.New("resource not found")
	ErrUnsuportedAudioFormat = errors.New("unduported audio format")
)

func GetFromCache(link string) (*os.File, error) {
	cdr, err := GetCacheDirOrCreate()
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

	fr, err := http.Get(link)
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

	fileKind, err := filetype.MatchReader(fr.Body)
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

	if _, err := io.Copy(file, fr.Body); err != nil {
		return nil, err
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	return file, nil
}

func GetCacheDirOrCreate() (string, error) {
	h, err := homedir.Dir()
	if err != nil {
		return "", err
	}

	cdr := filepath.Join(h, ".instants")
	if err := os.MkdirAll(cdr, os.ModePerm); err != nil {
		return "", err
	}

	return cdr, nil
}
