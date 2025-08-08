package main

import (
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/y-mitsuyoshi/go-face-blur-detector/internal/facedetector"
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

	// 顔検出エンドポイント
	r.POST("/detect", func(c *gin.Context) {
		// multipart/form-dataから画像ファイルを取得
		file, _, err := c.Request.FormFile("image")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "画像ファイルの取得に失敗しました: " + err.Error()})
			return
		}
		defer file.Close()

		// ファイルの内容を読み込む
		imgData, err := io.ReadAll(file)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "画像の読み込みに失敗しました: " + err.Error()})
			return
		}

		// 鮮明度を計算
		sharpness, err := facedetector.CalculateSharpness(imgData)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "鮮明度の計算に失敗しました: " + err.Error()})
			return
		}

		// 結果を返す
		c.JSON(http.StatusOK, gin.H{
			"sharpness_score": sharpness,
		})
	})

	log.Printf("サーバーをポート %s で起動中...", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("サーバーの起動に失敗しました:", err)
	}
}
