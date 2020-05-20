package main

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/go-redis/redis/v7"

	batch_v1 "k8s.io/api/batch/v1"
	core_v1 "k8s.io/api/core/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	flagRedis           = flag.String("redis", "localhost:6379", "")
	flagRedisMasterName = flag.String("redis-master-name", "", "")
	flagAPI             = flag.String("api", "http://api", "")
	flagJobNamespace    = flag.String("job-namespace", "app", "")
)

var redisClient redis.UniversalClient
var ctx context.Context
var k8sClient *kubernetes.Clientset

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

	config, err := rest.InClusterConfig()
	if err != nil {
		return err
	}
	k8sClient, err = kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	for {
		err := processOne()
		if err != nil {
			return err
		}
		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

type ConvertRequest struct {
	ImageID string `json:"image-id"`
	StyleID string `json:"style-id"`
}

func processOne() error {
	return redisClient.Watch(func(tx *redis.Tx) error {
		jobString, err := tx.LPop("jobs").Result()
		if err == redis.Nil {
			return nil
		} else if err != nil {
			return err
		}
		var req ConvertRequest
		err = json.Unmarshal([]byte(jobString), &req)
		if err != nil {
			return err
		}
		job := createJob(req)
		_, err = k8sClient.BatchV1().Jobs(*flagJobNamespace).Create(job)
		if k8s_errors.IsAlreadyExists(err) {
			return nil
		} else if err != nil {
			return err
		}
		return nil
	})
}

func createJob(req ConvertRequest) *batch_v1.Job {
	container := core_v1.Container{
		Name:            "job",
		Image:           "demo/job:latest",
		ImagePullPolicy: core_v1.PullAlways,
		Env: []core_v1.EnvVar{
			core_v1.EnvVar{
				Name:  "STYLE_IMAGE",
				Value: fmt.Sprintf("%s/api/images/%s", *flagAPI, req.StyleID),
			},
			core_v1.EnvVar{
				Name:  "CONTENT_IMAGE",
				Value: fmt.Sprintf("%s/api/images/%s", *flagAPI, req.ImageID),
			},
			core_v1.EnvVar{
				Name:  "UPLOAD_URL",
				Value: fmt.Sprintf("%s/api/images/upload", *flagAPI),
			},
		},
		Resources: core_v1.ResourceRequirements{
			Requests: core_v1.ResourceList{
				core_v1.ResourceCPU:    resource.MustParse("2000m"),
				core_v1.ResourceMemory: resource.MustParse("8Gi"),
			},
			Limits: core_v1.ResourceList{
				core_v1.ResourceCPU:    resource.MustParse("2000m"),
				core_v1.ResourceMemory: resource.MustParse("8Gi"),
				"nvidia.com/gpu":       resource.MustParse("1"),
			},
		},
	}
	id := sha1.New()
	id.Write([]byte(req.ImageID))
	id.Write([]byte(req.StyleID))
	hash := id.Sum(nil)
	jobID := hex.EncodeToString(hash[0:6])
	job := &batch_v1.Job{}
	job.Name = fmt.Sprintf("convert-%s", jobID)
	job.Spec = batch_v1.JobSpec{
		Template: core_v1.PodTemplateSpec{
			Spec: core_v1.PodSpec{
				Containers: []core_v1.Container{
					container,
				},
				RestartPolicy: core_v1.RestartPolicyOnFailure,
				NodeSelector: map[string]string{
					"cloud.google.com/gke-accelerator": "nvidia-tesla-t4",
				},
			},
		},
	}
	return job
}
