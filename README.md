### File concurrency downloader

#### install
```markdown
$ git clone https://github.com/magiclyde/downloader.git && cd downloader
$ make
```

#### usage 
```markdown
$ ./bin/downloader -h
NAME:
   downloader - File concurrency downloader

USAGE:
   downloader [global options] command [command options] [arguments...]

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --url URL, -u URL                 URL to download
   --filename filename, -f filename  Output filename
   --dir dir, -d dir                 Output dir
   --concurrency number, -n number   Concurrency number (default: 8)
   --proxy value, -p value           Proxy url
   --help, -h                        show help (default: false)

```