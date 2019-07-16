package main

import (
	"container/heap"
	"crypto/rand"
	"dead-drop/lib"
	"github.com/google/logger"
	"github.com/mitchellh/go-homedir"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"
)

func initDatabase(dataDirPath string, ttlMin uint) *Database {
	dataDir, err := createDataDir(dataDirPath)
	if err != nil {
		logger.Fatalf("Failed to create data directory: %v\n", err)
	}

	logger.Infof("Starting database with data directory %s\n", dataDir)

	objectMap := make(map[string]bool)
	expHeap := &ExpirationHeap{}
	if err = indexDataDir(objectMap, expHeap, &dataDir); err != nil {
		logger.Fatalf("Failed to index data directory: %v\n", err)
	}
	heap.Init(expHeap)

	db := &Database{
		objectMap: objectMap,
		expHeap:   expHeap,
		dataDir:   dataDir,
		ttlMin:    ttlMin,
	}

	go db.expiryJob()

	return db
}

func createDataDir(path string) (string, error) {
	dataDir, err := homedir.Expand(path)
	if err != nil {
		return "", err
	}
	return dataDir, os.MkdirAll(dataDir, 0770)
}

func indexDataDir(objectMap map[string]bool, expHeap *ExpirationHeap, dataDir *string) error {
	logger.Infof("Indexing data directory for existing objects")

	files, err := ioutil.ReadDir(*dataDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		oid := file.Name()

		objectMap[oid] = true
		expHeap.Push(&ObjectInfo{
			created: file.ModTime(),
			oid:     oid,
		})
	}

	return nil
}

type Database struct {
	lock      sync.Mutex
	objectMap map[string]bool
	expHeap   *ExpirationHeap
	dataDir   string
	ttlMin    uint
}

func (db *Database) expiryJob() {
	for {
		time.Sleep(time.Minute)

		expired := make([]*ObjectInfo, 0)

		db.lock.Lock()
		for !db.expHeap.IsEmpty() && db.expHeap.Peek().IsExpired(db.ttlMin) {
			oi := heap.Pop(db.expHeap).(*ObjectInfo)
			delete(db.objectMap, oi.oid)

			expired = append(expired, oi)
		}
		db.lock.Unlock()

		for _, oi := range expired {
			logger.Infof("Removing expired object %s\n", oi.oid)
			db.removeObject(oi.oid)
		}
	}
}

func (db *Database) pull(oid string) ([]byte, error) {
	db.lock.Lock()
	_, ok := db.objectMap[oid]
	db.lock.Unlock()
	if !ok {
		return nil, nil
	}

	return db.readObject(oid)
}

func (db *Database) drop(bytes []byte) string {
	const oidLen = 16
	const maxOidAttempts = 16

	db.lock.Lock()

	oid := ""
	attempt := 1
	for {
		oid = db.randomOid(oidLen)
		if _, ok := db.objectMap[oid]; !ok {
			break
		}
		attempt++
		if attempt > maxOidAttempts {
			logger.Error("Key-space is very full, data is being overwritten\n")
			break
		}
	}

	db.objectMap[oid] = true
	heap.Push(db.expHeap, &ObjectInfo{
		created: time.Now(),
		oid:     oid,
	})

	db.lock.Unlock()

	db.writeObject(oid, bytes)

	return oid
}

func (db *Database) randomOid(length int) string {
	const characters = "abcdefghijklmnopqrstuvwxyz"

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		logger.Fatalf("Failed to generate random oid: %v\n", err)
	}

	modulo := byte(len(characters))
	for i, b := range bytes {
		bytes[i] = characters[b%modulo]
	}
	return string(bytes)
}

func (db *Database) writeObject(oid string, data []byte) {
	if err := ioutil.WriteFile(db.objectPath(oid), data, lib.ObjectPerms); err != nil {
		logger.Errorf("Failed to write object %s to disk: %v\n", oid, err)
	}
}

func (db *Database) readObject(oid string) ([]byte, error) {
	data, err := ioutil.ReadFile(db.objectPath(oid))
	if err != nil {
		logger.Errorf("Failed to read object %s from disk: %v\n", oid, err)
	}
	return data, err
}

func (db *Database) removeObject(oid string) {
	if err := os.Remove(db.objectPath(oid)); err != nil {
		logger.Errorf("Failed to remove object %s: %v\n", oid, err)
	}
}

func (db *Database) objectPath(oid string) string {
	return filepath.Join(db.dataDir, oid)
}

type ObjectInfo struct {
	created time.Time
	oid     string
}

func (oi *ObjectInfo) IsExpired(ttlMin uint) bool {
	return oi.created.Add(time.Duration(ttlMin) * time.Minute).Before(time.Now())
}

type ExpirationHeap []*ObjectInfo

func (ttlQ ExpirationHeap) Peek() *ObjectInfo {
	return ttlQ[0]
}

func (ttlQ ExpirationHeap) IsEmpty() bool {
	return ttlQ.Len() == 0
}

func (ttlQ ExpirationHeap) Len() int {
	return len(ttlQ)
}

func (ttlQ ExpirationHeap) Less(i, j int) bool {
	return ttlQ[i].created.Before(ttlQ[j].created)
}

func (ttlQ ExpirationHeap) Swap(i, j int) {
	ttlQ[i], ttlQ[j] = ttlQ[j], ttlQ[i]
}

func (ttlQ *ExpirationHeap) Push(x interface{}) {
	*ttlQ = append(*ttlQ, x.(*ObjectInfo))
}

func (ttlQ *ExpirationHeap) Pop() interface{} {
	tempQ := *ttlQ
	newSize := len(tempQ) - 1

	first := tempQ[newSize]
	*ttlQ = tempQ[:newSize]
	return first
}
