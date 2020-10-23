package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

var config = Configuration{}
var s3Svc *s3.S3
var s3downloader *s3manager.Downloader
var prefix string
var writePath string
var compType string

type Configuration struct {
	AccessKey string
	SecretKey string
	Bucket    string
	Region    string
	Port      int
}
type FakeWriterAt struct {
	w io.Writer
}

func (fw FakeWriterAt) WriteAt(p []byte, offset int64) (n int, err error) {
	// ignore 'offset' because we forced sequential downloads
	return fw.w.Write(p)
}

func initS3() {
	creds := credentials.NewStaticCredentials(config.AccessKey, config.SecretKey, "")
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(config.Region),
		Credentials: creds,
	})
	if err != nil {
		fmt.Println("err", err)
	}
	s3Svc = s3.New(sess, &aws.Config{Region: aws.String(config.Region)}) // TODO - remove region?

	downloader := s3manager.NewDownloader(sess, func(d *s3manager.Downloader) {
		// d.PartSize = 64 * 1024 * 1024 // 64MB per part
		// d.Concurrency = 5
	})

	s3downloader = downloader

}

func main() {
	// Parse config
	configFile, _ := os.Open("conf.json")
	decoder := json.NewDecoder(configFile)
	err := decoder.Decode(&config)
	if err != nil {
		panic("Error reading conf")
	}
	// Init s3
	initS3()

	// Run server
	fmt.Println("Running on port", config.Port)
	http.HandleFunc("/", handler)
	err = http.ListenAndServe(":"+strconv.Itoa(config.Port), nil)
	if err != nil {
		fmt.Println("Error starting server", err)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	compType = "zip"
	health, ok := r.URL.Query()["health"]
	if len(health) > 0 {
		fmt.Fprintf(w, "OK")
		return
	}

	cType, ok := r.URL.Query()["comp"]
	if len(cType) > 0 {
		// fmt.Println("comp tpye", cType)
		if cType[0] == "tar" {
			compType = "tar"
		}
	}

	// Get "prefix" URL params
	pres, ok := r.URL.Query()["prefix"]
	if !ok || len(pres) < 1 {
		http.Error(w, "Prefix not specified. Pass ?prefix= to use.", 500)
		return
	}
	prefix = pres[0]

	// Get path url parms
	// Specifys the folder name they want to write to in the zip
	path, ok := r.URL.Query()["path"]

	if !ok || len(pres) < 1 {
		// if no path is specified write to root folder
		writePath = "/"
	} else {
		// If folder is specified make sure there is a trailing '/'
		fixedPath := path[0]
		if fixedPath[len(fixedPath)-1:] != "/" {
			fixedPath += "/"
		}
		writePath = fixedPath
	}

	res, err := s3Svc.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: aws.String(config.Bucket),
		Prefix: aws.String(prefix),
	})
	if err != nil {
		fmt.Println(err)
		return
	}

	// Start processing the response
	w.Header().Add("Content-Disposition", "attachment; filename=\""+GetName(prefix)+"."+compType+"\"")
	w.Header().Add("Content-Type", "application/"+compType)

	// tarit(res.Contents)
	if compType == "zip" {
		zipit(w, res.Contents)
	}
	if compType == "tar" {
		tarit(w, res.Contents)
	}

	// Loop over files, add them to the

	log.Printf("%s\t%s\t%s", compType, r.RequestURI, time.Since(start))
}

func tarit(w http.ResponseWriter, contents []*s3.Object) {
	gw := gzip.NewWriter(w)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()
	for _, file := range contents {
		// fmt.Println("File", *file)
		filePath := *file.Key
		path := strings.Replace(filePath, prefix, writePath, 1)

		header := new(tar.Header)
		header.Name = path
		header.Size = *file.Size
		// header.Mode = int64(0755) // TODO?
		header.ModTime = *file.LastModified
		// write the header to the tarball archive
		if err := tw.WriteHeader(header); err != nil {
			// return err;;
			fmt.Println("err", err)
		}

		// Read file from S3, log any errors
		_, err := s3downloader.Download(FakeWriterAt{tw}, &s3.GetObjectInput{
			Bucket: aws.String(config.Bucket),
			Key:    aws.String(*file.Key),
		})
		if err != nil {
			fmt.Println("dl err", err)
		}

	}
	return
}

func zipit(w http.ResponseWriter, contents []*s3.Object) {
	zipWriter := zip.NewWriter(w)
	// pr, pw := io.Pipe()
	// Create zip.Write which will writes to pipes

	for _, file := range contents {
		// fmt.Println("File", *file)

		// Ge rid of prefix in zip path
		filePath := *file.Key
		zipPath := strings.Replace(filePath, prefix, writePath, 1)
		// fmt.Println("zipPath", zipPath)
		h := &zip.FileHeader{
			Name:     zipPath,
			Method:   zip.Deflate,
			Flags:    0x800,
			Modified: *file.LastModified,
		}

		f, _ := zipWriter.CreateHeader(h)
		// Build safe file file name

		// Read file from S3, log any errors
		_, err := s3downloader.Download(FakeWriterAt{f}, &s3.GetObjectInput{
			Bucket: aws.String(config.Bucket),
			Key:    aws.String(*file.Key),
		})
		if err != nil {
			fmt.Println("dl err", err)
		}

	}

	zipWriter.Close()
	return
}

var makeSafeFileName = regexp.MustCompile(`[#<>:"/\|?*\\]`)

func GetName(path string) string {
	// Returns the last item of the prefix
	// should return either the file name or folder

	s := strings.Split(path, "/")

	root_item_name := s[len(s)-1]
	// Added because folders are often sandwitched between two /'s
	if root_item_name == "" {
		root_item_name = s[len(s)-2]
	}

	safeFileName := makeSafeFileName.ReplaceAllString(root_item_name, "")
	if safeFileName == "" { // Unlikely but just in case
		safeFileName = "file"
	}

	return safeFileName
}

// Todo:
// 	1. Better Error handling
// 	2. Turn any repeated code into functions
// 	3. Add api auth/key?
