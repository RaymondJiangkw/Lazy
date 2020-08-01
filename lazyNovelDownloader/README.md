# NovelDownloader
![Demo](./demo.gif)

lazyNovelDownloader is a tool for downloading available texts from websites.

## Declaration
>This software does not provide any text by itself. Rights of texts are all reserved for original authors, who have the rights to require users to delete their works.
>Any conflict, dispute and lawsuit resulted from texts are all attributed to users. RaymondJiangkw does not take any responsibility of it.
> 
>Copyright © 2020 RaymondJiangkw. All rights reserved.

## Building
```shell
$ go get -v github.com/RaymondJiangkw/Lazy/lazyNovelDownloader
$ go build lnd.go
```

## Usage
| Command | Description                        | Optional | Default               |
| ------- | ---------------------------------- | -------- | --------------------- |
| name    | Novel Name                         | false    |                       |
| source  | URL for Catalog Html File of Novel | false    |                       |
| author  | Novel Author                       | true     | ""                    |
| format  | txt/epub                           | true     | txt                   |
| o       | Output File Name(can include path) | true     | Arg of `name` command |
| h/help  | Log Help                           |          |                       |

## Acknowledge
* `bmaupin/go-epub`: used to generate `.epub` file.
* `mvdan/xurls`: used to delete `url` from text.
* [Ans in Stack Overflow](https://stackoverflow.com/questions/53666867/after-called-peek-method-the-origin-data-has-changed): used to decode html file.

## TODO
* Automatically detect potential websites for catalog, only given name of Novel.
* Develop mode: `<f> Fast` and `<q> Quality` for choosing appropriate catalog website.
* Compare catalogs between different websites, and Assess contents provided by different websites to collect high-quality text.

## Log
* 2020/08/01 Release Alpha Version.