module scws

go 1.16

require (
	github.com/HdrHistogram/hdrhistogram-go v1.1.2 // indirect
	github.com/araddon/gou v0.0.0-20190110011759-c797efecbb61
	github.com/hashicorp/vault/api v1.1.1
	github.com/lytics/cloudstorage v0.2.9
	github.com/opentracing/opentracing-go v1.2.0
	github.com/prometheus/client_golang v1.11.0
	github.com/stretchr/testify v1.7.0
	github.com/uber/jaeger-client-go v2.29.1+incompatible
	github.com/uber/jaeger-lib v2.4.1+incompatible
	go.uber.org/atomic v1.9.0 // indirect
)

replace github.com/lytics/cloudstorage v0.2.9 => github.com/JayCeeJr/cloudstorage v0.2.10-0.20211206173816-8f220ee37a11
