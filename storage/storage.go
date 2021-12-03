package storage

import (
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"log"
	"net/http"
	"scws/common"
	"scws/common/config"
	"scws/storage/s3"
	"strings"
)

const (
	FSStorage  = "filesystem"
	S3         = "s3"
	serverName = "static-backend"
)

type IStorage interface {
	ServeHTTP(c common.StaticSiteConfig, w http.ResponseWriter, r *http.Request)
	GetName() string
	ServeIndex(c common.StaticSiteConfig, w http.ResponseWriter, r *http.Request)
}

type Storage struct {
	storage IStorage
	config  *config.Config
}

func New(c *config.Config) (*Storage, error) {
	var err error
	s := Storage{config: c}
	s.storage, err = s3.New(c)
	if s.storage == nil {
		log.Println("couldn't connect to storage")
		return nil, err
	}
	return &s, nil
}

func (s *Storage) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tracer := opentracing.GlobalTracer()
	var span opentracing.Span
	if tracer != nil {
		spanCtx, err := tracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(r.Header))
		if err != nil {
			span = tracer.StartSpan("storage.ServeHTTP")
		} else {
			span = tracer.StartSpan("storage.ServeHTTP", ext.RPCServerOption(spanCtx))
		}
		defer span.Finish()
		span.SetTag("http.status_code", http.StatusOK)
		span.SetTag("http.url", r.URL.Path)
		span.SetTag("storage", s.storage.GetName())
	}
	c := common.Route(r.Host)
	if strings.HasSuffix(r.URL.Path, "/") || r.URL.Path == "/" {
		s.storage.ServeIndex(c, w, r)
	} else {
		s.storage.ServeHTTP(c, w, r)
	}
	log.Println(r.URL.Path, r.RemoteAddr, w.Header().Get("status"))
}
