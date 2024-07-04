package main

import (
	"fmt"
	"net/url"
	"strconv"
	"time"
)

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%dh%dm%ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm%ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

func getPort(urlString string) (int, error) {
	u, err := url.Parse(urlString)
	if err != nil {
		return 0, err
	}

	if u.Port() != "" {
		return strconv.Atoi(u.Port())
	}

	switch u.Scheme {
	case "http":
		return 80, nil
	case "https":
		return 443, nil
	default:
		return 0, fmt.Errorf("unknown scheme: %s", u.Scheme)
	}
}

func getPath(urlString string) (string, error) {
	u, err := url.Parse(urlString)
	if err != nil {
		return "", err
	}

	if u.Path == "" {
		return "/", nil
	}

	return u.Path, nil
}
