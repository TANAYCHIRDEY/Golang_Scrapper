package main

import (
	"context"
	"fmt"
	"log"
	"mime"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

func upload_on_S3(filename string, fileDest string, data []byte, fileType string) (string, error) {
    if data != nil {
        // Open or create the file with appropriate mode
        mode := os.O_CREATE | os.O_WRONLY
        if fileType == "a" {
            mode = os.O_APPEND | mode
        }

        file, err := os.OpenFile(filename, mode, 0644)
        if err != nil {
            return "", fmt.Errorf("failed to open file: %v", err)
        }
        defer file.Close()

        // Write data to the file
        if _, err = file.Write(data); err != nil {
            return "", fmt.Errorf("failed to write to file: %v", err)
        }
    }

    // Upload the file
    err := uploadFile(filename, fileDest, true)
    if err != nil {
        return "", fmt.Errorf("failed to upload file to S3: %v", err)
    }

    // Get the signed URL
    signedURL, err := GetSignedURL(fileDest, 15, true)
    if err != nil {
        log.Fatalf("Failed to generate signed URL: %v", err)
    }

    // Optionally remove the file after upload
    if err = os.Remove(filename); err != nil {
        log.Printf("Warning: Failed to remove file %s: %v", filename, err)
    }

    return signedURL, nil
}




var AWS_DETAILS = map[string]string{
	"ACCESS_KEY":"AKIAR4FZL3XBUDO3DJNH",
	"SECRET_KEY":           "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",
	"REGION":               "ap-south-1",
	"S3_BUCKET_NAME":       "tracking-tool-dev",
	"S3_PUBLIC_BUCKET_NAME": "cmtb2b",
	"S3_BASE_PATH":         "dev",
}

// uploadFile uploads a file to an S3 bucket
func uploadFile(fileSrc, fileDest string, public bool) error {

	if fileDest == "" {
		fileDest = fileSrc
	}
	fileDest = filepath.Join(AWS_DETAILS["S3_BASE_PATH"], fileDest)
	fileDest = filepath.ToSlash(fileDest)

	// Guess MIME type
	mimetype := getMimeType(fileDest)
	log.Printf("mime-type: %s", mimetype)     

	// Configure S3 client
	cfg, err := config.LoadDefaultConfig(context.TODO(),
    config.WithRegion(AWS_DETAILS["REGION"]),
    config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
        AWS_DETAILS["ACCESS_KEY"],
        AWS_DETAILS["SECRET_KEY"],
        "")), // Session token can be empty if not used
)

	if err != nil {
		log.Printf("Error loading AWS configuration: %v\n", err)
		return err
	}

	s3Client := s3.NewFromConfig(cfg)


	// Set ACL and content type
	acl := types.ObjectCannedACLPrivate
	bucket := AWS_DETAILS["S3_BUCKET_NAME"]
	if public {
		acl = types.ObjectCannedACLPublicRead
		bucket = AWS_DETAILS["S3_BUCKET_NAME"]
	}
    log.Printf("acl value %s:",acl);

	// Open the file
	file, err := os.Open(fileSrc)
	if err != nil {
		log.Printf("Error opening file %s: %v\n", fileSrc, err)
		return err
	}
	defer file.Close()

	// Upload the file
	_, err = s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(fileDest),
		Body:        file,
		ContentType: aws.String("text/plain"),
	})

	if err != nil {
		log.Printf("Error uploading file: %v\n", err)
		return err
	}

	fmt.Printf("File uploaded successfully to %s/%s\n", bucket, fileDest)
	return nil
}

// getMimeType guesses the MIME type of a file and handles charset for text files
func getMimeType(fileSrc string) string {

    ext := filepath.Ext(fileSrc)
    mimetype := mime.TypeByExtension(ext)
    if mimetype == "" {
        if ext == ".html" {
            mimetype = "text/html; charset=utf-8"
        } else if ext == ".txt" {
            mimetype = "text/plain; charset=utf-8"
        } else {
            mimetype = "application/octet-stream" // Default for unknown types
        }
        log.Println("Failed to guess mimetype, defaulting to:", mimetype)
    }
    return mimetype
}


func GetSignedURL(fileSrc string, expiresIn int64, public bool) (string, error) {
    cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(AWS_DETAILS["REGION"]),
        config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
            AWS_DETAILS["ACCESS_KEY"],
            AWS_DETAILS["SECRET_KEY"],
            "")), // Session token can be empty if not used
    )
    if err != nil {
        return "", err
    }

    // Construct the S3 full path
    fileSrc = filepath.Join(AWS_DETAILS["S3_BASE_PATH"], fileSrc)
    fileSrc = filepath.ToSlash(fileSrc) // Ensure correct path format

    s3Client := s3.NewFromConfig(cfg)
    
    req := &s3.GetObjectInput{
        Bucket: aws.String(AWS_DETAILS["S3_BUCKET_NAME"]), // Ensure this is correct
        Key:    aws.String(fileSrc),
    }

    // Create a presign client
    presignClient := s3.NewPresignClient(s3Client)
    presignedReq, err := presignClient.PresignGetObject(context.TODO(), req, s3.WithPresignExpires(time.Duration(expiresIn)*time.Minute))
    if err != nil {
        return "", err
    }

    return presignedReq.URL, nil
}








