/**
 * Created by GoLand.
 * @author: clyde
 * @date: 2021/10/27 下午4:18
 * @note:
 */

package downloader

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"testing"

	"golang.org/x/sync/semaphore"
)

const (
	limit  = 3 // 下载站点有连接数限制，因而设定同時并行运行的 goroutine 上限
	weight = 1 // 每个 goroutine 获取信号量资源的权重
)

func TestDownloader_Run(t *testing.T) {
	parentDir := os.TempDir()
	tmpDir, err := ioutil.TempDir(parentDir, "*-downloader")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	//t.Log(tmpDir)

	s := semaphore.NewWeighted(limit)
	var wg sync.WaitGroup
	options := []Option{
		WithTotalPart(5),
		WithOutputDir(tmpDir),
	}
	for _, url := range []string{
		"http://dl.magiclyde.com/xhprof-0.9.4.tgz",                       // Accept-Ranges: none
		"http://dl.magiclyde.com/mindoc-src.tar.gz",                      // Accept-Ranges: none, big file
		"http://dl.magiclyde.com/php-7.2.30.tar.gz",                      // Accept-Ranges: bytes
		"http://dl.magiclyde.com/openresty-1.19.9.1.tar.gz",              // Accept-Ranges: bytes
		"http://dl.magiclyde.com/tor-browser-linux64-9.0.1_zh-CN.tar.xz", // Accept-Ranges: bytes
	} {
		wg.Add(1)
		go func(url string) {
			s.Acquire(context.Background(), weight)
			downloader := NewDownloader(url, options...)
			if err := downloader.Run(); err != nil {
				t.Errorf("downloader.Run().err:%s", err.Error())
			}
			t.Logf("ok, %s", url)
			s.Release(weight)
			wg.Done()
		}(url)
	}
	wg.Wait()
}
