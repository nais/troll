package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"cloud.google.com/go/storage"
)

var (
	grytaEndpoint string
	bindAddr      string
	fakeResponse  string
	bucketName    string
)

func init() {
	flag.StringVar(&fakeResponse, "fake-response", os.Getenv("FAKE_RESPONSE"), "run in dev mode")
	flag.StringVar(&bucketName, "bucket-name", os.Getenv("BUCKET_NAME"), "name of bucket")
	flag.StringVar(&bindAddr, "bind-address", ":8080", "ip:port where http requests are served")

	flag.Parse()
}

func main() {
	// Fetch the elector URL from environment variable
	electorUrl := os.Getenv("ELECTOR_GET_URL")
	if electorUrl == "" {
		log.Fatal("ELECTOR_GET_URL environment variable is required")
	}

	signaller := New(electorUrl, bucketName)
	go signaller.Run()

	http.HandleFunc("/", func(wr http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(wr, "tihi")
	})

	if err := http.ListenAndServe(bindAddr, nil); err != nil {
		log.Fatal(err)
	}
}

type Signaller struct {
	leURL  string
	bucket string
}

func New(leUrl, bucket string) *Signaller {
	return &Signaller{
		leURL:  leUrl,
		bucket: bucket,
	}
}

func (s *Signaller) Run() error {
	for {
		leader, err := leaderHostname(s.leURL)
		if err != nil {
			return err
		}

		hostname, err := os.Hostname()
		if err != nil {
			return err
		}

		fmt.Printf("I am the leader: %v\n", leader)

		writeToBucket(leader == hostname, hostname, s.bucket)
		time.Sleep(5 * time.Second)
	}
}

func writeToBucket(leader bool, hostname, bucket string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("creating storage client: %s", err)
	}

	bkt := client.Bucket(bucket)
	obj := bkt.Object(hostname)

	writer := obj.NewWriter(ctx)

	payload := "false"
	if leader {
		payload = "true"
	}

	if _, err := fmt.Fprint(writer, payload); err != nil {
		return fmt.Errorf("writing data to bucket: %s", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("closing bucket writer: %s", err)
	}

	log.Println("Successfully wrote data to bucket")
	return nil
}

func leaderHostname(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var data struct {
		Name string `json:"name"`
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return "", err
	}

	return data.Name, nil
}
