package monitor

import (
	"fmt"
	"testing"
)

func TestGet(t *testing.T) {

}

func TestPut(t *testing.T) {
	SetDbFullPath("/home/joey/monitor_db")
	if levelDB, err := openLevelDB(16, 500); err != nil {
		t.Fatal("init monitor db fail", "err", err)
	} else {
		dbInstance := &monitorDB{path: dbFullPath, levelDB: levelDB, closed: false}
		data, err := dbInstance.Get([]byte("testKey"))
		fmt.Println("data=", data)
		fmt.Println("err=", err)
	}
}
