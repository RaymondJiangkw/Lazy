package utils

import (
	"os"
	"path"
	"path/filepath"
	"sync"
)

// Map string -> sync.Mutex

type request struct {
	key      string
	response chan<- *sync.Mutex
}

type MapMutex struct{ requests chan request }

func NewMapMutex(m map[string]*sync.Mutex) *MapMutex {
	mapMutex := &MapMutex{requests: make(chan request)}
	go mapMutex.server(m)
	return mapMutex
}

func (mapMutex *MapMutex) Get(key string) *sync.Mutex {
	response := make(chan *sync.Mutex)
	mapMutex.requests <- request{key, response}
	res := <-response
	return res
}

func (mapMutex *MapMutex) server(m map[string]*sync.Mutex) {
	for req := range mapMutex.requests {
		if _, ok := m[req.key]; !ok {
			m[req.key] = &sync.Mutex{}
		}
		go mapMutex.deliver(m[req.key], req.response)
	}
}

func (mapMutex *MapMutex) deliver(s *sync.Mutex, response chan<- *sync.Mutex) {
	response <- s
}

// Cache, using file

const (
	cacheFolder = ".cache"
)

type FileCache struct {
	rootPath string
	// cursor is a manually fixed `full` path, which will be used as default path when making/writing/getting file.
	cursor string
	valid  bool
	sync.Mutex
}

// @param rootPath accept full/partial path, which will be acted upon by filepath.Abs.
// {@link cursor} will be set to {@link rootPath} by default.
func NewCache(rootPath string) (c *FileCache, e error) {
	rootPath, e = filepath.Abs(rootPath)
	if e != nil {
		return
	}
	e = Mkdir(rootPath)
	if e != nil && e != Exist {
		return
	} else {
		e = nil
	}
	c = &FileCache{rootPath: rootPath, cursor: rootPath, valid: true}
	return
}

// Concurrency-Safe!
// @param folderPath must be a relative path to rootPath.
func (c *FileCache) Mkdir(folderPath string) error {
	if !c.valid {
		return Invalid
	}
	folderPath = path.Join(c.rootPath, folderPath)
	return Mkdir(folderPath)
}

// @param folderPath must be a relative path to rootPath.
func (c *FileCache) SetCursor(folderPath string) (e error) {
	if !c.valid {
		return Invalid
	}
	e = c.Mkdir(folderPath)
	if e != nil && e != Exist {
		return
	}
	folderPath = path.Join(c.rootPath, folderPath)
	c.cursor = folderPath
	return nil
}

func (c *FileCache) GetCursor() (string, error) {
	if !c.valid {
		return "", Invalid
	}
	return c.cursor, nil
}

// @param folderPath can be empty, in which case c.cursor will be used or relative.
func (c *FileCache) WriteBytes(folderPath string, fileName string, data []byte, isAppend bool) error {
	if !c.valid {
		return Invalid
	}
	if folderPath == "" {
		folderPath = c.cursor
	} else if !filepath.IsAbs(folderPath) {
		folderPath = path.Join(c.rootPath, folderPath)
	}
	writePath := path.Join(folderPath, fileName)
	return WriteFileBytes(writePath, data, isAppend)
}

// @param folderPath can be empty, in which case c.cursor will be used or relative.
func (c *FileCache) WriteString(folderPath string, fileName string, data *string, isAppend bool) error {
	if !c.valid {
		return Invalid
	}
	if folderPath == "" {
		folderPath = c.cursor
	} else if !filepath.IsAbs(folderPath) {
		folderPath = path.Join(c.rootPath, folderPath)
	}
	writePath := path.Join(folderPath, fileName)
	return WriteFileString(writePath, data, isAppend)
}

// @param folderPath can be empty, in which case c.cursor will be used or relative.
func (c *FileCache) ReadBytes(folderPath string, fileName string) ([]byte, error) {
	if !c.valid {
		return nil, Invalid
	}
	if folderPath == "" {
		folderPath = c.cursor
	} else if !filepath.IsAbs(folderPath) {
		folderPath = path.Join(c.rootPath, folderPath)
	}
	readPath := path.Join(folderPath, fileName)
	return ReadFileBytes(readPath)
}

// @param folderPath can be empty, in which case c.cursor will be used or relative.
func (c *FileCache) ReadString(folderPath string, fileName string) (string, error) {
	if !c.valid {
		return "", Invalid
	}
	if folderPath == "" {
		folderPath = c.cursor
	} else if !filepath.IsAbs(folderPath) {
		folderPath = path.Join(c.rootPath, folderPath)
	}
	readPath := path.Join(folderPath, fileName)
	return ReadFileString(readPath)
}

// List only reads, thus concurrency-safe.
// @param folderPath string Can be empty, in which case c.cursor will be used or relative.
// @return folderCursor string Place where files are in.
func (c *FileCache) List(folderPath string) (folderCursor string, fileNames []string, totalSize int64, err error) {
	if !c.valid {
		return "", nil, 0, Invalid
	}
	if folderPath == "" {
		folderPath = c.cursor
	} else if !filepath.IsAbs(folderPath) {
		folderPath = path.Join(c.rootPath, folderPath)
	}
	folder, err := os.Open(folderPath)
	if err != nil {
		return "", nil, 0, err
	}
	files, err := folder.Readdir(-1)
	if err != nil {
		return "", nil, 0, err
	}
	fileNames = make([]string, len(files))
	folderCursor = folderPath
	for _, file := range files {
		fileNames = append(fileNames, file.Name())
		totalSize += file.Size()
	}
	return
}
