package main

import (
	"code.google.com/p/go-html-transform/h5"
	"code.google.com/p/go-html-transform/html/transform"
	"fmt"
	"github.com/kierdavis/ansi"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

var DownloadPageURLSelector = transform.NewSelectorQuery("html", "body", "section", "div", "section", "div", "div", "div", ".unit size1of3 lastUnit", "ul", ".user-action user-action-download", "a")
var DownloadURLSelector = transform.NewSelectorQuery("html", "body", "section", "div", "section", "div", "div", "section", "header", "div", ".unit size1of3 lastUnit", "ul", "li", "span", "a")

func GetAttr(node *h5.Node, name string) (value string) {
	for _, attr := range node.Attr {
		if attr.Name == name {
			return attr.Value
		}
	}

	return ""
}

func GetDownloadPageURL(pluginName string) (downloadPageURL string, err error) {
	pluginPageURL := "http://dev.bukkit.org/server-mods/" + pluginName + "/"
	fmt.Printf("Fetching %s\n", pluginPageURL)

	resp, err := http.Get(pluginPageURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("HTTP request returned %s", resp.Status)
	}

	p := h5.NewParser(resp.Body)
	err = p.Parse()
	if err != nil {
		return "", err
	}

	tree := p.Tree()
	results := DownloadPageURLSelector.Apply(tree)
	if len(results) < 1 {
		return "", fmt.Errorf("Download page link element was not found (bad selector?)")
	}

	el := results[0]
	downloadPageURL = "http://dev.bukkit.org" + GetAttr(el, "href")
	return downloadPageURL, nil
}

func GetDownloadURL(downloadPageURL string) (downloadURL string, err error) {
	fmt.Printf("Fetching %s\n", downloadPageURL)

	resp, err := http.Get(downloadPageURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("HTTP request returned %s", resp.Status)
	}

	p := h5.NewParser(resp.Body)
	err = p.Parse()
	if err != nil {
		return "", err
	}

	tree := p.Tree()
	results := DownloadURLSelector.Apply(tree)
	if len(results) < 1 {
		return "", fmt.Errorf("Download link element was not found (bad selector?)")
	}

	el := results[0]
	downloadURL = GetAttr(el, "href")
	return downloadURL, nil
}

func Download(downloadURL string) (filename string, err error) {
	filename = filepath.Base(downloadURL)
	fmt.Printf("Fetching %s\n", downloadURL)

	resp, err := http.Get(downloadURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("HTTP request returned %s", resp.Status)
	}

	file, err := os.Create(filename)
	if err != nil {
		return "", err
	}

	var bytesDownloaded uint64

	totalSize, err := strconv.ParseUint(resp.Header.Get("Content-Length"), 10, 64)
	if err != nil {
		return "", err
	}

	chunk := make([]byte, 1024*8)

	fmt.Printf("Downloading: 0.0%%")

	for {
		n, err := resp.Body.Read(chunk)
		if err != nil {
			if err != io.EOF {
				fmt.Println()
				return "", err
			}

			break
		}

		_, err = file.Write(chunk[:n])
		if err != nil {
			fmt.Println()
			return "", err
		}

		bytesDownloaded += uint64(n)
		percentage := (float32(bytesDownloaded) / float32(totalSize)) * 100.0

		ansi.ClearLine()
		ansi.CursorHozPosition(1)
		fmt.Printf("Downloading: %.1f%%", percentage)
	}

	fmt.Println()
	return filename, nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: gettit <plugin-name>\n")
		os.Exit(2)
	}

	pluginName := os.Args[1]
	downloadPageURL, err := GetDownloadPageURL(pluginName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(1)
	}

	downloadURL, err := GetDownloadURL(downloadPageURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(1)
	}

	filename, err := Download(downloadURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(1)
	}

	fmt.Printf("Downloaded to %s\n", filename)
}
