package handler

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/joho/godotenv"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/nfnt/resize"
	"github.com/disintegration/imaging"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	redisClient      *redis.Client
	mongoClient      *mongo.Client
	cacheEnabled     bool
	useMongoDB       bool
	redisDB          int
	mongoDB          string
	colorsCollection *mongo.Collection
	allowedReferers  []string
)

func init() {
	loadConfig()
	initRedis()
	initMongoDB()
}

func loadConfig() {
	// 加载配置文件代码保持不变
}

func initRedis() {
	// 初始化 Redis 代码保持不变
}

func initMongoDB() {
	// 初始化 MongoDB 代码保持不变
}

func calculateMD5Hash(data []byte) string {
	// calculateMD5Hash 函数保持不变
}

func extractMainColor(imgURL string) (string, error) {
	// extractMainColor 函数保持不变
}

func detectImageFormat(reader io.Reader) (image.Image, error) {
	// 重置读取器的位置，确保可以多次读取
	type resetter interface {
		Seek(offset int64, whence int) (int64, error)
	}

	if reset, ok := reader.(resetter); ok {
		reset.Seek(0, io.SeekStart)
	} else {
		log.Println("Reader does not support Seek")
	}

	img, format, err := imaging.Decode(reader)
	if err != nil {
		return nil, err
	}

	return img, format, nil
}

func storeColorInCacheAndDB(hash, color, url string) {
	// storeColorInCacheAndDB 函数保持不变
}

func handleImageColor(w http.ResponseWriter, r *http.Request) {
	// handleImageColor 函数保持不变
}

func parseReferers(referers string) []string {
	// parseReferers 函数保持不变
}

func isRefererAllowed(referer string) bool {
	// isRefererAllowed 函数保持不变
}

// 导出的函数，用于 Vercel Serverless Functions
func Handler(w http.ResponseWriter, r *http.Request) {
	handleImageColor(w, r)
}

func main() {
	// main 函数保持不变
}
