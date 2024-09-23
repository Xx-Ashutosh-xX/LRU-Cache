package main

import (
    "container/list"
    "encoding/json"
    "net/http"
    "sync"
    "time"
)

// CacheItem represents a single cache entry
type CacheItem struct {
    key        string
    value      string
    expiration time.Time
}

// LRUCache represents a thread-safe LRU cache
type LRUCache struct {
    capacity int
    cache    map[string]*list.Element
    list     *list.List
    mutex    sync.Mutex
}

// NewLRUCache creates a new LRUCache
func NewLRUCache(capacity int) *LRUCache {
    return &LRUCache{
        capacity: capacity,
        cache:    make(map[string]*list.Element),
        list:     list.New(),
    }
}

// Get retrieves a value from the cache
func (c *LRUCache) Get(key string) (string, bool) {
    c.mutex.Lock()
    defer c.mutex.Unlock()

    if elem, found := c.cache[key]; found {
        item := elem.Value.(*CacheItem)
        if time.Now().After(item.expiration) {
            c.list.Remove(elem)
            delete(c.cache, key)
            return "", false
        }
        c.list.MoveToFront(elem)
        return item.value, true
    }
    return "", false
}

// Set adds a value to the cache
func (c *LRUCache) Set(key string, value string, expiration time.Duration) {
    c.mutex.Lock()
    defer c.mutex.Unlock()

    if elem, found := c.cache[key]; found {
        c.list.MoveToFront(elem)
        elem.Value.(*CacheItem).value = value
        elem.Value.(*CacheItem).expiration = time.Now().Add(expiration)
        return
    }

    if c.list.Len() >= c.capacity {
        oldest := c.list.Back()
        if oldest != nil {
            c.list.Remove(oldest)
            delete(c.cache, oldest.Value.(*CacheItem).key)
        }
    }

    item := &CacheItem{
        key:        key,
        value:      value,
        expiration: time.Now().Add(expiration),
    }
    elem := c.list.PushFront(item)
    c.cache[key] = elem
}

var cache = NewLRUCache(1024)

// CacheRequest represents the expected structure of a cache set request
type CacheRequest struct {
    Key        string `json:"key"`
    Value      string `json:"value"`
    Expiration int    `json:"expiration"`
}

// enableCors sets CORS headers to the response
func enableCors(w *http.ResponseWriter) {
    (*w).Header().Set("Access-Control-Allow-Origin", "*")
    (*w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
    (*w).Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

// getCacheHandler handles GET requests for retrieving cache data
func getCacheHandler(w http.ResponseWriter, r *http.Request) {
    enableCors(&w) // Enable CORS
    key := r.URL.Query().Get("key")
    if value, found := cache.Get(key); found {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(value))
    } else {
        http.Error(w, "Key not found", http.StatusNotFound)
    }
}

// setCacheHandler handles POST requests for setting cache data
func setCacheHandler(w http.ResponseWriter, r *http.Request) {
    enableCors(&w) // Enable CORS
    var req CacheRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Bad request", http.StatusBadRequest)
        return
    }

    expiration := time.Duration(req.Expiration) * time.Second
    cache.Set(req.Key, req.Value, expiration)
    w.WriteHeader(http.StatusOK)
}

func main() {
    http.HandleFunc("/cache", func(w http.ResponseWriter, r *http.Request) {
        enableCors(&w) // Enable CORS

        switch r.Method {
        case "GET":
            getCacheHandler(w, r)
        case "POST":
            setCacheHandler(w, r)
        case "OPTIONS":
            w.WriteHeader(http.StatusOK) // Handle preflight requests for CORS
        default:
            http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        }
    })

    http.ListenAndServe(":8080", nil)
}