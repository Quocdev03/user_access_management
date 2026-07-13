package service

import (
	"strings"
	"testing"
)

func TestParseUserAgent_ChromeWindows(t *testing.T) {
	ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.0.0 Safari/537.36"
	osName, browser, deviceType := parseUserAgent(ua)
	if deviceType != "desktop" {
		t.Fatalf("deviceType=%q want desktop", deviceType)
	}
	if !strings.Contains(strings.ToLower(osName), "windows") {
		t.Fatalf("os=%q want Windows", osName)
	}
	if !strings.Contains(strings.ToLower(browser), "chrome") {
		t.Fatalf("browser=%q want Chrome", browser)
	}
}

func TestParseUserAgent_iPhone(t *testing.T) {
	ua := "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1"
	_, _, deviceType := parseUserAgent(ua)
	if deviceType != "mobile" {
		t.Fatalf("deviceType=%q want mobile", deviceType)
	}
}

func TestParseUserAgent_iPad(t *testing.T) {
	ua := "Mozilla/5.0 (iPad; CPU OS 16_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.0 Mobile/15E148 Safari/604.1"
	_, _, deviceType := parseUserAgent(ua)
	if deviceType != "tablet" && deviceType != "mobile" {
		t.Fatalf("deviceType=%q want tablet or mobile", deviceType)
	}
}

func TestParseUserAgent_Empty(t *testing.T) {
	osName, browser, deviceType := parseUserAgent("")
	if osName != "Unknown" || browser != "Unknown" || deviceType != "desktop" {
		t.Fatalf("empty ua: os=%q browser=%q type=%q", osName, browser, deviceType)
	}
}
