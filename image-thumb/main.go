package main

import (
	"bytes"
	"context"
	"image/jpeg"
	"log"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"github.com/nfnt/resize"
)

// Handler is our lambda handler invoked by the `lambda.Start` function call
func Handler(ctx context.Context, event events.S3Event) {
	for _, record := range event.Records {
		// Set variables
		bucket := record.S3.Bucket.Name
		key := record.S3.Object.Key

		session := session.New()
		buffer := &aws.WriteAtBuffer{}

		// Prevent recursive Lambda trigger
		if strings.Contains(key, "_thumb.") {
			continue
		}

		// Download from s3
		downloader := s3manager.NewDownloader(session)
		_, err := downloader.Download(buffer, &s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		})
		if err != nil {
			log.Printf("Could not download from S3: %v", err)
		}

		// Decode image into bytes
		reader := bytes.NewReader(buffer.Bytes())
		img, err := jpeg.Decode(reader)
		if err != nil {
			log.Printf("bad response: %s", err)
		}

		// Create thumbnail
		thumbnail := resize.Thumbnail(600, 600, img, resize.Lanczos2)
		if thumbnail == nil {
			log.Printf("resize thumbnail returned nil")
		}

		// Encode bytes into image
		uBuffer := new(bytes.Buffer)
		err = jpeg.Encode(uBuffer, thumbnail, nil)
		if err != nil {
			log.Printf("JPEG encoding error: %v", err)
		}

		// Upload thumbnail into s3
		thumbkey := strings.Replace(key, ".", "_thumb.", -1)

		uploader := s3manager.NewUploader(session)
		result, err := uploader.Upload(&s3manager.UploadInput{
			Body:   bytes.NewReader(uBuffer.Bytes()),
			Bucket: aws.String(bucket),
			Key:    aws.String(thumbkey),
		})
		if err != nil {
			log.Printf("Failed to upload: %v", err)
		}

		log.Printf("Successfully uploaded to: %v", result.Location)
	}
}

func main() {
	lambda.Start(Handler)
}
