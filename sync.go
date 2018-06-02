package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// SyncFolderIterator is used to upload a given folder
// to Amazon S3.
type SyncFolderIterator struct {
	bucket    string
	fileInfos []fileInfo
	err       error
}

type fileInfo struct {
	key         string
	fullpath    string
	contentType string
}

func detectContentType(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil {
		return "", err
	}

	contentType := http.DetectContentType(buffer)
	fmt.Println(path, contentType)

	return contentType, nil
}

// NewSyncFolderIterator will walk the path, and store the key and full path
// of the object to be uploaded. This will return a new SyncFolderIterator
// with the data provided from walking the path.
func NewSyncFolderIterator(path, bucket string) *SyncFolderIterator {
	metadata := []fileInfo{}
	filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			key := strings.TrimPrefix(p, path)
			contentType, err := detectContentType(p)
			if err != nil {
				errors.New("no file context could be assigned")
			}
			metadata = append(metadata, fileInfo{key, p, contentType})
		}

		return nil
	})

	return &SyncFolderIterator{
		bucket,
		metadata,
		nil,
	}
}

// Next will determine whether or not there is any remaining files to
// be uploaded.
func (iter *SyncFolderIterator) Next() bool {
	return len(iter.fileInfos) > 0
}

// Err returns any error when os.Open is called.
func (iter *SyncFolderIterator) Err() error {
	return iter.err
}

// UploadObject will prep the new upload object by open that file and constructing a new
// s3manager.UploadInput.
func (iter *SyncFolderIterator) UploadObject() s3manager.BatchUploadObject {
	fi := iter.fileInfos[0]
	iter.fileInfos = iter.fileInfos[1:]
	body, err := os.Open(fi.fullpath)
	if err != nil {
		iter.err = err
	}

	input := s3manager.UploadInput{
		Bucket:      &iter.bucket,
		Key:         &fi.key,
		Body:        body,
		ContentType: &fi.contentType,
	}

	return s3manager.BatchUploadObject{
		&input,
		nil,
	}
}

func SyncFolderToBucket(bucket string, region string, path string) error {
	p := path
	b := bucket
	r := region

	sess := session.New(&aws.Config{
		Region: &r,
	})
	uploader := s3manager.NewUploader(sess)

	iter := NewSyncFolderIterator(p, b)
	if err := uploader.UploadWithIterator(aws.BackgroundContext(), iter); err != nil {
		fmt.Fprintf(os.Stderr, "unexpected error has occured: %v", err)
	}

	if err := iter.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "unexpected error occured during file walking: %v", err)
	}

	fmt.Println("Successful Sync to S3")

	return nil
}
