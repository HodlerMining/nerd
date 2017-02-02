package data

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/nerdalize/nerd/nerd"
)

const DirectoryPermissions = 0755

type KeyWriter interface {
	Write(k string) error
}

type Client struct {
	Session *session.Session
}

func NewClient(awsCreds *credentials.Credentials) (*Client, error) {
	sess, err := session.NewSession(&aws.Config{
		Credentials: awsCreds,
		Region:      aws.String("eu-west-1"),
	})
	if err != nil {
		return nil, fmt.Errorf("could not create AWS sessions: %v", err)
	}
	return &Client{
		Session: sess,
	}, nil
}

func (client *Client) UploadFile(filePath string, dataset string) error {
	file, err := os.Open(filePath)
	defer file.Close()
	if err != nil {
		return fmt.Errorf("could not open file '%v': %v", filePath, err)
	}
	svc := s3.New(client.Session)
	params := &s3.PutObjectInput{
		Bucket: aws.String(nerd.GetCurrentUser().AWSBucket),             // Required
		Key:    aws.String(path.Join(dataset, filepath.Base(filePath))), // Required
		Body:   file,
	}
	_, err = svc.PutObject(params)
	if err != nil {
		return fmt.Errorf("could not put file '%v': %v", filePath, err)
	}
	return nil
}

func (client *Client) UploadDir(dir string, dataset string) error {
	err := filepath.Walk(dir, func(path string, f os.FileInfo, err error) error {
		if f.Mode().IsRegular() {
			return client.UploadFile(path, dataset)
		}
		return nil
	})
	return err
}

func (client *Client) UploadFiles(files []string, dataset string, kw KeyWriter, concurrency int) error {

	type item struct {
		filePath string
		resCh    chan bool
		err      error
	}

	work := func(it *item) {
		it.err = client.UploadFile(it.filePath, dataset)
		it.resCh <- true
	}

	itemCh := make(chan *item, concurrency)
	go func() {
		defer close(itemCh)
		for i := 0; i < len(files); i++ {
			it := &item{
				filePath: files[i],
				resCh:    make(chan bool),
			}

			go work(it)  //create work
			itemCh <- it //send to fan-in thread for syncing results
		}
	}()

	//fan-in
	for it := range itemCh {
		<-it.resCh
		if it.err != nil {
			return fmt.Errorf("failed to upload '%v': %v", it.filePath, it.err)
		}

		err := kw.Write(it.filePath)
		if err != nil {
			return fmt.Errorf("failed to write key: %v", err)
		}
	}

	return nil
}

func (client *Client) DownloadFile(key string, outDir string) error {
	base := filepath.Dir(path.Join(outDir, key))
	err := os.MkdirAll(base, DirectoryPermissions)
	if err != nil {
		return fmt.Errorf("failed to create path '%v': %v", base, err)
	}
	outFile, err := os.Create(path.Join(outDir, key))
	defer outFile.Close()
	if err != nil {
		return fmt.Errorf("failed to create local file '%v': %v", path.Join(outDir, key), err)
	}

	svc := s3.New(client.Session)
	params := &s3.GetObjectInput{
		Bucket: aws.String(nerd.GetCurrentUser().AWSBucket), // Required
		Key:    aws.String(key),                             // Required
	}
	resp, err := svc.GetObject(params)

	if err != nil {
		return fmt.Errorf("failed to download '%v': %v", key, err)
	}

	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write output to '%v': %v", path.Join(outDir, key), err)
	}

	return nil
}

func (client *Client) ListDataset(dataset string) (keys []string, err error) {
	svc := s3.New(client.Session)

	params := &s3.ListObjectsInput{
		Bucket: aws.String(nerd.GetCurrentUser().AWSBucket), // Required
		Prefix: aws.String(dataset),
	}
	resp, err := svc.ListObjects(params)

	if err != nil {
		return nil, fmt.Errorf("failed to list dataset '%v': %v", dataset, err)
	}

	for _, object := range resp.Contents {
		keys = append(keys, aws.StringValue(object.Key))
	}

	return
}

func (client *Client) DownloadFiles(dataset string, outDir string, kw KeyWriter, concurrency int) error {
	keys, err := client.ListDataset(dataset)
	if err != nil {
		return err
	}

	type item struct {
		key    string
		outDir string
		resCh  chan bool
		err    error
	}

	work := func(it *item) {
		it.err = client.DownloadFile(it.key, it.outDir)
		it.resCh <- true
	}

	itemCh := make(chan *item, concurrency)
	go func() {
		defer close(itemCh)
		for i := 0; i < len(keys); i++ {
			it := &item{
				key:    keys[i],
				outDir: outDir,
				resCh:  make(chan bool),
			}

			go work(it)  //create work
			itemCh <- it //send to fan-in thread for syncing results
		}
	}()

	//fan-in
	for it := range itemCh {
		<-it.resCh
		if it.err != nil {
			return fmt.Errorf("failed to download '%v': %v", it.key, it.err)
		}

		err := kw.Write(path.Join(outDir, it.key))
		if err != nil {
			return fmt.Errorf("failed to write key: %v", err)
		}
	}

	return nil
}
