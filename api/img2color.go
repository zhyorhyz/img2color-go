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
	"regexp"
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
	"golang.org/x/net/context"
)

var (
	redisClient      *redis.Client
	mongoClient      *mongo.Client
	cacheEnabled     bool
	useMongoDB       bool
	redisDB          int
	mongoDB          string
	ctx              = context.Background()
	colorsCollection *mongo.Collection
	allowedReferers  []string
)

func init() {
	loadConfig()
	initRedis()
	initMongoDB()
}

func loadConfig() {
	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("获取当前工作目录路径时出错：%v", err)
	}

	envFile := filepath.Join(currentDir, ".env")

	err = godotenv.Load(envFile)
	if err != nil {
		log.Fatalf("加载 .env 文件时出错：%v", err)
	}

	redisAddr := os.Getenv("REDIS_ADDRESS")
	redisPassword := os.Getenv("REDIS_PASSWORD")
	cacheEnabledStr := os.Getenv("USE_REDIS_CACHE")
	redisDBStr := os.Getenv("REDIS_DB")
	mongoDB = os.Getenv("MONGO_DB")
	mongoURI := os.Getenv("MONGO_URI")
	referers := os.Getenv("ALLOWED_REFERERS")

	redisDB, err = strconv.Atoi(redisDBStr)
	if err != nil {
		redisDB = 0
	}

	cacheEnabled = cacheEnabledStr == "true"

	useMongoDBStr := os.Getenv("USE_MONGODB")
	useMongoDB = useMongoDBStr == "true"

	allowedReferers = parseReferers(referers)
}

func initRedis() {
	redisAddr := os.Getenv("REDIS_ADDRESS")
	redisPassword := os.Getenv("REDIS_PASSWORD")
	redisDBStr := os.Getenv("REDIS_DB")

	redisDB, err := strconv.Atoi(redisDBStr)
	if err != nil {
		redisDB = 0
	}

	redisClient = redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       redisDB,
	})

	_, err = redisClient.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("连接到Redis时出错：%v", err)
	}
	log.Println("已连接到Redis！")
}

func initMongoDB() {
	mongoURI := os.Getenv("MONGO_URI")

	clientOptions := options.Client().ApplyURI(mongoURI)
	mongoClient, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatalf("连接到MongoDB时出错：%v", err)
	}
	err = mongoClient.Ping(ctx, nil)
	if err != nil {
		log.Fatalf("连接到MongoDB时出错：%v", err)
	}
	log.Println("已连接到MongoDB！")

	mongoDB = os.Getenv("MONGO_DB")
	colorsCollection = mongoClient.Database(mongoDB).Collection("colors")
}

func calculateMD5Hash(data []byte) string {
	hash := md5.Sum(data)
	return base64.StdEncoding.EncodeToString(hash[:])
}

func extractMainColor(imgURL string) (string, error) {
	md5Hash := calculateMD5Hash([]byte(imgURL))

	if cacheEnabled && redisClient != nil {
		cachedColor, err := redisClient.Get(ctx, md5Hash).Result()
		if err == nil && cachedColor != "" {
			return cachedColor, nil
		}
	}

	req, err := http.NewRequest("GET", imgURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/115.0.0.0 Safari/537.36 Edg/115.0.1901.253")

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	img, format, err := detectImageFormat(resp.Body)
	if err != nil {
		return "", err
	}

	img = resize.Resize(50, 0, img, resize.Lanczos3)

	bounds := img.Bounds()
	var r, g, b uint32
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := img.At(x, y)
			r0, g0, b0, _ := c.RGBA()
			r += r0
			g += g0
			b += b0
		}
	}

	totalPixels := uint32(bounds.Dx() * bounds.Dy())
	averageR := r / totalPixels
	averageG := g / totalPixels
	averageB := b / totalPixels

	mainColor := colorful.Color{R: float64(averageR) / 0xFFFF, G: float64(averageG) / 0xFFFF, B: float64(averageB) / 0xFFFF}
	colorHex := mainColor.Hex()

	storeColorInCacheAndDB(md5Hash, colorHex, imgURL)

	return colorHex, nil
}

func detectImageFormat(reader io.Reader) (image.Image, imaging.Format, error) {
	// 重置读取器的位置，确保可以多次读取
	if _, err := reader.Seek(0, 0); err != nil {
		return nil, imaging.FormatUnknown, err
	}

	img, format, err := image.Decode(reader)
	if err != nil {
		return nil, imaging.FormatUnknown, err
	}

	return img, format, nil
}

func storeColorInCacheAndDB(hash, color, url string) {
	if cacheEnabled && redisClient != nil {
		err := redisClient.Set(ctx, hash, color, 24*time.Hour).Err()
		if err != nil {
			log.Printf("设置Redis缓存失败: %v", err)
		}
	}

	if useMongoDB && colorsCollection != nil {
		filter := bson.M{"image_url": url}
		update := bson.M{"$set": bson.M{"main_color": color}}
		opts := options.Update().SetUpsert(true)

		_, err := colorsCollection.UpdateOne(ctx, filter, update, opts)
		if err != nil {
			log.Printf("更新MongoDB文档失败: %v", err)
		}
	}
}

func handleImageColor(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Referer")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	referer := r.Header.Get("Referer")
	if !isRefererAllowed(referer) {
		http.Error(w, "禁止访问", http.StatusForbidden)
		return
	}

	imgURL := r.URL.Query().Get("img")
	if imgURL == "" {
		http.Error(w, "缺少img参数", http.StatusBadRequest)
		return
	}

	color, err := extractMainColor(imgURL)
	if err != nil {
		http.Error(w, fmt.Sprintf("提取主色调失败：%v", err), http.StatusInternalServerError)
		return
	}

	data := map[string]string{
		"RGB": color,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func parseReferers(referers string) []string {
	refererList := strings.Split(referers, ",")
	for i, referer := range refererList {
		refererList[i] = strings.TrimSpace(referer)
	}
	return refererList
}

func isRefererAllowed(referer string) bool {
	if len(allowedReferers) == 0 {
		return false
	}

	for _, allowedReferer := range allowedReferers {
		allowedReferer = strings.ReplaceAll(allowedReferer, ".", "\\.")
		allowedReferer = strings.ReplaceAll(allowedReferer, "*", ".*")
		match, _ := regexp.MatchString(allowedReferer, referer)
		if match {
			return true
		}
	}

	return false
}

// 导出的函数，用于 Vercel Serverless Functions
func Handler(w http.ResponseWriter, r *http.Request) {
	handleImageColor(w, r)
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	http.HandleFunc("/api", Handler) // 使用导出的函数

	log.Printf("服务器监听在：%s...\n", port)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatalf("启动服务器时出错：%v\n", err)
	}
}
