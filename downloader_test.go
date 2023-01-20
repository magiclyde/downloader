/**
 * Created by GoLand.
 * @author: clyde
 * @date: 2021/10/27 下午4:18
 * @note:
 */

package downloader

import (
	"io/ioutil"
	"log"
	"os"
	"testing"
)

func TestDownloader_Run(t *testing.T) {
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
	}
	for _, url := range []string{
		"http://dl.magiclyde.com/mindoc-src.tar.gz",                // Accept-Ranges: none, big file
		"http://dl.magiclyde.com/xhprof-0.9.4.tgz",                 // Accept-Ranges: none
		"http://dl.magiclyde.com/agentzh-nginx-tutorials-zhcn.pdf", // Accept-Ranges: bytes
	} {
		downloader := NewDownloader(url, options...)
		if err := downloader.Run(); err != nil {
			t.Errorf("downloader.Run().err:%s", err.Error())
		}
		t.Logf("ok, %s", url)
	}
}
