package log

import (
	"fmt"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

type DeploymentTest struct {
	Id            string            `json:"id"`
	PackageId     string            `json:"packageId"`
	StatusMessage string            `json:"statusMessage"`
	Status        string            `json:"status"`
	Variables     map[string]string `json:"replacements"`
	Watch         bool              `json:"watch"`
	Template      string            `json:"template"`
}

func TestJournalWrite(t *testing.T) {
	InitLogger(os.Stdout, os.Stdout, os.Stdout, os.Stderr)
	journal := FileJournal{
		FilePath:      "./",
		FSyncOnWrite:  true,
		BackupOnWrite: true,
		MaxBackups:    5,
	}
	filename := journal.FilePath + "deployd.j001"
	os.Remove(filename)
	vars := make(map[string]string)
	vars["var1"] = "val1"
	vars["var2"] = "val2"
	for i := 0; i < 10; i++ {
		u1 := uuid.NewV4().String()

		deployment := DeploymentTest{Id: u1, PackageId: "test", Status: "NOT STARTED", StatusMessage: "Not Started", Variables: vars, Watch: true}
		ok := journal.WriteEntry(deployment)
		assert.Equal(t, ok, true, "")
	}

	data, err := os.Stat(filename)
	for i := 1; i <= 5; i++ {
		filename := journal.FilePath + fmt.Sprintf("deployd.j%03d", i)
		os.Remove(filename)
	}
	assert.Equal(t, !os.IsNotExist(err), true, "")
	assert.Equal(t, data.Size(), int64(1930), "")

}

func TestJournalRead(t *testing.T) {
	InitLogger(os.Stdout, os.Stdout, os.Stdout, os.Stderr)
	journal := FileJournal{
		FilePath:      "./",
		FSyncOnWrite:  true,
		BackupOnWrite: false,
	}
	filename := journal.FilePath + "deployd.j001"
	os.Remove(filename)
	vars := make(map[string]string)
	vars["var1"] = "val1"
	vars["var2"] = "val2"
	deployments := make([]DeploymentTest, 10)
	for i := 0; i < 10; i++ {
		u1 := uuid.NewV4().String()

		deployment := DeploymentTest{Id: u1, PackageId: "test", Status: "NOT STARTED", StatusMessage: "Not Started", Variables: vars, Watch: true}
		deployments[i] = deployment
		ok := journal.WriteEntry(deployment)
		assert.Equal(t, ok, true, "")
	}

	rawEntries := journal.ReadEntries(func() interface{} {
		return &DeploymentTest{}
	})
	entries := rawEntries.([]interface{})
	assert.Equal(t, len(entries), 10, "")

	for i := 0; i < len(entries); i++ {
		entry := entries[i].(*DeploymentTest)
		assert.Equal(t, entry.PackageId, "test", "")
		assert.Equal(t, entry.Id, deployments[i].Id, "")
	}
	os.Remove(filename)
}
