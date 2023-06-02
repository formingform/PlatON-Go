package monitor

import (
	"errors"
	"fmt"
	"github.com/PlatONnetwork/PlatON-Go/core"
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
	once       sync.Once
	dbFullPath string
	dbInstance *monitorDB
	blockchain *core.BlockChain
	//ErrNotFound when db not found
	ErrNotFound = errors.New("monitorDB: not found")
)

type monitorDB struct {
	path       string
	levelDB    *leveldb.DB
	closed     bool
	blockchain *core.BlockChain
}

func SetDbFullPath(fullPath string) {
	dbFullPath = fullPath
	log.Info("set monitor db", "path", dbFullPath)
}

func SetBlockChain(c *core.BlockChain) {
	blockchain = c
	log.Info("set blockchain")
}

// monitorDBInstance returns the instance of monitorDB. REMEMBER: call SetDbFullPath() and SetBlockChain() first.
func monitorDBInstance() *monitorDB {
	once.Do(func() {
		if levelDB, err := openLevelDB(16, 500); err != nil {
			log.Crit("init monitor db fail", "err", err)
		} else {
			dbInstance = &monitorDB{path: dbFullPath, levelDB: levelDB, blockchain: blockchain, closed: false}
		}
	})
	return dbInstance
}

func (db *monitorDB) Put(key, value []byte) error {
	err := db.levelDB.Put(key, value, nil)
	if err != nil {
		log.Crit("Failed write to level db", "error", err)
		return err
	}
	return nil
}

func (db *monitorDB) Delete(key []byte) error {
	err := db.levelDB.Delete(key, nil)
	if err != nil {
		log.Crit("Failed delete from level db", "error", err)
		return err
	}
	return nil
}

func (db *monitorDB) Get(key []byte) ([]byte, error) {
	if v, err := db.levelDB.Get(key, nil); err == nil {
		return v, nil
	} else if err != leveldb.ErrNotFound {
		log.Crit("Failed read from level db", "error", err)
		return nil, err
	} else {
		return nil, ErrNotFound
	}
}

func (db *monitorDB) Has(key []byte) (bool, error) {
	_, err := db.Get(key)
	if err == nil {
		return true, nil
	} else if err == ErrNotFound {
		return true, ErrNotFound
	} else {
		return false, err
	}
}

func (db *monitorDB) Close() error {
	if db.levelDB != nil {
		if err := db.levelDB.Close(); err != nil {
			return fmt.Errorf("[statsDB]close level db fail:%v", err)
		}
	}
	db.closed = true
	return nil
}

func openLevelDB(cache int, handles int) (*leveldb.DB, error) {
	db, err := leveldb.OpenFile(dbFullPath, &opt.Options{
		OpenFilesCacheCapacity: handles,
		BlockCacheCapacity:     cache / 2 * opt.MiB,
		WriteBuffer:            cache / 4 * opt.MiB, // Two of these are used internally
		Filter:                 filter.NewBloomFilter(10),
	})
	if err != nil {
		if _, corrupted := err.(*leveldbError.ErrCorrupted); corrupted {
			db, err = leveldb.RecoverFile(dbFullPath, nil)
			if err != nil {
				return nil, fmt.Errorf("[MonitorDB.recover]RecoverFile levelDB fail:%v", err)
			}
		} else {
			return nil, err
		}
	}
	return db, nil
}
