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
--filename filename, -f filename  Output filename
--dir dir, -d dir                 The destination dir
--concurrency number, -n number   Concurrency number (default: 8)
--proxy URL, -p URL               Proxy URL
--help, -h                        show help
```
