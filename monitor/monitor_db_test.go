package monitor

import (
	"fmt"
	"testing"
)

func TestGet(t *testing.T) {

}

func TestPut(t *testing.T) {
	SetDbFullPath("/home/joey/monitor_db")
	data, err := monitorDBInstance().GetLevelDB([]byte("testKey"))
	fmt.Println("data=", data)
	fmt.Println("err=", err)
}
