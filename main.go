package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/jhunt/go-s3"
	"golang.org/x/crypto/acme/autocert"
)

func main() {
	key, ok := os.LookupEnv("S3_KEY")
	if !ok {
		log.Fatal("Could not find S3_KEY, please ensure it is set.")
	}
	secret, ok := os.LookupEnv("S3_SECRET")
	if !ok {
		log.Fatal("Could not find S3_SECRET, please ensure it is set.")
	}
	domain, ok := os.LookupEnv("S3_DOMAIN")
	if !ok {
		log.Fatal("Could not find S3_DOMAIN, please ensure it is set.")
	}
	bucket, ok := os.LookupEnv("S3_BUCKET")
	if !ok {
		log.Fatal("Could not find S3_BUCKET, please ensure it is set.")
	}

	kd := newKubeDirector()

	man := autocert.Manager{
		Prompt: autocert.AcceptTOS,
		Cache:  newS3Cache(key, secret, domain, bucket),
		HostPolicy: func(_ context.Context, host string) error {
			if _, ok := kd.domains[host]; ok {
				return nil
			}
			return fmt.Errorf("Host %q not handled", host)
		},
	}

	go func() {
		err := http.ListenAndServe(":80", man.HTTPHandler(nil))
		if err != nil {
			log.Fatal(err)
		}
	}()

	proxy := &httputil.ReverseProxy{
		Director:      kd.Direct,
		FlushInterval: 5 * time.Second,
	}

	go func() {
		err := http.ListenAndServe(":8080", nil)
		if err != nil {
			log.Fatal(err)
		}
	}()

	err := http.Serve(man.Listener(), proxy)
	if err != nil {
		log.Fatal(err)
	}
}

func newS3Cache(key, secret, domain, bucket string) *S3Cache {
	client, _ := s3.NewClient(&s3.Client{
		AccessKeyID:     key,
		SecretAccessKey: secret,
		Domain:          domain,
		Bucket:          bucket,
		UsePathBuckets:  true,
	})
	return &S3Cache{client}
}

type S3Cache struct {
	s3 *s3.Client
}

func (s *S3Cache) Get(ctx context.Context, key string) ([]byte, error) {
	r, err := s.s3.Get(key)
	if err != nil {
		return nil, autocert.ErrCacheMiss
	}
	buf, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func (s *S3Cache) Put(ctx context.Context, key string, data []byte) error {
	u, err := s.s3.NewUpload(key, nil)
	if err != nil {
		return err
	}
	err = u.Write(data)
	if err != nil {
		return err
	}
	return u.Done()
}

func (s *S3Cache) Delete(ctx context.Context, key string) error {
	return s.s3.Delete(key)
}
