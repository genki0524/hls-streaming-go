package crud

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"cloud.google.com/go/storage"

	"github.com/genki0524/hls_striming_go/utils/program"
)

func uploadFile(w io.Writer, bucket, object string) error {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("storage.NewClient: %w", err)
	}
	defer client.Close()

	f, err := os.Open("notex.txt")
	if err != nil {
		return fmt.Errorf("os.Open: %w", err)
	}
	defer f.Close()

	ctx, cancel := context.WithTimeout(ctx, time.Second*50)
	defer cancel()

	o := client.Bucket(bucket).Object(object)

	o = o.If(storage.Conditions{DoesNotExist: true})

	wc := o.NewWriter(ctx)
	if _, err = io.Copy(wc, f); err != nil {
		return fmt.Errorf("io.Copy: %w", err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("Writer.Close: %w", err)
	}
	fmt.Fprintf(w, "Blob %v uploaded.\n", object)
	return nil
}

func downloadFileIntoMemory(bucket, object string) ([]byte, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("storage.NewClient: %w", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(ctx, time.Second*50)
	defer cancel()

	rc, err := client.Bucket(bucket).Object(object).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("Object(%q).NewReader: %w", object, err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("io.ReadAll: %w", err)
	}
	return data, nil
}

func createSignaturedUrl(bucket, object string) (string, error) {
	expiresTime := 3 * time.Minute
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		fmt.Println(fmt.Errorf("storage.NewClient: %w", err))
		return "", err
	}
	defer client.Close()

	u, err := client.Bucket(bucket).SignedURL(object, &storage.SignedURLOptions{
		Scheme:  storage.SigningSchemeV4,
		Method:  http.MethodGet,
		Expires: time.Now().Add(expiresTime),
	})
	if err != nil {
		fmt.Println(fmt.Errorf("Bucket(%q).SignedURL: %w", bucket, err))
		return "", err
	}
	return u, nil
}

func CreateSignedM3u8(bucket, date string, programName string) (*program.M3U8Playlist, error) {
	resourcePath := date + "/" + programName
	m3u8file, err := downloadFileIntoMemory(bucket, resourcePath+"/video.m3u8")
	if err != nil {
		fmt.Errorf("downloadFileIntoMemory: %v", err)
		return &program.M3U8Playlist{}, err
	}

	playlist, err := program.ParseM3U8File(string(m3u8file))
	if err != nil {
		fmt.Errorf("ParseM3U8File: %v", playlist)
		return &program.M3U8Playlist{}, err
	}

	for index, segment := range playlist.Segments {
		fileName := segment.Filename
		url, err := createSignaturedUrl(bucket, resourcePath+"/"+fileName)
		if err != nil {
			fmt.Errorf("createSignaturedUrl: %v", err)
		}
		playlist.Segments[index].Filename = url
	}
	return playlist, nil
}
