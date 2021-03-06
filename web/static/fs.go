package static

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"sync"
	"time"
)

type _escLocalFS struct{}

var _escLocal _escLocalFS

type _escStaticFS struct{}

var _escStatic _escStaticFS

type _escDir struct {
	fs   http.FileSystem
	name string
}

type _escFile struct {
	compressed string
	size       int64
	local      string
	isDir      bool

	data []byte
	once sync.Once
	name string
}

func (_escLocalFS) Open(name string) (http.File, error) {
	f, present := _escData[path.Clean(name)]
	if !present {
		return nil, os.ErrNotExist
	}
	return os.Open(f.local)
}

func (_escStaticFS) prepare(name string) (*_escFile, error) {
	f, present := _escData[path.Clean(name)]
	if !present {
		return nil, os.ErrNotExist
	}
	var err error
	f.once.Do(func() {
		f.name = path.Base(name)
		if f.size == 0 {
			return
		}
		var gr *gzip.Reader
		b64 := base64.NewDecoder(base64.StdEncoding, bytes.NewBufferString(f.compressed))
		gr, err = gzip.NewReader(b64)
		if err != nil {
			return
		}
		f.data, err = ioutil.ReadAll(gr)
	})
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (fs _escStaticFS) Open(name string) (http.File, error) {
	f, err := fs.prepare(name)
	if err != nil {
		return nil, err
	}
	return f.File()
}

func (dir _escDir) Open(name string) (http.File, error) {
	return dir.fs.Open(dir.name + name)
}

func (f *_escFile) File() (http.File, error) {
	type httpFile struct {
		*bytes.Reader
		*_escFile
	}
	return &httpFile{
		Reader:   bytes.NewReader(f.data),
		_escFile: f,
	}, nil
}

func (f *_escFile) Close() error {
	return nil
}

func (f *_escFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, nil
}

func (f *_escFile) Stat() (os.FileInfo, error) {
	return f, nil
}

func (f *_escFile) Name() string {
	return f.name
}

func (f *_escFile) Size() int64 {
	return f.size
}

func (f *_escFile) Mode() os.FileMode {
	return 0
}

func (f *_escFile) ModTime() time.Time {
	return time.Time{}
}

func (f *_escFile) IsDir() bool {
	return f.isDir
}

func (f *_escFile) Sys() interface{} {
	return f
}

// FS returns a http.Filesystem for the embedded assets. If useLocal is true,
// the filesystem's contents are instead used.
func FS(useLocal bool) http.FileSystem {
	if useLocal {
		return _escLocal
	}
	return _escStatic
}

// Dir returns a http.Filesystem for the embedded assets on a given prefix dir.
// If useLocal is true, the filesystem's contents are instead used.
func Dir(useLocal bool, name string) http.FileSystem {
	if useLocal {
		return _escDir{fs: _escLocal, name: name}
	}
	return _escDir{fs: _escStatic, name: name}
}

// FSByte returns the named file from the embedded assets. If useLocal is
// true, the filesystem's contents are instead used.
func FSByte(useLocal bool, name string) ([]byte, error) {
	if useLocal {
		f, err := _escLocal.Open(name)
		if err != nil {
			return nil, err
		}
		return ioutil.ReadAll(f)
	}
	f, err := _escStatic.prepare(name)
	if err != nil {
		return nil, err
	}
	return f.data, nil
}

// FSMustByte is the same as FSByte, but panics if name is not present.
func FSMustByte(useLocal bool, name string) []byte {
	b, err := FSByte(useLocal, name)
	if err != nil {
		panic(err)
	}
	return b
}

// FSString is the string version of FSByte.
func FSString(useLocal bool, name string) (string, error) {
	b, err := FSByte(useLocal, name)
	return string(b), err
}

// FSMustString is the string version of FSMustByte.
func FSMustString(useLocal bool, name string) string {
	return string(FSMustByte(useLocal, name))
}

var _escData = map[string]*_escFile{

	"/static/css/application.css": {
		local: "web/assets/static/css/application.css",
		size:  107,
		compressed: `
H4sIAAAJbogA/9JLKsnTTUwuyczPU6jmUgCC3MSi9Mw83ZL8AisFY4OCCmuwaEFiSkpmXrqVgqFFQYWC
kQVMPC0/r0S3OLMq1UrByAgmmJRflJJapFuUmJJZWmylAFZcywUIAAD//xiYznRrAAAA
`,
	},

	"/static/js/garagepi.js": {
		local: "web/assets/static/js/garagepi.js",
		size:  878,
		compressed: `
H4sIAAAJbogA/4xS3WqDMBS+9ynOMqERimXXRXYzGGxlvWhfINXEhblE4rHbKH33JTHWtnPSC8Ek30/y
nY+0DYcGjcyRLKMopoXO20+uMEkNZ8UPFa3KUWpFk0MUAeyZgXiHaiXLd4QMYkru+yVJlj2kcuu1sgB6
QqfIv5EmkGVAtq1RsBYCzom9FaAuy4o/M8NK/qS1saSDPQeI01o3SMmC1XKxf1h0QEcHOF5KWINVd4l/
2f6Sjw0y5JlWZA41Mw33rI3bnJQV4jZdIW4SvkLQgiHr9d2/izr1oJfN+q07XvrTIWq3mYZH+0ABpAAa
AL0awNVERobhUEfgle3GNEldcQbXO38b/5hXpb9G7WujazorZMN2FS9mc0DT8pNUkAsNcz3YhnGnto4k
r2T+YbMdChoc/rbHx911bDCf1BjN7XL4oyld1O7sIfbffb8BAAD//zKtiT1uAwAA
`,
	},

	"/": {
		isDir: true,
		local: "web/assets",
	},

	"/static": {
		isDir: true,
		local: "web/assets/static",
	},

	"/static/css": {
		isDir: true,
		local: "web/assets/static/css",
	},

	"/static/js": {
		isDir: true,
		local: "web/assets/static/js",
	},
}
