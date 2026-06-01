package main

import (
	"slices"
	"sync"
	"time"
)

type Article struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Tags        []string  `json:"tags"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (a *Article) PreSave() {
	a.CreatedAt = time.Now()
	a.UpdatedAt = time.Now()
}

type DB struct {
	currentID int
	data      []*Article
	index     map[int]int
	mu        sync.RWMutex
}

func (db *DB) Get(id int) *Article {
	var article *Article
	db.mu.RLock()
	if idx, ok := db.index[id]; ok {
		article = db.data[idx]
	}
	db.mu.RUnlock()
	return article
}

func (db *DB) Save(a *Article) {
	db.mu.Lock()
	db.currentID++
	a.ID = db.currentID
	db.index[a.ID] = len(db.data)
	a.PreSave()
	db.data = append(db.data, a)
	db.mu.Unlock()
}

func (db *DB) Update(a *Article) {
	db.mu.Lock()
	if idx, ok := db.index[a.ID]; ok {
		old := db.data[idx]
		old.Title = a.Title
		old.Description = a.Description
		old.Tags = a.Tags
		old.UpdatedAt = time.Now()
	}
	db.mu.Unlock()
}

func (db *DB) Delete(id int) {
	db.mu.Lock()
	if idx, ok := db.index[id]; ok {
		slices.Delete(db.data, idx, idx)
	}
	delete(db.index, id)
	db.mu.Unlock()
}

func (db *DB) ListByTag(tag string) []*Article {
	var data []*Article
	db.mu.RLock()
	for _, a := range db.data {
		if slices.Contains(a.Tags, tag) {
			data = append(data, a)
		}
	}
	db.mu.RUnlock()
	return data
}

func (db *DB) List() []*Article {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.data
}
