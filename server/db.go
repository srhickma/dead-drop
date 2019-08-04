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

// TODO(shane) make these configurable.
const heapCleanThresholdNumber = 4096
const heapCleanThresholdPercent = 0.5

func initDatabase(dataDirPath string, ttlMin uint, destructiveRead bool) *Database {
	dataDir, err := createDataDir(dataDirPath)
	if err != nil {
		logger.Fatalf("Failed to create data directory: %v", err)
	}

	logger.Infof("Starting database with data directory %s", dataDir)

	objectMap := make(map[string]bool)
	expHeap := &ExpirationHeap{}
	if err = indexDataDir(objectMap, expHeap, &dataDir); err != nil {
		logger.Fatalf("Failed to index data directory: %v", err)
	}
	heap.Init(expHeap)

	lock := &sync.RWMutex{}

	db := &Database{
		lock:             lock,
		objectMap:        objectMap,
		expHeap:          expHeap,
		heapCleanCond:    sync.NewCond(lock),
		dirtyHeapBlocks:  0,
		heapCleanPending: false,
		dataDir:          dataDir,
		ttlMin:           ttlMin,
		destructiveRead:  destructiveRead,
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
	lock             *sync.RWMutex
	objectMap        map[string]bool
	expHeap          *ExpirationHeap
	heapCleanCond    *sync.Cond
	dirtyHeapBlocks  uint
	heapCleanPending bool
	dataDir          string
	ttlMin           uint
	destructiveRead  bool
}

func (db *Database) pull(oid string) ([]byte, error) {
	db.lock.RLock()
	_, ok := db.objectMap[oid]
	db.lock.RUnlock()
	if !ok {
		return nil, nil
	}

	data, err := db.readObject(oid)

	if db.destructiveRead {
		go db.destroyObject(oid)
	}

	return data, err
}

func (db *Database) drop(bytes []byte) string {
	const oidLen = 16
	const maxOidAttempts = 16

	db.lock.Lock()

	for db.heapCleanPending {
		db.heapCleanCond.Wait()
	}

	oid := ""
	attempt := 1
	for {
		oid = db.randomOid(oidLen)
		if _, ok := db.objectMap[oid]; !ok {
			break
		}
		attempt++
		if attempt > maxOidAttempts {
			logger.Error("Key-space is very full, data is being overwritten")
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

func (db *Database) expiryJob() {
	for {
		time.Sleep(time.Minute)

		expired := make([]*ObjectInfo, 0)

		db.lock.Lock()

		for db.heapCleanPending {
			db.heapCleanCond.Wait()
		}

		for !db.expHeap.IsEmpty() && db.expHeap.Peek().IsExpired(db.ttlMin) {
			oi := heap.Pop(db.expHeap).(*ObjectInfo)

			if _, ok := db.objectMap[oi.oid]; ok {
				delete(db.objectMap, oi.oid)
				expired = append(expired, oi)
			} else {
				db.dirtyHeapBlocks -= 1
			}
		}
		db.lock.Unlock()

		for _, oi := range expired {
			logger.Infof("Removing expired object %s", oi.oid)
			db.removeObject(oi.oid)
		}
	}
}

func (db *Database) heapCleanerJob() {
	logger.Infof("Starting heap cleaner job")

	db.lock.RLock()

	dirtyBlocks := db.dirtyHeapBlocks
	newHeap := make([]*ObjectInfo, uint(db.expHeap.Len())-dirtyBlocks)

	newCursor := 0
	oldCursor := 0
	for oldCursor < db.expHeap.Len() {
		current := (*db.expHeap)[oldCursor]

		if _, ok := db.objectMap[current.oid]; ok {
			newHeap[newCursor] = current
			newCursor += 1
		}

		oldCursor += 1
	}

	// Mind the lock gap.
	db.lock.RUnlock()

	logger.Infof("Finished heap compaction")

	newExpHeap := ExpirationHeap(newHeap)

	db.lock.Lock()

	db.expHeap = &newExpHeap
	db.dirtyHeapBlocks -= dirtyBlocks
	db.heapCleanPending = false

	db.lock.Unlock()

	logger.Infof("Finished swap to compacted heap")
}

func (db *Database) destroyObject(oid string) {
	shouldStartHeapCleaner := false

	db.lock.Lock()

	delete(db.objectMap, oid)
	db.dirtyHeapBlocks += 1

	pastNumberThreshold := db.dirtyHeapBlocks > heapCleanThresholdNumber
	pastPercentageThreshold := float32(db.dirtyHeapBlocks)/float32(db.expHeap.Len()) > heapCleanThresholdPercent
	if !db.heapCleanPending && pastNumberThreshold && pastPercentageThreshold {
		db.heapCleanPending = true
		shouldStartHeapCleaner = true
	}

	db.lock.Unlock()

	if shouldStartHeapCleaner {
		go db.heapCleanerJob()
	}

	db.removeObject(oid)
}

func (db *Database) randomOid(length int) string {
	const characters = "abcdefghijklmnopqrstuvwxyz"

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		logger.Fatalf("Failed to generate random oid: %v", err)
	}

	modulo := byte(len(characters))
	for i, b := range bytes {
		bytes[i] = characters[b%modulo]
	}
	return string(bytes)
}

func (db *Database) writeObject(oid string, data []byte) {
	if err := ioutil.WriteFile(db.objectPath(oid), data, lib.ObjectPerms); err != nil {
		logger.Errorf("Failed to write object %s to disk: %v", oid, err)
	}
}

func (db *Database) readObject(oid string) ([]byte, error) {
	data, err := ioutil.ReadFile(db.objectPath(oid))
	if err != nil {
		logger.Errorf("Failed to read object %s from disk: %v", oid, err)
	}
	return data, err
}

func (db *Database) removeObject(oid string) {
	if err := os.Remove(db.objectPath(oid)); err != nil {
		logger.Errorf("Failed to remove object %s: %v", oid, err)
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
