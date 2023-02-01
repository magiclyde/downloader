/**
 * Created by GoLand.
 * @author: clyde
 * @date: 2021/10/27 上午11:15
 * @note: HTTP 断点续传多线程下载大文件
 * @refer:
	https://mojotv.cn/go/go-range-download
	https://polarisxu.studygolang.com/posts/go/action/build-a-concurrent-file-downloader/
*/

package downloader

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/k0kubun/go-ansi"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/sync/errgroup"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
)

const BUFFER_SIZE = 64 * 1024

var ERR_FILE_IS_INCOMPLETE = errors.New("文件不完整")

// filePart 文件分片
type filePart struct {
	index int    // 文件分片的序号
	from  int64  // 开始 byte
	to    int64  // 结束 byte
	data  []byte // http 下载得到的文件内容
}

// Downloader 文件下载器
type Downloader struct {
	url            string
	fileSize       int64
	totalPart      int
	doneFilePart   []filePart
	outputDir      string
	outputFilename string
	proxyUrl       string
	bar            *progressbar.ProgressBar
}

type Option func(*Downloader)

func WithTotalPart(n int) Option {
	return func(d *Downloader) {
		d.totalPart = n
	}
}

func WithOutputDir(dir string) Option {
	return func(d *Downloader) {
		d.outputDir = dir
	}
}

func WithOutputFilename(filename string) Option {
	return func(d *Downloader) {
		d.outputFilename = filename
	}
}

func WithProxyUrl(url string) Option {
	return func(d *Downloader) {
		d.proxyUrl = url
	}
}

func NewDownloader(url string, options ...Option) *Downloader {
	d := &Downloader{
		url:            url,
		outputFilename: path.Base(url),
		totalPart:      runtime.NumCPU(),
	}
	for _, o := range options {
		o(d)
	}
	return d
}

func (d *Downloader) Run() error {
	resp, err := d.head()
	if err != nil {
		return err
	}

	fi, err := os.Stat(d.getAbsFileName())
	if err == nil && fi.Size() == resp.ContentLength {
		log.Println("file already exist")
		return nil
	}

	d.setBar(resp.ContentLength)
	d.fileSize = resp.ContentLength

	if resp.Header.Get("Accept-Ranges") != "bytes" {
		// 服务器不支持文件断点续传, see https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Accept-Ranges
		return d.singleDownload()
	}
	return d.multiDownload()
}

func (d *Downloader) getAbsFileName() string {
	return filepath.Join(d.outputDir, d.outputFilename)
}

func (d *Downloader) getAbsFilePartName(i int) string {
	return filepath.Join(d.outputDir, d.outputFilename+"."+strconv.Itoa(i))
}

func (d *Downloader) head() (resp *http.Response, err error) {
	req, err := d.getNewRequest("HEAD")
	if err != nil {
		return nil, fmt.Errorf("cannot process, getNewRequest.err is %s", err.Error())
	}

	resp, err = d.getHttpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("cannot process, client.Do.err is %s", err.Error())
	}

	if resp.StatusCode > 299 {
		return nil, fmt.Errorf("cannot process, response code is %d", resp.StatusCode)
	}

	return
}

func (d *Downloader) getNewRequest(method string) (*http.Request, error) {
	r, err := http.NewRequest(method, d.url, nil)
	if err != nil {
		return nil, err
	}
	r.Header.Set("User-Agent", "go downloader")
	return r, nil
}

func (d *Downloader) getHttpClient() *http.Client {
	if d.proxyUrl != "" {
		return d.getHttpClientFromProxy(d.proxyUrl)
	}
	return http.DefaultClient
}

func (d *Downloader) getHttpClientFromProxy(givenUrl string) *http.Client {
	proxyUrl, _ := url.Parse(givenUrl)
	tr := &http.Transport{
		Proxy:           http.ProxyURL(proxyUrl),
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	return &http.Client{Transport: tr}
}

func (d *Downloader) singleDownload() error {
	resp, err := d.getHttpClient().Get(d.url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	f, err := os.OpenFile(d.getAbsFileName(), os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer f.Close()

	buf := make([]byte, BUFFER_SIZE)
	_, err = io.CopyBuffer(io.MultiWriter(f, d.bar), resp.Body, buf)
	return err
}

func (d *Downloader) multiDownload() error {
	d.doneFilePart = make([]filePart, d.totalPart)

	fileParts := make([]filePart, d.totalPart)
	eachSize := d.fileSize / int64(d.totalPart)

	for i := range fileParts {
		fileParts[i].index = i
		if i == 0 {
			fileParts[i].from = 0
		} else {
			fileParts[i].from = fileParts[i-1].to + 1
		}
		if i < d.totalPart-1 {
			fileParts[i].to = fileParts[i].from + eachSize
		} else {
			// the last filePart
			fileParts[i].to = d.fileSize - 1
		}
	}

	ctx := context.Background()
	eg, _ := errgroup.WithContext(ctx)
	for _, part := range fileParts {
		part := part
		eg.Go(func() error {
			if err := d.downloadPart(part); err != nil {
				return fmt.Errorf("下载分片文件 %+v 失败了，原因是 %s", part, err.Error())
			}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return err
	}

	return d.mergeFileParts()
}

func (d *Downloader) downloadPart(c filePart) error {
	partLen := c.to - c.from + 1
	partFileName := d.getAbsFilePartName(c.index)

	fi, err := os.Stat(partFileName)
	if err == nil {
		if fi.Size() == partLen {
			return nil
		}
		c.from += fi.Size()
	}

	req, err := d.getNewRequest("GET")
	if err != nil {
		return err
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%v-%v", c.from, c.to))

	client := d.getHttpClient()
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	// https://www.belugacdn.com/http-response-codes/
	if resp.StatusCode > 299 {
		return fmt.Errorf("服务器错误状态码: %d", resp.StatusCode)
	}
	defer resp.Body.Close()

	partFile, err := os.OpenFile(partFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	defer partFile.Close()

	buf := make([]byte, BUFFER_SIZE)
	_, err = io.CopyBuffer(io.MultiWriter(partFile, d.bar), resp.Body, buf)
	return err
}

func (d *Downloader) mergeFileParts() error {
	mergedFile, err := os.Create(d.getAbsFileName())
	if err != nil {
		return err
	}
	defer mergedFile.Close()

	var totalSize int64
	for i := 0; i < d.totalPart; i++ {
		data, err := ioutil.ReadFile(d.getAbsFilePartName(i))
		if err != nil {
			return fmt.Errorf("read part file got err: %s", err.Error())
		}
		n, err := mergedFile.Write(data)
		if err != nil {
			return fmt.Errorf("merge part file got err: %s", err.Error())
		}
		totalSize += int64(n)
	}

	if totalSize != d.fileSize {
		return ERR_FILE_IS_INCOMPLETE
	}

	for i := 0; i < d.totalPart; i++ {
		os.Remove(d.getAbsFilePartName(i))
	}

	return nil
}

func (d *Downloader) setBar(max int64) {
	d.bar = progressbar.NewOptions64(
		max,
		progressbar.OptionSetWriter(ansi.NewAnsiStdout()),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(50),
		progressbar.OptionSetDescription("downloading..."),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
	)
}
