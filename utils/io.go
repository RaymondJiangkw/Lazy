// io take the responsibility of manipulating I/O.
package utils

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
)

var Exist error = errors.New("File/Folder exists.")

var ioTokens chan struct{}

var ioInitLock sync.Once

// These two maps of locks are read-only.
var dirMemo *MapMutex
var dirLock map[string]*sync.Mutex
var fileMemo *MapMutex
var fileLock map[string]*sync.Mutex

const (
	ioMaximumRoutines = 100
)

func ioInit() {
	ioTokens = make(chan struct{}, ioMaximumRoutines)
	dirLock = make(map[string]*sync.Mutex)
	dirMemo = NewMapMutex(dirLock)
	fileLock = make(map[string]*sync.Mutex)
	fileMemo = NewMapMutex(fileLock)
}

// mkdir will create the folder if not exists.
// Concurrency-Safe!
// @param folderPath string accept foldername/folderpath. It will be acted upon by filepath.Abs.
func mkdir(folderPath string) (e error) {
	folderPath, e = filepath.Abs(folderPath)
	if e != nil {
		return
	}
	locker := dirMemo.Get(folderPath)
	locker.Lock()
	defer locker.Unlock()
	if _, stat := os.Stat(folderPath); !os.IsNotExist(stat) {
		return Exist
	}
	e = os.MkdirAll(folderPath, 0777)
	return
}

// Concurrency-Safe!
func Mkdir(folderPath string) (e error) {
	ioInitLock.Do(ioInit)
	ioTokens <- struct{}{} // Acquire a Token
	e = mkdir(folderPath)
	<-ioTokens // Release Token
	return
}

// absMkdir will create the folder even if it exists. In such case, absMkdir will delete it first.
// Concurrency-Safe!
// @param folderPath string accept foldername/folderpath. It will be acted upon by filepath.Abs.
func absMkdir(folderPath string) (e error) {
	folderPath, e = filepath.Abs(folderPath)
	if e != nil {
		return
	}
	locker := dirMemo.Get(folderPath)
	locker.Lock()
	defer locker.Unlock()
	if _, stat := os.Stat(folderPath); !os.IsNotExist(stat) {
		e = os.RemoveAll(folderPath)
		if e != nil {
			return
		}
	}
	e = os.MkdirAll(folderPath, 0777)
	return
}

// Concurrency-Safe!
func AbsMkdir(folderPath string) (e error) {
	ioInitLock.Do(ioInit)
	ioTokens <- struct{}{}
	e = absMkdir(folderPath)
	<-ioTokens
	return
}

// touch will create the file if not exists.
// Concurrency-Safe!
// @param filePath string accept filename/filepath (with extension). It will be acted upon by filepath.Abs.
func touch(filePath string) (e error) {
	filePath, e = filepath.Abs(filePath)
	if e != nil {
		return
	}
	locker := fileMemo.Get(filePath)
	locker.Lock()
	defer locker.Unlock()
	if _, stat := os.Stat(filePath); !os.IsNotExist(stat) {
		return Exist
	}
	file, e := os.Create(filePath)
	defer file.Close()
	return
}

// Concurrency-Safe!
func Touch(filePath string) (e error) {
	ioInitLock.Do(ioInit)
	ioTokens <- struct{}{}
	e = touch(filePath)
	<-ioTokens
	return
}

// absTouch will create the file even if it exists. In such case, absTouch will delete it first.
// Concurrency-Safe!
// @param filePath string accept filename/filepath (with extension). It will be acted upon by filepath.Abs.
func absTouch(filePath string) (e error) {
	filePath, e = filepath.Abs(filePath)
	if e != nil {
		return
	}
	locker := fileMemo.Get(filePath)
	locker.Lock()
	defer locker.Unlock()
	if _, stat := os.Stat(filePath); !os.IsNotExist(stat) {
		e = os.Remove(filePath)
		if e != nil {
			return
		}
	}
	file, e := os.Create(filePath)
	defer file.Close()
	return
}

// Concurrency-Safe!
func AbsTouch(filePath string) (e error) {
	ioInitLock.Do(ioInit)
	ioTokens <- struct{}{}
	e = absTouch(filePath)
	<-ioTokens
	return
}

// writeFileString will write string to file.
// Concurrency-Safe!
// @param filePath string accept filename/filepath (with extension). It will be acted upon by filepath.Abs.
func writeFileString(filePath string, data *string) (e error) {
	filePath, e = filepath.Abs(filePath)
	if e != nil {
		return
	}
	locker := fileMemo.Get(filePath)
	locker.Lock()
	defer locker.Unlock()
	if _, stat := os.Stat(filePath); stat != nil {
		return stat
	}
	file, e := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0777)
	defer file.Close()
	if e != nil {
		return
	}
	_, e = file.WriteString(*data)
	return
}

