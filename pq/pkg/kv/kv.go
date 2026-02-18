package kv

import (
	"errors"

	badger "github.com/dgraph-io/badger/v4"
)

// DB wraps a Badger key-value store.
type DB struct{ db *badger.DB }

// Open opens (or creates) a Badger store at dir.
func Open(dir string) (*DB, error) {
	opts := badger.DefaultOptions(dir).WithLogger(nil)
	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}
	return &DB{db}, nil
}

func (d *DB) Close() { d.db.Close() }

// Get returns the value for key, or nil if not found.
func (d *DB) Get(key string) ([]byte, error) {
	var val []byte
	err := d.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}
		val, err = item.ValueCopy(nil)
		return err
	})
	if errors.Is(err, badger.ErrKeyNotFound) {
		return nil, nil
	}
	return val, err
}

// Set stores a key-value pair.
func (d *DB) Set(key string, val []byte) error {
	return d.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), val)
	})
}
