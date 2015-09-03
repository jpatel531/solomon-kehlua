package main

import (
	"flag"
	"fmt"
	"github.com/mitchellh/goamz/aws"
	"github.com/mitchellh/goamz/s3"
	"io/ioutil"
	"path/filepath"
	s "strings"
)

type file struct {
	path        string
	data        []byte
	contentType string
}

const permissions = s3.BucketOwnerFull

func prepareFilesInDir(dirname string, files *[]file) {
	dir, err := ioutil.ReadDir(dirname)
	if err != nil {
		panic(err)
	}

	for _, _file := range dir {
		name := _file.Name()

		if s.HasPrefix(name, ".") {
			continue
		}

		path := fmt.Sprintf("%s/%s", dirname, name)
		if _file.IsDir() {
			prepareFilesInDir(path, files)
		} else {
			data, fileErr := ioutil.ReadFile(path)
			if fileErr != nil {
				panic(fileErr)
			}
			contentType := CONTENT_TYPE_LOOKUP[filepath.Ext(path)]
			path = s.Replace(path, "./", "", 1)
			item := file{path, data, contentType}
			*files = append(*files, item)
		}
	}
}

func upload(_file file, uploads chan<- bool, client *s3.S3, bucket *s3.Bucket) {
	bucket.Put(_file.path, _file.data, _file.contentType, permissions)
	uploads <- true
	fmt.Printf("Uploaded %s!\n", _file.path)
}

func main() {

	dirnamePtr := flag.String("folder", "", "the older where the assets reside")

	flag.Parse()

	dirname := *dirnamePtr

	if dirname == "" {
		fmt.Println("`folder` must be specified")
		return
	}

	var files []file

	prepareFilesInDir(dirname, &files)

	requestNumber := len(files)

	uploads := make(chan bool, requestNumber)

	auth, err := aws.EnvAuth()
	if err != nil {
		panic(err)
	}

	client := s3.New(auth, aws.USEast)
	bucket := client.Bucket("testington")

	for _, _file := range files {
		go upload(_file, uploads, client, bucket)
	}

	for u := 1; u <= requestNumber; u++ {
		<-uploads
	}

	close(uploads)

	fmt.Println("DONE")

}
