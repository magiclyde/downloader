/**
 * Created by GoLand.
 * @author: clyde
 * @date: 2021/10/27 下午4:56
 * @note:
 */

package main

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"runtime"

	. "github.com/magiclyde/downloader"
	"github.com/urfave/cli/v2"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// 默认并发数
	concurrencyN := runtime.NumCPU()

	app := &cli.App{
		Name:  "downloader",
		Usage: "File concurrency downloader",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "filename",
				Aliases: []string{"f"},
				Usage:   "Output `filename`",
			},
			&cli.StringFlag{
				Name:    "dir",
				Aliases: []string{"d"},
				Usage:   "The destination `dir`",
			},
			&cli.IntFlag{
				Name:    "concurrency",
				Aliases: []string{"n"},
				Value:   concurrencyN,
				Usage:   "Concurrency `number`",
			},
			&cli.StringFlag{
				Name:    "proxy",
				Aliases: []string{"p"},
				Usage:   "Proxy `URL`",
			},
		},
		Action: func(c *cli.Context) error {
			givenUrl := c.Args().First()
			if len(givenUrl) < len("http://") {
				return errors.New("url is invalid")
			}
			_, err := url.ParseRequestURI(givenUrl)
			if err != nil {
				return errors.New("url is invalid")
			}

			filename := c.String("filename")
			dir := c.String("dir")
			concurrency := c.Int("concurrency")
			proxyUrl := c.String("proxy")
			var options []Option
			options = append(options, WithTotalPart(concurrency))
			if filename != "" {
				options = append(options, WithOutputFilename(filename))
			}
			if dir != "" {
				options = append(options, WithOutputDir(dir))
			}
			if proxyUrl != "" {
				options = append(options, WithProxyUrl(proxyUrl))
			}

			return NewDownloader(givenUrl, options...).Run()
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("")
}
