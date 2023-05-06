package monitor

import (
	"fmt"
	"testing"
)

func TestGet(t *testing.T) {

}

func TestPut(t *testing.T) {
	SetDBPath("/home/joey/monitor_db")
	data, err := getMonitorDB().GetLevelDB([]byte("testKey"))
	fmt.Println("data=", data)
	fmt.Println("err=", err)
}
