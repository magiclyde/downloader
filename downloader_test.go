/**
 * Created by GoLand.
 * @author: clyde
 * @date: 2021/10/27 下午4:18
 * @note:
 */

package downloader

import (
	"context"
	"fmt"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
	"io/ioutil"
	"log"
	"os"
	"testing"
	"time"
)

const (
	limit  = 3 // 并发下载的 goroutine 上限, 因为下载站点有连接数限制
	weight = 1 // 每个 goroutine 获取信号量资源的权重
)

func TestDownloader_Run(t *testing.T) {
	deadline := time.Now().Add(time.Second * 5)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	done := make(chan struct{})

	go func() {
		run(t, ctx, done)
	}()

	select {
	case <-done:
		t.Log("done")
	case <-ctx.Done():
		t.Error(ctx.Err())
	}
}

func run(t *testing.T, ctx context.Context, done chan struct{}) {
	parentDir := os.TempDir()
	tmpDir, err := ioutil.TempDir(parentDir, "*-downloader")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	//t.Log(tmpDir)
	options := []Option{
		WithTotalPart(5),
		WithOutputDir(tmpDir),
		WithContext(ctx),
	}

	eg, _ := errgroup.WithContext(ctx)
	sem := semaphore.NewWeighted(limit)
	for _, url := range []string{
		"http://xxx", // invalid url
		"http://dl.magiclyde.com/xhprof-0.9.4.tgz",                       // Accept-Ranges: none
		"http://dl.magiclyde.com/mindoc-src.tar.gz",                      // Accept-Ranges: none, big file
		"http://dl.magiclyde.com/php-7.2.30.tar.gz",                      // Accept-Ranges: bytes
		"http://dl.magiclyde.com/openresty-1.19.9.1.tar.gz",              // Accept-Ranges: bytes
		"http://dl.magiclyde.com/tor-browser-linux64-9.0.1_zh-CN.tar.xz", // Accept-Ranges: bytes
	} {
		url := url
		eg.Go(func() error {
			defer sem.Release(weight)
			if err := sem.Acquire(ctx, weight); err != nil {
				return fmt.Errorf("failed to acquire semaphore: %s", err.Error())
			}
			if err := NewDownloader(url, options...).Run(); err != nil {
				return fmt.Errorf("downloader.Run got err: %s", err.Error())
			}
			t.Logf("download ok, %s", url)
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		t.Log(err)
	}

	done <- struct{}{}
}
