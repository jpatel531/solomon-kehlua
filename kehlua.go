package main

import (
	"flag"
	"fmt"
	"github.com/mitchellh/goamz/aws"
	"github.com/mitchellh/goamz/s3"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	s "strings"
)

type file struct {
	path        string
	data        []byte
	contentType string
}

const permissions = s3.BucketOwnerFull

var versionRegex = regexp.MustCompile(`\d+-\d+-\d+`)

func prepareFilesInDir(localDir, s3Dir string, files *[]file) {
	dir, err := ioutil.ReadDir(localDir)
	if err != nil {
		panic(err)
	}

	for _, _file := range dir {
		name := _file.Name()

		if s.HasPrefix(name, ".") {
			continue
		}

		localPath := fmt.Sprintf("%s/%s", localDir, name)
		s3Path := fmt.Sprintf("%s/%s", s3Dir, name)
		if _file.IsDir() {
			prepareFilesInDir(localPath, s3Path, files)
		} else {
			data, fileErr := ioutil.ReadFile(localPath)
			if fileErr != nil {
				panic(fileErr)
			}
			contentType := CONTENT_TYPE_LOOKUP[filepath.Ext(localPath)]
			s3Path = s.Replace(s3Path, "./", "", 1)
			item := file{s3Path, data, contentType}
			*files = append(*files, item)
		}
	}
}

func upload(_file file, uploads chan<- bool, client *s3.S3, bucket *s3.Bucket) {
	err := bucket.Put(_file.path, _file.data, _file.contentType, permissions)
	if err != nil {
		fmt.Printf("UPLOAD ERROR: %+v\n", err)
		panic(err)
	}
	uploads <- true
	fmt.Printf("Uploaded %s!\n", _file.path)
}

func main() {

	dirnamePtr := flag.String("folder", "", "the older where the assets reside")
	versionPtr := flag.String("version", "", "the version of the styleguide")

	flag.Parse()

	dirname := *dirnamePtr
	version := *versionPtr

	if dirname == "" {
		fmt.Println("`folder` must be specified")
		return
	}

	if !versionRegex.MatchString(version) {
		fmt.Println("`version` must be specified in the format major-minor-patch")
		return
	}

	var files []file

	prepareFilesInDir(dirname, version, &files)

	requestNumber := len(files)

	uploads := make(chan bool, requestNumber)

	auth, err := aws.EnvAuth()
	if err != nil {
		panic(err)
		os.Exit(1)
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
