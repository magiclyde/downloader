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
	options := []Option{
		WithTotalPart(5),
		WithOutputDir("/tmp"),
	}

	for _, url := range []string{
		"http://dl.magiclyde.com/xhprof-0.9.4.tgz",                        // Accept-Ranges: none
		"https://openresty.org/download/agentzh-nginx-tutorials-zhcn.pdf", // Accept-Ranges: bytes
	} {
		downloader := NewDownloader(url, options...)
		if err := downloader.Run(); err != nil {
			t.Errorf("downloader.Run().err:%s", err.Error())
		}
		t.Logf("ok, %s", url)
	}
}
