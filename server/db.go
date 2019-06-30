package main

import (
	"crypto/rand"
	"github.com/google/logger"
	"github.com/mitchellh/go-homedir"
	"io/ioutil"
	"os"
	"path/filepath"
)

func initDatabase(dataDirPath string) *Database {
	dataDir, err := createDataDir(dataDirPath)
	if err != nil {
		logger.Fatalf("Failed to create data directory: %v\n", err)
	}

	objectMap := make(map[string]bool)
	if err = indexDataDir(objectMap, &dataDir); err != nil {
		logger.Fatalf("Failed to index data directory: %v\n", err)
	}

	return &Database {
		objectMap: objectMap,
		dataDir: dataDir,
	}
}

func createDataDir(path string) (string, error) {
	dataDir, err := homedir.Expand(path)
	if err != nil {
		return "", err
	}
	return dataDir, os.MkdirAll(dataDir, 0770)
}

func indexDataDir(objectMap map[string]bool, dataDir *string) error {
	files, err := ioutil.ReadDir(*dataDir)
	if err != nil {
		return err
	}
	for _, file := range files {
		objectMap[file.Name()] = true
	}

	return nil
}

type Database struct {
	objectMap map[string]bool
	dataDir string
}

func (db *Database) pull(oid string) ([]byte, error) {
	if _, ok := db.objectMap[oid]; !ok {
		return make([]byte, 0), nil
	}

	return db.readObject(oid, &db.dataDir)
}

func (db *Database) drop(bytes []byte) string {
	const oidLen = 16
	const maxOidAttempts = 16

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
	db.writeObject(oid, bytes, &db.dataDir)

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

func (db *Database) writeObject(oid string, data []byte, dataDir *string) {
	if err := ioutil.WriteFile(db.objectPath(oid, dataDir), data, 0660); err != nil {
		logger.Errorf("Failed to write object %s to disk: %v\n", oid, err)
	}
}

func (db *Database) readObject(oid string, dataDir *string) ([]byte, error) {
	data, err := ioutil.ReadFile(db.objectPath(oid, dataDir))
	if err != nil {
		logger.Errorf("Failed to read object %s from disk: %v\n", oid, err)
	}
	return data, err
}

func (db *Database) objectPath(oid string, dataDir *string) string {
	return filepath.Join(*dataDir, oid)
}