# s3zip
A Go microservice that zips either a folder recursively or a singular item 


## How to use
Make an Http request to the server. Specify the s3 folder prefix of the file you want to download as an argument to the server

### Args
#### Prefix (required): The folder path you want to download in your s3 bucket
Notes: 
1. prefix does NOT start with a /
2. Make sure to end prefix with a / to endure you are downloading that folder
otherwise folders with the prefix as the start will be downloaded

#### Path: The folder path you want to write to in your zip file (default is /)

Example : http://0.0.0.0:4005/?prefix=path/to/code/&path=/Download/

## Downloading / Installing
1. Clone Repo
2. Copy sample_conf.json to conf.json
3. Input correct options
4. Run server > go run main.go

## Inpirations 
https://engineroom.teamwork.com/how-to-securely-provide-a-zip-download-of-a-s3-file-bundle-1c3c779ae3f1 (github.com/Teamwork/s3zipper)
https://dev.to/flowup/using-io-reader-io-writer-in-go-to-stream-data-3i7b

Note: This app isnt built to download individual files