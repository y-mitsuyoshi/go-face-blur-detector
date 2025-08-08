package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	// 環境変数から設定を取得
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	env := os.Getenv("ENV")
	if env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Ginルーターを作成
	r := gin.Default()

	// ヘルスチェック用エンドポイント
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "go-face-blur-detector",
		})
	})

	// 基本的なルート
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Face Blur Detector API",
			"version": "v1.0.0",
		})
	})

	// TODO: 顔検出エンドポイントを追加
	r.POST("/detect", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Face detection endpoint - Coming soon",
		})
	})

	log.Printf("サーバーをポート %s で起動中...", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("サーバーの起動に失敗しました:", err)
	}
}
