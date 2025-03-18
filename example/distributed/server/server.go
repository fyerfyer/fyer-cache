package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type CacheRequest struct {
	Key        string        `json:"key"`
	Value      interface{}   `json:"value,omitempty"`
	Expiration time.Duration `json:"expiration,omitempty"`
}

type CacheResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Value   interface{} `json:"value,omitempty"`
}

// 创建一个简单的内存缓存
var memCache = make(map[string]interface{})
var cacheLock = &sync.Mutex{}

func main() {
	// 设置自定义路由
	mux := http.NewServeMux()

	// 添加日志中间件
	loggingHandler := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			log.Printf("REQUEST: %s %s", r.Method, r.URL.Path)

			// 读取和记录请求体
			if r.Body != nil {
				bodyBytes, _ := ioutil.ReadAll(r.Body)
				if len(bodyBytes) > 0 {
					log.Printf("REQUEST BODY: %s", string(bodyBytes))
					// 重新设置请求体，因为读取后会消耗
					r.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
				}
			}

			h.ServeHTTP(w, r)
			log.Printf("RESPONSE: %s %s - completed in %v", r.Method, r.URL.Path, time.Since(start))
		})
	}

	// 捕获所有请求的处理器
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("DEBUG: Received unhandled request: %s %s", r.Method, r.URL.Path)
		http.NotFound(w, r)
	})

	// 修改为匹配客户端期望的路径

	// GET /cache/get - 获取缓存值
	mux.HandleFunc("/cache/get", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Handling GET request at /cache/get")
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req CacheRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		key := req.Key
		if key == "" {
			http.Error(w, "Key is required", http.StatusBadRequest)
			return
		}

		log.Printf("Looking up key: %s", key)

		cacheLock.Lock()
		value, exists := memCache[key]
		cacheLock.Unlock()

		if !exists {
			writeJSONResponse(w, CacheResponse{
				Success: false,
				Message: "Key not found",
			})
			return
		}

		writeJSONResponse(w, CacheResponse{
			Success: true,
			Value:   value,
		})
	})

	// POST /cache/set - 设置缓存值
	mux.HandleFunc("/cache/set", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Handling POST request at /cache/set")
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req CacheRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Key == "" {
			http.Error(w, "Key is required", http.StatusBadRequest)
			return
		}

		cacheLock.Lock()
		memCache[req.Key] = req.Value
		cacheLock.Unlock()

		log.Printf("Set key: %s, value: %v", req.Key, req.Value)

		writeJSONResponse(w, CacheResponse{
			Success: true,
			Message: "Key set successfully",
		})
	})

	// DELETE /cache/delete - 删除缓存值
	mux.HandleFunc("/cache/delete", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Handling DELETE request at /cache/delete")
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req CacheRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		key := req.Key
		if key == "" {
			http.Error(w, "Key is required", http.StatusBadRequest)
			return
		}

		cacheLock.Lock()
		delete(memCache, key)
		cacheLock.Unlock()

		log.Printf("Deleted key: %s", key)

		writeJSONResponse(w, CacheResponse{
			Success: true,
			Message: "Key deleted successfully",
		})
	})

	// 添加健康检查端点
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// 启动 API 服务器
	server := &http.Server{
		Addr:    ":8080",
		Handler: loggingHandler(mux),
	}

	// 优雅关机
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		log.Println("\nShutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Fatalf("Server shutdown failed: %v", err)
		}
	}()

	// 启动服务器
	log.Println("Starting cache server at http://localhost:8080")
	log.Println("Press Ctrl+C to stop the server")
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}

// writeJSONResponse 写入 JSON 响应
func writeJSONResponse(w http.ResponseWriter, response interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}