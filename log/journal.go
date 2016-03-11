package log

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
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
	mutex          *sync.Mutex
}

func (j FileJournal) String() string {
	return fmt.Sprintf("FilePath: %s, FSyncInterval: %d, FSyncOnWrite: %t, BackupInterval: %d, BackupOnWrite: %t, MaxBackups: %d", j.FilePath, j.FSyncInterval, j.FSyncOnWrite, j.BackupInterval, j.BackupOnWrite)
}

func (j FileJournal) WriteEntry(entry interface{}) bool {

	// TODO is this a race condition?
	if j.mutex == nil {
		j.mutex = &sync.Mutex{}
	}

	j.mutex.Lock()
	defer j.mutex.Unlock()

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
	Trace.Printf("Writing journal entry to %s %v", filename, j)
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

func (j FileJournal) ReadEntries(marshalFactory func() interface{}) (entries []interface{}) {
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

func JournalFromConfig(config map[string]interface{}) Journal {
	t, ok := config["type"]
	var journalType string
	if !ok {
		journalType = "file"
	} else {
		journalType = t.(string)
	}

	var j Journal
	switch strings.ToLower(journalType) {
	case "file":
		fj := FileJournal{mutex: &sync.Mutex{}}
		configureFileJournal(&fj, config)
		j = fj
	}

	return j
}

func configureFileJournal(j *FileJournal, config map[string]interface{}) {
	j.FSyncOnWrite = false
	j.FSyncInterval = 300
	j.BackupOnWrite = false
	j.BackupInterval = 3600
	j.FilePath = "/var/lib/deployd/"
	j.MaxBackups = 10
	for key, val := range config {
		switch strings.ToLower(key) {
		case "filepath":
			// TODO ensure filepath ends with a slash
			j.FilePath = val.(string)
		case "sync-on-write":
			j.FSyncOnWrite = val.(bool)
		case "sync-interval":
		case "backup-on-write":
			j.BackupOnWrite = val.(bool)
		case "backup-interval":
			j.BackupInterval = int64(val.(int))
		case "max-backups":
			j.BackupInterval = int64(val.(int))
		}
	}
	d, err := os.Stat(j.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			Warning.Printf("Journal directory does not exist: %s", j.FilePath)
			err := os.Mkdir(j.FilePath, 0600)
			if err != nil {
				Error.Printf("Journal directory does not exist and could not be created %v", err)
			}
		} else {
			Error.Printf("Error reading journal directory %v", err)
		}
	} else {
		if !d.IsDir() {
			Error.Printf("Journal filepath is not a directory: %s", j.FilePath)
		}
	}
	Trace.Printf("Loaded Journal Config: %v", j)
}
