package log

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"
)

type Journal interface {
	WriteEntry(entry interface{}) bool
	ReadEntries(marshalFactory func() interface{}) []interface{}
}

type FileJournal struct {
	FilePath       string
	LastFSync      int64
	FSyncInterval  int64
	FSyncOnWrite   bool
	LastBackup     int64
	BackupInterval int64
	BackupOnWrite  bool
	MaxBackups     int64
}

func (j *FileJournal) WriteEntry(entry interface{}) bool {
	data, err := json.Marshal(entry)
	//Prepend with data length
	dlen := len(data)
	prep := make([]byte, 4)
	binary.LittleEndian.PutUint32(prep, uint32(dlen))
	data = append(prep, data...)
	if err != nil {
		Error.Printf("Could not prep data: %v", err)
		return false

	}
	var f *os.File
	filename := j.FilePath + "deployd.j001"
	if _, err := os.Stat(filename); !os.IsNotExist(err) {
		if j.BackupOnWrite || (j.BackupInterval > 0 && time.Now().Unix()-j.LastBackup >= j.BackupInterval) {
			j.rotateBackups()
			backup := j.FilePath + "deployd.j002"
			fileCopy(filename, backup)
			j.LastBackup = time.Now().Unix()
		}
		f, err = os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0655)
		if err != nil {
			Error.Printf("Could not open file: %v", err)
			return false
		}
	} else {
		f, err = os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0655)
		if err != nil {
			Error.Printf("Could not create file: %v", err)
			return false
		}
	}

	defer f.Close()

	if _, err = f.Write(data); err != nil {
		Error.Printf("Could not write data: %v", err)
		return false
	}

	if j.FSyncOnWrite || (j.FSyncInterval > 0 && time.Now().Unix()-j.LastFSync >= j.FSyncInterval) {
		// TODO is returning false the correct action to take
		// if we fail an fsync?
		if err = f.Sync(); err != nil {
			Error.Printf("Could not sync file: %v", err)
			return false
		}
		j.LastFSync = time.Now().Unix()
	}
	return true
}

func (j *FileJournal) ReadEntries(marshalFactory func() interface{}) (entries interface{}) {
	filename := j.FilePath + "deployd.j001"
	f, err := os.OpenFile(filename, os.O_RDONLY, 0000)
	defer f.Close()
	if err != nil {
		return nil
	}
	var results []interface{}
	for {
		blen := make([]byte, 4)
		read, err := f.Read(blen)
		if err != nil || read != 4 {
			// TODO decide how to handle errors
			break
		}
		dlen := binary.LittleEndian.Uint32(blen)
		data := make([]byte, dlen)
		read, err = f.Read(data)
		if err != nil || uint32(read) != dlen {
			// TODO decide how to handle errors
			Error.Printf("Failed to read journal data %v. Journal file may be corrupt", err)
			break
		}
		val := marshalFactory()
		if err = json.Unmarshal(data, &val); err != nil {
			Error.Printf("Failed to parse journal data %s: %v. Journal file may be corrupt", data, err)
			break
		}
		results = append(results, val)
	}
	return results
}

func (j *FileJournal) rotateBackups() {
	_, err := os.Stat("deployd.j002")
	if !os.IsNotExist(err) {
		j.rotateFile("deployd.j002")
	}
}

func (j *FileJournal) rotateFile(file string) {
	num, err := strconv.Atoi(file[9:])
	if err != nil {
		Error.Printf("Could not parse journal backup file number %s", file)
		return
	}
	num += 1
	if int64(num) <= j.MaxBackups {

		newfile := "deployd.j" + fmt.Sprintf("%03d", num)
		_, err := os.Stat(newfile)
		if !os.IsNotExist(err) {
			j.rotateFile(newfile)
		}
		os.Rename(j.FilePath+file, j.FilePath+newfile)
	} else {
		os.Remove(j.FilePath + file)
	}

}

func fileCopy(original, backup string) (err error) {
	b, err := os.Stat(backup)
	if err != nil {
		if !os.IsNotExist(err) {
			return
		}
	} else {
		if !(b.Mode().IsRegular()) {
			return fmt.Errorf("Destination is not a file")
		}
	}
	err = fileCopyData(original, backup)
	return
}

func fileCopyData(original, backup string) (err error) {
	f, err := os.Open(original)
	if err != nil {
		return
	}
	defer f.Close()
	b, err := os.Create(backup)
	if err != nil {
		return
	}
	defer func() {
		cerr := b.Close()
		if err == nil {
			err = cerr
		}
	}()
	if _, err = io.Copy(f, b); err != nil {
		return
	}
	err = b.Sync()
	return
}
