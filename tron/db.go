package tron

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"

	"golang.org/x/crypto/ssh"

	"github.com/boltdb/bolt"
)

var (
	playerBucket = []byte("players")
	configBucket = []byte("config")
	configSSHKey = []byte("ssh-private-key")
)

//store is a storage mechanism for
//various game structs. disk or memory.
type Database struct {
	*bolt.DB
}

func NewDatabase(loc string, reset bool) (*Database, error) {
	b, err := bolt.Open(loc, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("Database error (%s)", err)
	}
	db := &Database{
		DB: b,
	}
	if reset {
		db.Update(func(tx *bolt.Tx) error {
			return tx.DeleteBucket(playerBucket)
		})
	}
	return db, nil
}

func (db *Database) SavePlayer(p *Player) error {
	err := db.Update(func(tx *bolt.Tx) error {
		ps, err := tx.CreateBucketIfNotExists(playerBucket)
		if err != nil {
			return err
		}
		val, err := json.Marshal(p)
		if err != nil {
			return err
		}
		if err := ps.Put(p.dbkey, val); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		// log.Printf("failed to save player scores: %s", p.dbkey)
		return err
	}
	return nil
}

func (db *Database) LoadPlayer(p *Player) error {
	err := db.View(func(tx *bolt.Tx) error {
		ps := tx.Bucket(playerBucket)
		if ps == nil {
			return nil
		}
		val := ps.Get(p.dbkey)
		if val == nil {
			return nil
		}
		if err := json.Unmarshal(val, p); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		// log.Printf("failed to load player scores: %s", p.dbkey)
		return err
	}
	return nil
}

func (db *Database) GetPrivateKey(s *Server) error {
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(configBucket)
		if b == nil {
			return nil
		}
		key := b.Get(configSSHKey)
		if key != nil {
			if p, err := ssh.ParsePrivateKey(key); err == nil {
				s.privateKey = p
				return nil
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if s.privateKey != nil {
		return nil
	}
	val, err := genPrivateKey()
	if err != nil {
		return err
	}
	if p, err := ssh.ParsePrivateKey(val); err == nil {
		s.privateKey = p
	} else {
		return err
	}
	err = db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(configBucket)
		if err != nil {
			return err
		}
		if err := b.Put(configSSHKey, val); err == nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func genPrivateKey() ([]byte, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	ec, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return nil, fmt.Errorf("Unable to marshal ECDSA private key: %v", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: ec}), nil
}
