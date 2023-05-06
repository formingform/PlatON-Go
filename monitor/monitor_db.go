package monitor

import (
	"errors"
	"fmt"
	"sync"

	leveldbError "github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/opt"

	"github.com/PlatONnetwork/PlatON-Go/log"
	"github.com/syndtr/goleveldb/leveldb"
)

const (
	DBPath = "monitordb"
)

var (
	dbpath string

	dbInstance *MonitorDB

	instance sync.Mutex

	levelDBcache   = int(16)
	levelDBhandles = int(500)

	logger = log.Root().New("package", "statsdb")

	//ErrNotFound when db not found
	ErrNotFound = errors.New("statsDB: not found")
)

type MonitorDB struct {
	path    string
	levelDB *leveldb.DB
	closed  bool
	dbError error
}

func (db *MonitorDB) PutLevelDB(key, value []byte) error {
	err := db.levelDB.Put(key, value, nil)
	if err != nil {
		log.Crit("Failed write to level db", "error", err)
		return err
	}
	return nil
}

func (db *MonitorDB) DeleteLevelDB(key []byte) error {
	err := db.levelDB.Delete(key, nil)
	if err != nil {
		log.Crit("Failed delete from level db", "error", err)
		return err
	}
	return nil
}

func (db *MonitorDB) GetLevelDB(key []byte) ([]byte, error) {
	if v, err := db.levelDB.Get(key, nil); err == nil {
		return v, nil
	} else if err != leveldb.ErrNotFound {
		log.Crit("Failed read from level db", "error", err)
		return nil, err
	} else {
		return nil, ErrNotFound
	}
}

func (db *MonitorDB) HasLevelDB(key []byte) (bool, error) {
	_, err := db.GetLevelDB(key)
	if err == nil {
		return true, nil
	} else if err == ErrNotFound {
		return true, ErrNotFound
	} else {
		return false, err
	}
}

func (db *MonitorDB) Close() error {
	if db.levelDB != nil {
		if err := db.levelDB.Close(); err != nil {
			return fmt.Errorf("[statsDB]close level db fail:%v", err)
		}
	}
	db.closed = true
	return nil
}

func SetDBPath(path string) {
	dbpath = path
	logger.Info("set path", "path", dbpath)
}

func SetDBOptions(cache int, handles int) {
	levelDBcache = cache
	levelDBhandles = handles
}

func getMonitorDB() *MonitorDB {
	instance.Lock()
	defer instance.Unlock()
	if dbInstance == nil || dbInstance.closed {
		logger.Debug("dbInstance is nil", "path", dbpath)
		if dbInstance == nil {
			dbInstance = new(MonitorDB)
		}
		if levelDB, err := openLevelDB(levelDBcache, levelDBhandles); err != nil {
			logger.Error("init db fail", "err", err)
			panic(err)
		} else {
			dbInstance.levelDB = levelDB
		}
	}
	return dbInstance
}

func openLevelDB(cache int, handles int) (*leveldb.DB, error) {
	db, err := leveldb.OpenFile(dbpath, &opt.Options{
		OpenFilesCacheCapacity: handles,
		BlockCacheCapacity:     cache / 2 * opt.MiB,
		WriteBuffer:            cache / 4 * opt.MiB, // Two of these are used internally
		Filter:                 filter.NewBloomFilter(10),
	})
	if err != nil {
		if _, corrupted := err.(*leveldbError.ErrCorrupted); corrupted {
			db, err = leveldb.RecoverFile(dbpath, nil)
			if err != nil {
				return nil, fmt.Errorf("[MonitorDB.recover]RecoverFile levelDB fail:%v", err)
			}
		} else {
			return nil, err
		}
	}
	return db, nil
}
