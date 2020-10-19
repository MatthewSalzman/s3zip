# s3zip
A Golang microservice that compresses an s3 folder recursively into a single zip or tar file


## How to use
Make an http request to the server. Specify the s3 folder prefix of the file you want to download as an argument to the server

Example : http://0.0.0.0:4005/?prefix=path/to/code/&path=/Download/&comp=tar

### Args
#### prefix (required): The folder path you want to download in your s3 bucket
Notes: 
1. prefix does NOT start with a /
2. Make sure to end prefix with a / to ensure you are downloading that folder

#### path: The folder path you want to write to in your zip file (default is /)

#### comp : The compression type to use on the s3 folder (default is zip)



## Downloading / Installing

### Dev
1. Clone Repo
2. Copy sample_conf.json to conf.json
3. Input correct options
4. Run server > go run main.go

### Prod
Put config file somewhere accessible 
Use docker: 
$ docker build -t "s3zip" .
$ docker run -p 4005:4005 --detach s3zip


## Inspirations & Attribution
[s3zipper](github.com/Teamwork/s3zipper) https://engineroom.teamwork.com/how-to-securely-provide-a-zip-download-of-a-s3-file-bundle-1c3c779ae3f1 
https://dev.to/flowup/using-io-reader-io-writer-in-go-to-stream-data-3i7b


## Other Notes
1. It is recommended to run this service on an EC2 instance because data transfer from S3 to EC2 is *free
2. This isnt built to download individual files, its designed to download s3 folders



