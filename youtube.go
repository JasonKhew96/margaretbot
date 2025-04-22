package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

type YoutubeHelper struct {
	client              *http.Client
	checkIsShortsClient *http.Client
	service             *youtube.Service
}

func NewYoutubeHelper() (*YoutubeHelper, error) {
	data, err := os.ReadFile("client_secret.json")
	if err != nil {
		return nil, err
	}
	config, err := google.ConfigFromJSON(data, youtube.YoutubeReadonlyScope)
	if err != nil {
		return nil, err
	}

	tokenCacheDir := "./.credentials"
	os.MkdirAll(tokenCacheDir, 0700)
	tokenFile := filepath.Join(tokenCacheDir, "youtube.json")

	// authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	// fmt.Println("Go to the following link in your browser then type the authorization code:")
	// fmt.Println(authURL)

	// var code string
	// if _, err := fmt.Scan(&code); err != nil {
	// 	return nil, err
	// }
	// token, err := config.Exchange(context.Background(), code)
	// if err != nil {
	// 	return nil, err
	// }
	// f, err := os.OpenFile(tokenFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	// if err != nil {
	// 	return nil, err
	// }
	// defer f.Close()
	// if err := json.NewEncoder(f).Encode(token); err != nil {
	// 	return nil, err
	// }

	f, err := os.Open(tokenFile)
	if err != nil {
		return nil, err
	}
	token := &oauth2.Token{}
	if err := json.NewDecoder(f).Decode(token); err != nil {
		return nil, err
	}

	client := config.Client(context.Background(), token)
	client.Timeout = 10 * time.Second
	service, err := youtube.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	checkShortsClient := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	return &YoutubeHelper{client: client, checkIsShortsClient: checkShortsClient, service: service}, nil
}

func (h *YoutubeHelper) IsShort(videoId string) (bool, error) {
	req, err := http.NewRequest(http.MethodHead, fmt.Sprintf("https://www.youtube.com/shorts/%s", videoId), nil)
	if err != nil {
		return false, err
	}
	resp, err := h.checkIsShortsClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK, nil
}

func (h *YoutubeHelper) Close() {
	token, err := h.client.Transport.(*oauth2.Transport).Source.Token()
	if err != nil {
		log.Println(err)
		return
	}
	f, err := os.Open("./.credentials/youtube.json")
	if err != nil {
		log.Println(err)
		return
	}
	err = json.NewEncoder(f).Encode(token)
	if err != nil {
		log.Println(err)
		return
	}
}