// WriteFileString will create file if not exists.
// Concurrency-Safe!
func WriteFileString(filePath string, data *string, isAppend bool) (e error) {
	ioInitLock.Do(ioInit)
	ioTokens <- struct{}{}
	if !isAppend {
		e = absTouch(filePath)
	} else {
		e = touch(filePath)
	}
	if e == nil || e == Exist {
		e = writeFileString(filePath, data)
	}
	<-ioTokens
	return
}

// writeFileBytes will write []byte to file.
// Concurrency-Safe!
// @param filePath string accept filename/filepath (with extension). It will be acted upon by filepath.Abs.
func writeFileBytes(filePath string, data []byte) (e error) {
	filePath, e = filepath.Abs(filePath)
	if e != nil {
		return
	}
	locker := fileMemo.Get(filePath)
	locker.Lock()
	defer locker.Unlock()
	if _, stat := os.Stat(filePath); stat != nil {
		return stat
	}
	file, e := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0777)
	defer file.Close()
	if e != nil {
		return
	}
	_, e = file.Write(data)
	return
}

// WriteFileBytes will create file if not exists.
// Concurrency-Safe!
func WriteFileBytes(filePath string, data []byte, isAppend bool) (e error) {
	ioInitLock.Do(ioInit)
	ioTokens <- struct{}{}
	if !isAppend {
		e = absTouch(filePath)
	} else {
		e = touch(filePath)
	}
	if e == nil || e == Exist {
		e = writeFileBytes(filePath, data)
	}
	<-ioTokens
	return
}

// Concurrency-Safe!
// @param filePath string accept filename/filepath (with extension). It will be acted upon by filepath.Abs.
func readFileBytes(filePath string) (data []byte, e error) {
	filePath, e = filepath.Abs(filePath)
	if e != nil {
		return
	}
	locker := fileMemo.Get(filePath)
	locker.Lock()
	defer locker.Unlock()
	if _, e = os.Stat(filePath); e != nil {
		return
	}
	file, e := os.Open(filePath)
	defer file.Close()
	if e != nil {
		return
	}
	data, e = ioutil.ReadAll(file)
	return
}

// Concurrency-Safe!
// @param filePath string accept filename/filepath (with extension). It will be acted upon by filepath.Abs.
func readFileString(filePath string) (string, error) {
	bytes, e := readFileBytes(filePath)
	return string(bytes), e
}

// Concurrency-Safe!
func ReadFileBytes(filePath string) (data []byte, e error) {
	ioInitLock.Do(ioInit)
	ioTokens <- struct{}{}
	data, e = readFileBytes(filePath)
	<-ioTokens
	return
}

// Concurrency-Safe!
func ReadFileString(filePath string) (data string, e error) {
	ioInitLock.Do(ioInit)
	ioTokens <- struct{}{}
	data, e = readFileString(filePath)
	<-ioTokens
	return
}

// Concurrency-Safe!
// @param filePath string accept filename/filepath (with extension). It will be acted upon by filepath.Abs.
func removeFile(filePath string) (e error) {
	filePath, e = filepath.Abs(filePath)
	if e != nil {
		return
	}
	locker := fileMemo.Get(filePath)
	locker.Lock()
	defer locker.Unlock()
	if _, e = os.Stat(filePath); e != nil {
		return e
	}
	e = os.Remove(filePath)
	return
}

// Concurrency-Safe!
// @param folderPath string accept foldername/folderpath. It will be acted upon by filepath.Abs.
func removeFolder(folderPath string) (e error) {
	folderPath, e = filepath.Abs(folderPath)
	if e != nil {
		return
	}
	locker := dirMemo.Get(folderPath)
	locker.Lock()
	defer locker.Unlock()
	if _, e = os.Stat(folderPath); e != nil {
		return e
	}
	e = os.RemoveAll(folderPath)
	return
}

// Concurrency-Safe!
func RemoveFile(filepath string) (e error) {
	ioInitLock.Do(ioInit)
	ioTokens <- struct{}{}
	e = removeFile(filepath)
	<-ioTokens
	return
}

// Concurrency-Safe!
func RemoveFolder(folderPath string) (e error) {
	ioInitLock.Do(ioInit)
	ioTokens <- struct{}{}
	e = removeFolder(folderPath)
	<-ioTokens
	return
}
