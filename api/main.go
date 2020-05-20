package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"cloud.google.com/go/storage"
	"github.com/go-redis/redis/v7"
	"github.com/gorilla/mux"
	"google.golang.org/api/option"
)

var (
	flagRedis             = flag.String("redis", "localhost:6379", "")
	flagRedisMasterName   = flag.String("redis-master-name", "", "")
	flagBucket            = flag.String("bucket", "smiling-xanflorp", "")
	flagBucketCredentials = flag.String("bucket-credentials", "", "")
)

var redisClient redis.UniversalClient
var bucket *storage.BucketHandle
var ctx context.Context

func main() {
	flag.Parse()
	err := run()
	if err != nil {
		panic(err)
	}
}

func run() error {
	var err error
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	redisClient = redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:      strings.Split(*flagRedis, ","),
		MasterName: *flagRedisMasterName,
	})

	bucketOptions := []option.ClientOption(nil)
	if *flagBucketCredentials != "" {
		bucketOptions = append(bucketOptions, option.WithCredentialsFile(*flagBucketCredentials))
	}
	bucketClient, err := storage.NewClient(ctx, bucketOptions...)
	if err != nil {
		return err
	}
	bucket = bucketClient.Bucket(*flagBucket)

	r := mux.NewRouter()
	r.Methods("GET").Path("/").HandlerFunc(defaultHandler)
	r.Methods("GET").Path("/api").HandlerFunc(defaultHandler)
	r.Methods("GET").Path("/api/images").HandlerFunc(imageListHandler)
	r.Methods("POST").Path("/api/images/upload").HandlerFunc(imageUploadHandler)
	r.Methods("GET").Path("/api/images/{id}").HandlerFunc(imageGetHandler)
	r.Methods("POST").Path("/api/convert").HandlerFunc(convertHandler)
	http.Handle("/", r)
	return http.ListenAndServe(":80", nil)
}

func handleError(w http.ResponseWriter, err error) {
	w.Write([]byte(err.Error()))
	w.WriteHeader(http.StatusInternalServerError)
}

func defaultHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func imageListHandler(w http.ResponseWriter, r *http.Request) {
	images, err := redisClient.LRange("images", 0, -1).Result()
	if err != nil {
		handleError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(images)
}

func imageUploadHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	id := uuid.Must(uuid.NewRandom()).String()
	writer := bucket.Object(id).NewWriter(ctx)
	file, _, err := r.FormFile("file")
	if err != nil {
		handleError(w, err)
		return
	}
	defer file.Close()
	_, err = io.Copy(writer, file)
	if err != nil {
		handleError(w, err)
		return
	}
	err = writer.Close()
	if err != nil {
		handleError(w, err)
		return
	}
	_, err = redisClient.LPush("images", id).Result()
	if err != nil {
		handleError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func imageGetHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	vars := mux.Vars(r)
	reader, err := bucket.Object(vars["id"]).NewReader(ctx)
	if err != nil {
		handleError(w, err)
		return
	}
	defer reader.Close()
	buffer := &bytes.Buffer{}
	_, err = io.CopyN(buffer, reader, 512)
	if err == io.EOF {
	} else if err != nil {
		handleError(w, err)
		return
	}
	mimeType := http.DetectContentType(buffer.Bytes())
	ext, err := mime.ExtensionsByType(mimeType)
	if err != nil {
		handleError(w, err)
		return
	}
	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s%s"`, vars["id"], ext[0]))
	w.WriteHeader(http.StatusOK)
	io.Copy(w, buffer)
	io.Copy(w, reader)
}

type ConvertRequest struct {
	ImageID string `json:"image-id"`
	StyleID string `json:"style-id"`
}

func convertHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var req ConvertRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	r.Body.Close()
	if err != nil {
		handleError(w, err)
		return
	}
	_, err = bucket.Object(req.ImageID).Attrs(ctx)
	if err != nil {
		handleError(w, err)
		return
	}
	_, err = bucket.Object(req.StyleID).Attrs(ctx)
	if err != nil {
		handleError(w, err)
		return
	}
	job, err := json.Marshal(&req)
	if err != nil {
		handleError(w, err)
		return
	}
	_, err = redisClient.LPush("jobs", string(job)).Result()
	if err != nil {
		handleError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}
