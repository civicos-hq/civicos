package uploads

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// ErrNotFound is returned by Storage.Open when the named object does not exist.
var ErrNotFound = errors.New("upload not found")

// File is the readable handle Storage.Open hands back. ReadSeeker (not just
// Reader) is deliberate: HTTP Range requests are what let a browser scrub a
// video without downloading the whole file, and http.ServeContent needs to
// seek to satisfy them. *os.File already satisfies this interface.
type File interface {
	io.ReadSeekCloser
	Stat() (fs.FileInfo, error)
}

// Storage is where uploaded media lives. The handlers only ever talk to this
// interface, so moving to S3/R2/GCS later is a new implementation of these two
// methods plus a wiring change in main.go — not a rewrite of the upload and
// serve paths.
//
// Names passed in are the generated storage names (UUID + extension), never
// caller-supplied filenames. Implementations must still refuse anything that
// isn't a plain name — see safeStoredName.
type Storage interface {
	// Save writes src under name, replacing any existing object with that name.
	Save(name string, src io.Reader) error
	// Open returns the stored object for reading. It returns ErrNotFound if
	// no object exists under name.
	Open(name string) (File, error)
}

// LocalStorage keeps uploads on the machine's own filesystem.
//
// NOTE ON DURABILITY: this directory is NOT durable across container restarts
// unless a persistent volume is mounted at it. On an ephemeral filesystem
// (Cloud Run, a plain `docker run`, most PaaS default configs) every uploaded
// image and video disappears the moment the container is replaced, while the
// URLs stay in the database — issues end up pointing at 404s. Mount a volume,
// or swap this implementation for object storage, before this is load-bearing.
type LocalStorage struct {
	dir string
}

// NewLocalStorage creates the backing directory if it does not exist.
func NewLocalStorage(dir string) (*LocalStorage, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &LocalStorage{dir: dir}, nil
}

func (s *LocalStorage) Save(name string, src io.Reader) error {
	path, err := s.resolve(name)
	if err != nil {
		return err
	}
	dst, err := os.Create(path)
	if err != nil {
		return err
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		return err
	}
	return dst.Sync()
}

func (s *LocalStorage) Open(name string) (File, error) {
	path, err := s.resolve(name)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return f, nil
}

// resolve turns a storage name into a path inside the backing directory.
// The name check is defence in depth — the handlers validate too, but a
// storage implementation that joins paths must never trust its caller.
func (s *LocalStorage) resolve(name string) (string, error) {
	if !safeStoredName(name) {
		return "", ErrNotFound
	}
	return filepath.Join(s.dir, name), nil
}
