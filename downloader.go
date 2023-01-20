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
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/k0kubun/go-ansi"
	"github.com/schollz/progressbar/v3"
)

const MAX_FILE_SIZE = 1024 * 1024 * 100 // 100MB
const BUFFER_SIZE = 1024 * 1024 * 10

var ERR_FILE_IS_INCOMPLETE = errors.New("文件不完整")

// filePart 文件分片
type filePart struct {
	index int    //文件分片的序号
	from  int    //开始 byte
	to    int    //结束 byte
	data  []byte //http 下载得到的文件内容
}

// Downloader 文件下载器
type Downloader struct {
	url            string
	fileSize       int
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
		url:       url,
		totalPart: runtime.NumCPU(),
	}

	for _, o := range options {
		o(d)
	}

	return d
}

func (d *Downloader) Run() error {
	if d.outputFilename == "" {
		d.outputFilename = path.Base(d.url)
	}

	resp, err := d.head()
	if err != nil {
		return err
	}

	if resp.Header.Get("Accept-Ranges") != "bytes" {
		// 服务器不支持文件断点续传, see https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Accept-Ranges
		return d.singleDownload()
	}

	d.fileSize = int(resp.ContentLength)
	return d.multiDownload()
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

	contentLength := int(resp.ContentLength)
	d.setBar(contentLength)

	file := filepath.Join(d.outputDir, d.outputFilename)
	f, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer f.Close()

	buf := make([]byte, BUFFER_SIZE)

	if contentLength <= MAX_FILE_SIZE {
		_, err = io.CopyBuffer(io.MultiWriter(f, d.bar), resp.Body, buf)
		newLine()
		return err
	}

	// 大文件分片逐步写入
	fn := func(f io.Writer, buf []byte, n int) error {
		if n <= 0 {
			return nil
		}
		if _, err := f.Write(buf[:n]); err != nil {
			return err
		}
		if err := d.bar.Add(n); err != nil {
			return err
		}
		newLine()
		return nil
	}
	var totalSize int
	var n int
	for {
		n, err = resp.Body.Read(buf)
		if err != nil {
			if err == io.EOF {
				// got last piece
				err = fn(f, buf, n)
				if err != nil {
					break
				}
				totalSize += n
				err = nil
			}
			break
		}
		err = fn(f, buf, n)
		if err != nil {
			break
		}
		totalSize += n
	}
	if err != nil {
		return err
	}
	if contentLength != totalSize {
		return ERR_FILE_IS_INCOMPLETE
	}
	return nil
}

func (d *Downloader) multiDownload() error {
	d.setBar(d.fileSize)
	d.doneFilePart = make([]filePart, d.totalPart)

	fileParts := make([]filePart, d.totalPart)
	eachSize := d.fileSize / d.totalPart

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

	var wg sync.WaitGroup
	for _, part := range fileParts {
		wg.Add(1)
		go func(part filePart) {
			defer wg.Done()
			if err := d.downloadPart(part); err != nil {
				log.Printf("下载分片文件 %+v 失败了，原因是 %s", part, err.Error())
			}
		}(part)
	}
	wg.Wait()

	return d.mergeFileParts()
}

func (d *Downloader) downloadPart(c filePart) error {
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

	byt, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	partLen := len(byt)
	if partLen != (c.to - c.from + 1) {
		return errors.New("下载文件分片长度错误")
	}
	c.data = byt
	d.doneFilePart[c.index] = c
	d.bar.Add(partLen)
	return nil
}

func (d *Downloader) mergeFileParts() error {
	file := filepath.Join(d.outputDir, d.outputFilename)
	mergedFile, err := os.Create(file)
	if err != nil {
		return err
	}
	defer mergedFile.Close()

	var totalSize int
	for _, s := range d.doneFilePart {
		mergedFile.Write(s.data)
		totalSize += len(s.data)
	}

	if totalSize != d.fileSize {
		return ERR_FILE_IS_INCOMPLETE
	}

	newLine()

	return nil
}

func (d *Downloader) setBar(length int) {
	d.bar = progressbar.NewOptions(
		length,
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

func newLine() {
	fmt.Println("")
}
