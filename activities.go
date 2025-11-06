package iplocate

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

type HTTPGetter interface {
	Get(url string) (*http.Response, error)
}

type IPActivities struct {
	HTTPClient HTTPGetter
	mu         sync.Mutex
	cache      map[string]string
}

func (i *IPActivities) GetIP(ctx context.Context) (string, error) {
	resp, err := i.HTTPClient.Get("https://api.ipify.org")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(body)), nil
}

func (i *IPActivities) GetLocationInfo(ctx context.Context, ip string) (string, error) {
	url := "http://ip-api.com/json/" + ip
	fmt.Printf("DEBUG: Fetching location for IP [%s] from URL: %s\n", ip, url)

	resp, err := i.HTTPClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("HTTP GET error: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read body error: %w", err)
	}

	fmt.Printf("DEBUG: Response body: %s\n", string(body))

	var data struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		City    string `json:"city"`
		Region  string `json:"regionName"`
		Country string `json:"country"`
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return "", fmt.Errorf("JSON unmarshal error: %w", err)
	}

	if data.Status == "fail" {
		return "", fmt.Errorf("API error: %s", data.Message)
	}

	fmt.Printf("DEBUG: Parsed data - City: %s, Region: %s, Country: %s\n", data.City, data.Region, data.Country)

	return fmt.Sprintf("City: %s, Region: %s, Country: %s", data.City, data.Region, data.Country), nil
}

func (i *IPActivities) GetTimeZone(ctx context.Context, ip string) (string, error) {
	url := "http://ip-api.com/json/" + ip + "?fields=timezone"

	resp, err := i.HTTPClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("HTTP GET error: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read body error: %w", err)
	}

	var data struct {
		Status   string `json:"status"`
		Message  string `json:"message"`
		Timezone string `json:"timezone"`
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return "", fmt.Errorf("JSON unmarshal error: %w", err)
	}

	if data.Status == "fail" {
		return "", fmt.Errorf("API error: %s", data.Message)
	}

	return data.Timezone, nil
}

func (i *IPActivities) RecordLookup(ctx context.Context, ip string) (string, error) {
	recordId := fmt.Sprintf("%d-%s", time.Now().Unix(), ip)
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.cache == nil {
		i.cache = make(map[string]string)
	}
	i.cache[recordId] = ip
	fmt.Printf("Recorded lookup: %s -> %s\n", recordId, ip)

	return recordId, nil

}

func (i *IPActivities) CompensateLookup(ctx context.Context, recordId string) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	if _, ok := i.cache[recordId]; ok {
		delete(i.cache, recordId)
		fmt.Printf("Compensated lookup, removed record: %s\n", recordId)
	}
	return nil
}
