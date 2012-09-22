package main

import (
	"code.google.com/p/go-html-transform/h5"
	"code.google.com/p/go-html-transform/html/transform"
	"fmt"
	"github.com/kierdavis/ansi"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
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
	fmt.Printf("[%s] Fetching %s\n", pluginName, pluginPageURL)

	resp, err := http.Get(pluginPageURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("[%s] HTTP request returned %s", pluginName, resp.Status)
	}

	p := h5.NewParser(resp.Body)
	err = p.Parse()
	if err != nil {
		return "", err
	}

	tree := p.Tree()
	results := DownloadPageURLSelector.Apply(tree)
	if len(results) < 1 {
		return "", fmt.Errorf("[%s] Download page link element was not found (bad selector?)", pluginName)
	}

	el := results[0]
	downloadPageURL = "http://dev.bukkit.org" + GetAttr(el, "href")
	return downloadPageURL, nil
}

func GetDownloadURL(pluginName string, downloadPageURL string) (downloadURL string, err error) {
	fmt.Printf("[%s] Fetching %s\n", pluginName, downloadPageURL)

	resp, err := http.Get(downloadPageURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("[%s] HTTP request returned %s", pluginName, resp.Status)
	}

	p := h5.NewParser(resp.Body)
	err = p.Parse()
	if err != nil {
		return "", err
	}

	tree := p.Tree()
	results := DownloadURLSelector.Apply(tree)
	if len(results) < 1 {
		return "", fmt.Errorf("[%s] Download link element was not found (bad selector?)", pluginName)
	}

	el := results[0]
	downloadURL = GetAttr(el, "href")
	return downloadURL, nil
}

func Download(pluginName string, downloadURL string) (filename string, err error) {
	filename = filepath.Base(downloadURL)
	fmt.Printf("[%s] Fetching %s\n", pluginName, downloadURL)

	resp, err := http.Get(downloadURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("[%s] HTTP request returned %s", pluginName, resp.Status)
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

	fmt.Printf("[%s] Downloading: 0.0%%", pluginName)

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
		fmt.Printf("[%s] Downloading: %.1f%%", pluginName, percentage)
	}

	fmt.Println()
	return filename, nil
}

func GetPlugin(pluginName string) {
	downloadPageURL, err := GetDownloadPageURL(pluginName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		return
	}

	downloadURL, err := GetDownloadURL(pluginName, downloadPageURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		return
	}

	filename, err := Download(pluginName, downloadURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		return
	}

	if strings.HasSuffix(filename, ".zip") {
		fmt.Printf("[%s] Extracting %s\n", pluginName, filename)

		cmd := exec.Command("unzip", filename)
		err = cmd.Run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
			return
		}

		fmt.Printf("[%s] Removing %s\n\n", pluginName, filename)
		err = os.Remove(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
			return
		}

	} else {
		fmt.Printf("[%s] Downloaded to %s\n\n", pluginName, filename)
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: gettit <plugins...>\n")
		os.Exit(2)
	}

	for _, pluginName := range os.Args[1:] {
		GetPlugin(pluginName)
	}
}
