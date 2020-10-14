package main

import (
	"archive/zip"
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
	http.ListenAndServe(":"+strconv.Itoa(config.Port), nil)
}

func handler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	health, ok := r.URL.Query()["health"]
	if len(health) > 0 {
		fmt.Fprintf(w, "OK")
		return
	}

	// Get "ref" URL params
	pres, ok := r.URL.Query()["prefix"]
	if !ok || len(pres) < 1 {
		http.Error(w, "Prefix not specified. Pass ?prefix= to use.", 500)
		return
	}
	_ = pres[0]

	// Get "downloadas" URL params
	downloadas, ok := r.URL.Query()["downloadas"]
	if !ok && len(downloadas) > 0 {
		downloadas[0] = makeSafeFileName.ReplaceAllString(downloadas[0], "")
		if downloadas[0] == "" {
			downloadas[0] = "download.zip"
		}
	} else {
		downloadas = append(downloadas, "download.zip")
	}

	res, err := s3Svc.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: aws.String(config.Bucket),
		Prefix: aws.String("code/MyProject/main.py"),
	})

	// Start processing the response
	w.Header().Add("Content-Disposition", "attachment; filename=\""+downloadas[0]+"\"")
	w.Header().Add("Content-Type", "application/zip")

	// Loop over files, add them to the
	zipWriter := zip.NewWriter(w)
	// pr, pw := io.Pipe()
	// Create zip.Write which will writes to pipes

	for _, file := range res.Contents {
		fmt.Println("File", *file)
		filePath := *file.Key
		// safeFileName := makeSafeFileName.ReplaceAllString(*file.Key, "")
		// if safeFileName == "" { // Unlikely but just in case
		// 	safeFileName = "file"
		// }

		// Build a good path for the file within the zip
		zipPath := "test/"
		zipPath += filePath

		h := &zip.FileHeader{
			Name:     zipPath,
			Method:   zip.Deflate,
			Flags:    0x800,
			Modified: *file.LastModified,
		}

		// h.SetModTime(*file.LastModified)

		// if file.Modified != "" {
		// 	h.SetModTime(file.ModifiedTime)
		// }

		f, _ := zipWriter.CreateHeader(h)
		// Build safe file file name

		// Read file from S3, log any errors
		_, err = s3downloader.Download(FakeWriterAt{f}, &s3.GetObjectInput{
			Bucket: aws.String(config.Bucket),
			Key:    aws.String(*file.Key),
		})
		if err != nil {
			fmt.Println(err)
		}

	}

	zipWriter.Close()

	log.Printf("%s\t%s\t%s", r.Method, r.RequestURI, time.Since(start))
}

var makeSafeFileName = regexp.MustCompile(`[#<>:"/\|?*\\]`)

func GetLastItem(path string) string {
	// Returns the last item of the prefix
	// should return either the file name or folder

	s := strings.Split(path, "/")

	root_item_name := s[len(s)-1]
	// Added because folders are often sandwitched between two /'s
	if root_item_name == "" {
		root_item_name = s[len(s)-2]
	}

	return root_item_name
}
