package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/temoto/robotstxt"
)

type Config struct {
	Url      string `json:"url"`
	Selector string `json:"selector"`
}

type Result struct {
	Title string `json:"title"`
	Url   string `json:"url"`
}

func main() {
	// 環境変数 "TARGET_PATH" からパスを取得
	targetPath := os.Getenv("TARGET_PATH")
	if targetPath == "" {
		log.Println("環境変数 TARGET_PATH が設定されていません")
		return
	}
	dirInfo, err := os.Stat(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Println("ディレクトリが存在しません: %w", err, targetPath)
		}
		log.Println("ディレクトリの情報取得に失敗しました: %w", err)
		return
	}
	if !dirInfo.IsDir() {
		log.Println("指定されたパスはディレクトリではありません")
		return
	}
	if !strings.HasSuffix(targetPath, "/") {
		targetPath = targetPath + "/"
	}

	settingFile := targetPath + "setting.json"
	info, err := os.Stat(settingFile)
	if err != nil {
		if os.IsNotExist(err) {
			log.Println("設定ファイルが存在しません: %w", err, settingFile)
		}
		log.Println("設定ファイルの情報取得に失敗しました: %w", err)
		return
	}
	if info.IsDir() {
		log.Println("設定ファイルと同名のディレクトリが存在します")
		return
	}
	file, err := os.Open(settingFile)
	if err != nil {
		log.Println("ファイルを開けませんでした: %w", err)
	}
	defer file.Close()

	// ファイル内容をデコード
	var config Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		log.Println("JSONデコードに失敗しました: %w", err)
	}
	if config.Url == "" {
		log.Println("URLが設定されていません path:", settingFile)
		return
	}
	if config.Selector == "" {
		log.Println("セレクタが設定されていません path:", settingFile)
		return
	}

	err = config.isCrawlerAllowed()
	if err != nil {
		log.Println("クロール許可の確認に失敗しました: ", err)
		return
	}

	res, err := http.Get(config.Url)
	if err != nil {
		log.Println("HTTPリクエストに失敗しました:", err, config.Url)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		log.Println("HTTPステータスコードエラー:", res.StatusCode, res.Status, config.Url)
		return
	}

	// HTMLをパース
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatalf("HTMLのパースに失敗しました: %v", err)
	}

	// querySelectorのように特定の要素を取得
	// 例: <a>タグ内のリンクを取得
	results := []Result{}
	doc.Find(config.Selector).Each(func(index int, item *goquery.Selection) {
		href, exists := item.Attr("href")
		if exists {
			url, err := config.makeUrl(href)
			if err != nil {
				log.Println(err)
			} else {
				results = append(results, Result{Title: formatTitle(item.Text()), Url: url})
			}
		}
	})
	outputFile, err := os.Create(targetPath + "result.json")
	if err != nil {
		log.Println("ファイルの作成に失敗しました: %w", err)
		return
	}
	defer file.Close()

	// JSONエンコーダーを使用してファイルに書き込む
	encoder := json.NewEncoder(outputFile)
	encoder.SetIndent("", "  ") // 読みやすい形式で書き出す
	if err := encoder.Encode(results); err != nil {
		log.Println("JSONのエンコードに失敗しました: %w", err)
	}
}

func (config Config) makeUrl(href string) (string, error) {
	if strings.HasPrefix(href, "http") {
		return href, nil
	}
	url, err := url.Parse(config.Url)
	if err != nil {
		return href, fmt.Errorf("URLのパースに失敗しました feed: %s, paht: %s, %w", config.Url, href, err)
	}
	if strings.HasPrefix(href, "/") {
		return url.Scheme + "://" + url.Host + href, nil
	}
	return url.Scheme + "://" + url.Host + "/" + href, nil
}

func (config Config) isCrawlerAllowed() error {
	url, err := url.Parse(config.Url)
	if err != nil {
		return fmt.Errorf("URLのパースに失敗しました: %w", err)
	}
	robotsUrl := url.Scheme + "://" + url.Host + "/robots.txt"
	res, err := http.Get(robotsUrl)
	if err != nil {
		return fmt.Errorf("robots.txt確認のHTTPリクエストに失敗しました: %w, %s", err, robotsUrl)
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusForbidden || res.StatusCode == http.StatusInternalServerError {
		return fmt.Errorf("クロール許可されていません: %d, %s, %s", res.StatusCode, res.Status, robotsUrl)
	}
	if res.StatusCode != http.StatusOK {
		log.Println("HTTPステータスコードエラー: ", res.StatusCode, res.Status, robotsUrl)
		return nil
	}
	// Parse robots.txt
	robotsData, err := robotstxt.FromResponse(res)
	if err != nil {
		log.Println("failed to parse robots.txt:", err)
		return nil
	}

	// Check crawlability for the path
	allowed := robotsData.TestAgent(url.Path, "bot")
	if !allowed {
		return fmt.Errorf("クロール許可されていません: %s", config.Url)
	}
	return nil
}

func removeExtraSpaces(input string) string {
	// 正規表現で連続する空文字を1つにまとめる
	re := regexp.MustCompile(`\s+`)
	return re.ReplaceAllString(input, " ")
}

func formatTitle(s string) string {
	tabTrimed := strings.ReplaceAll(s, "\t", "")
	extraSpacesTrimed := removeExtraSpaces(tabTrimed)
	return extraSpacesTrimed
}
