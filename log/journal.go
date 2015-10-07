package log

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

type Journal interface {
	WriteEntry(entry interface{}) bool
	ReadEntries() []interface{}
}

type FileJournal struct {
	FilePath       string
	LastFSync      int64
	FSyncInterval  int64
	FSyncOnWrite   bool
	LastBackup     int64
	BackupInterval int64
	BackupOnWrite  bool
}

func (j *FileJournal) WriteEntry(entry interface{}) bool {
	data, err := json.Marshal(entry)
	// Use two null bytes as our separator
	// TODO Figure out how to to just read out individual json
	// objects without having a separator
	data = append(data, 0x00, 0x00)
	if err != nil {
		return false
	}
	filename := j.FilePath + "/deployd.001"
	if j.BackupOnWrite || time.Now().Unix()-j.LastBackup >= j.BackupInterval {
		backup := j.FilePath + "/deployd.002"
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			fileCopy(filename, backup)
			j.LastBackup = time.Now().Unix()
		} else {
			// TODO Is returning false the correct action to take
			// on backup failure?
			return false
		}
	}
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0655)
	if err != nil {
		return false
	}

	defer f.Close()

	if _, err = f.Write(data); err != nil {
		return false
	}

	if j.FSyncOnWrite || time.Now().Unix()-j.LastFSync >= j.FSyncInterval {
		// TODO is returning false the correct action to take
		// if we fail an fsync?
		if err = f.Sync(); err != nil {
			return false
		}
		j.LastFSync = time.Now().Unix()
	}
	return true
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
