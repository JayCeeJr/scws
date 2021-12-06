package s3

import (
	"context"
	"fmt"
	"log"
	"mime"
	"net/http"
	"net/textproto"
	"path"
	"path/filepath"
	"scws/common"
	"scws/common/config"
	"strings"
	"time"

	"github.com/araddon/gou"
	"github.com/lytics/cloudstorage"
	"github.com/lytics/cloudstorage/awss3"
)

func New(c *config.Config) (*S3Storage, error) {
	s3Config := config.S3Config{}
	s3Config.ParseEnv()
	if c.IsVaultEnabled() {
		err := s3Config.GetVaultSecrets(c.VaultPaths)
		if err != nil {
			log.Println("s3.New", err)
			return nil, err
		}
		log.Println("vault secrets have been loaded successfully")
	}
	s := S3Storage{
		config: &s3Config,
	}

	conf := &cloudstorage.Config{
		Type:       awss3.StoreType,
		AuthMethod: "SharedCredentials",
		Bucket:     s.config.Bucket,
		Settings:   make(gou.JsonHelper),
		Region:     s3Config.AwsRegion,
	}
	conf.Settings[awss3.ConfKeyAccessKey] = s3Config.AwsAccessKeyID
	conf.Settings[awss3.ConfKeyAccessSecret] = s3Config.AwsSecretAccessKey
	store, err := cloudstorage.NewStore(conf)
	if err != nil {
		log.Println("s3.New", err.Error())
		return nil, err
	}
	s.store = store
	return &s, nil
}

type S3Storage struct {
	config *config.S3Config
	//scwsConfig *config.Config
	store cloudstorage.Store
	index string
}

type object struct {
	prefix string
	store  cloudstorage.Store
	index  string
}

func (o *object) getObject(name string) (cloudstorage.Object, error) {
obj, err := o.store.Get(context.Background(), path.Join(o.prefix, name))
	if err != nil {
		log.Println("s3.getObject", err.Error())
		return nil, err
	}

	return obj, nil
}

func (o *object) Open(name string) (http.File, error) {
	obj, err := o.getObject(name)
	if err != nil {
		return nil, err
	}
	f, err := obj.Open(cloudstorage.ReadOnly)
	if err != nil {
		log.Println("s3.Open", err.Error())
		return nil, err
	}
	return f, nil
}
// scanETag determines if a syntactically valid ETag is present at s. If so,
// the ETag and remaining text after consuming ETag is returned. Otherwise,
// it returns "", "".
func scanETag(s string) (etag string, remain string) {
	s = textproto.TrimString(s)
	start := 0
	if strings.HasPrefix(s, "W/") {
		start = 2
	}
	if len(s[start:]) < 2 || s[start] != '"' {
		return "", ""
	}
	// ETag is either W/"text" or "text".
	// See RFC 7232 2.3.
	for i := start + 1; i < len(s); i++ {
		c := s[i]
		switch {
		// Character values allowed in ETags.
		case c == 0x21 || c >= 0x23 && c <= 0x7E || c >= 0x80:
		case c == '"':
			return s[:i+1], s[i+1:]
		default:
			return "", ""
		}
	}
	return "", ""
}
// etagWeakMatch reports whether a and b match using weak ETag comparison.
// Assumes a and b are valid ETags.
func etagWeakMatch(a, b string) bool {
	return strings.TrimPrefix(a, "W/") == strings.TrimPrefix(b, "W/")
}
func checkIfNoneMatch(w http.ResponseWriter, r *http.Request) bool {
	inm := r.Header.Get("If-None-Match")
	if inm == "" {
		return false
	}
	buf := inm
	for {
		buf = textproto.TrimString(buf)
		if len(buf) == 0 {
			break
		}
		if buf[0] == ',' {
			buf = buf[1:]
			continue
		}
		if buf[0] == '*' {
			return false
		}
		etag, remain := scanETag(buf)
		if etag == "" {
			break
		}
		if etagWeakMatch(etag, w.Header().Get("Etag")) {
			return false
		}
		buf = remain
	}
	return true
}
func writeNotModified(w http.ResponseWriter) {
	// RFC 7232 section 4.1:
	// a sender SHOULD NOT generate representation metadata other than the
	// above listed fields unless said metadata exists for the purpose of
	// guiding cache updates (e.g., Last-Modified might be useful if the
	// response does not have an ETag field).
	h := w.Header()
	delete(h, "Content-Type")
	delete(h, "Content-Length")
	if h.Get("Etag") != "" {
		delete(h, "Last-Modified")
	}
	w.WriteHeader(http.StatusNotModified)
}

func (s *S3Storage) ServeHTTP(c common.StaticSiteConfig, w http.ResponseWriter, r *http.Request) {
	if checkIfNoneMatch(w, r) {
		if r.Method == "GET" || r.Method == "HEAD" {
			writeNotModified(w)
			return
		} else {
			w.WriteHeader(http.StatusPreconditionFailed)
		}
	}
	eTag := c.ETag
	staticFilePath := staticFilePath(r)
	o := s.newObject()
	fileHandle, error := o.Open(filepath.Join(c.Path, staticFilePath))
	if error != nil {
		fileHandle, error = o.Open(filepath.Join(c.Path, c.Error))
		if serve404OnErr(error, w) {
			return
		}
	}
	defer fileHandle.Close()
	fileInfo, error := fileHandle.Stat()
	if serve404OnErr(error, w) {
		return
	}
	w.Header().Set("Etag", fmt.Sprintf("\"%s\"", eTag))
	w.Header().Set("Cache-Control", "public must-revalidate stale-if-error=86400")
	w.Header().Set("Content-Type", mime.TypeByExtension(filepath.Ext(staticFilePath)))
	http.ServeContent(w, r, fileInfo.Name(), time.Unix(0,0), fileHandle)
}
func staticFilePath(request *http.Request) string {
	return request.URL.Path
}
func serve404OnErr(err error, responseWriter http.ResponseWriter) bool {
	if err != nil {
		serve404(responseWriter)
		return true
	}
	return false
}
func serve404(responseWriter http.ResponseWriter) {
	responseWriter.WriteHeader(http.StatusNotFound)
	template := []byte("Error 404 - Page Not Found.")
	fmt.Fprint(responseWriter, string(template))
}
func (s *S3Storage) ServeIndex(c common.StaticSiteConfig, w http.ResponseWriter, r *http.Request) {
	staticFilePath := r.URL.Path
	if !strings.HasPrefix(staticFilePath, "/") {
		staticFilePath = "/" + staticFilePath
		r.URL.Path = staticFilePath
	}
	if strings.HasSuffix(staticFilePath, "/") {
		staticFilePath = staticFilePath + c.Index
		r.URL.Path = staticFilePath
	}
	r.URL.Path = staticFilePath
	s.ServeHTTP(c, w, r)
}

func (s *S3Storage) newObject() *object {
	return &object{
		prefix: s.config.Prefix,
		store:  s.store,
		index:  s.index,
	}
}

func (s *S3Storage) GetName() string {
	return "s3"
}