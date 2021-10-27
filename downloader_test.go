/**
 * Created by GoLand.
 * @author: clyde
 * @date: 2021/10/27 下午4:18
 * @note:
 */

package downloader

import (
	"testing"
)

func TestDownloader_Run(t *testing.T) {
	url := "https://openresty.org/download/agentzh-nginx-tutorials-zhcn.pdf"
	options := []Option{
		WithTotalPart(5),
		WithOutputDir("/tmp"),
		WithOutputFilename("nginx.pdf"),
	}

	downloader := NewDownloader(url, options...)
	if err := downloader.Run(); err != nil {
		t.Errorf("downloader.Run().err:%s", err.Error())
	}
}
