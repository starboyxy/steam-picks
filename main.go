package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Game represents a Steam game
type Game struct {
	AppID            int      `json:"appid"`
	Name             string   `json:"name"`
	HeaderImage      string   `json:"header_image"`
	ShortDescription string   `json:"short_description"`
	IsFree           bool     `json:"is_free"`
	Price            string   `json:"price"`
	OriginalPrice    string   `json:"original_price"`
	Discount         int      `json:"discount_percent"`
	Genres           []string `json:"genres"`
	ReleaseDate      string   `json:"release_date"`
	ReviewScore      string   `json:"review_score"`
	StoreURL         string   `json:"store_url"`
}

// FeaturedResponse from Steam API
type FeaturedResponse struct {
	FeaturedWin []FeaturedItem `json:"featured_win"`
	FeaturedMac []FeaturedItem `json:"featured_mac"`
	LargeCapsules []FeaturedItem `json:"large_capsules"`
	FeaturedLinux []FeaturedItem `json:"featured_linux"`
}

type FeaturedItem struct {
	ID               int    `json:"id"`
	Name             string `json:"name"`
	DiscountedPrice  int    `json:"discounted_price"`
	OriginalPrice    int    `json:"original_price"`
	DiscountPercent  int    `json:"discount_percent"`
	HeaderImage      string `json:"header_image"`
	LargeCapsuleImage string `json:"large_capsule_image"`
	Currency         string `json:"currency"`
	IsFree           bool   `json:"is_free"`
}

// AppDetailsResponse from Steam store API
type AppDetailsResponse map[string]struct {
	Success bool `json:"success"`
	Data    struct {
		Name             string `json:"name"`
		SteamAppID       int    `json:"steam_appid"`
		IsFree           bool   `json:"is_free"`
		ShortDescription string `json:"short_description"`
		HeaderImage      string `json:"header_image"`
		Genres           []struct {
			Description string `json:"description"`
		} `json:"genres"`
		ReleaseDate struct {
			Date string `json:"date"`
		} `json:"release_date"`
		PriceOverview *struct {
			Currency        string `json:"currency"`
			Initial         int    `json:"initial"`
			Final           int    `json:"final"`
			DiscountPercent int    `json:"discount_percent"`
			InitialFormatted string `json:"initial_formatted"`
			FinalFormatted   string `json:"final_formatted"`
		} `json:"price_overview"`
	} `json:"data"`
}

// TopSellersResponse from Steam Charts
type TopSellersResponse struct {
	Response struct {
		Ranks []struct {
			Rank  int `json:"rank"`
			AppID int `json:"appid"`
		} `json:"ranks"`
	} `json:"response"`
}

func fetchJSON(url string, target interface{}) error {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}
	client := &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return json.Unmarshal(body, target)
}

func getFeaturedGames() ([]int, error) {
	url := "https://store.steampowered.com/api/featured/?cc=cn&l=schinese"
	var resp FeaturedResponse
	if err := fetchJSON(url, &resp); err != nil {
		return nil, fmt.Errorf("fetch featured: %w", err)
	}

	seen := make(map[int]bool)
	var appIDs []int

	// Collect from all featured lists
	allItems := append(resp.LargeCapsules, resp.FeaturedWin...)
	allItems = append(allItems, resp.FeaturedMac...)

	for _, item := range allItems {
		if item.ID > 0 && !seen[item.ID] {
			seen[item.ID] = true
			appIDs = append(appIDs, item.ID)
		}
	}

	return appIDs, nil
}

func getAppDetails(appID int) (*Game, error) {
	url := fmt.Sprintf("https://store.steampowered.com/api/appdetails?appids=%d&cc=cn&l=schinese", appID)
	var resp AppDetailsResponse
	if err := fetchJSON(url, &resp); err != nil {
		return nil, err
	}

	key := fmt.Sprintf("%d", appID)
	data, ok := resp[key]
	if !ok || !data.Success {
		return nil, fmt.Errorf("app %d not found", appID)
	}

	game := &Game{
		AppID:            data.Data.SteamAppID,
		Name:             data.Data.Name,
		HeaderImage:      data.Data.HeaderImage,
		ShortDescription: data.Data.ShortDescription,
		IsFree:           data.Data.IsFree,
		ReleaseDate:      data.Data.ReleaseDate.Date,
		StoreURL:         fmt.Sprintf("https://store.steampowered.com/app/%d", appID),
	}

	var genres []string
	for _, g := range data.Data.Genres {
		genres = append(genres, g.Description)
	}
	game.Genres = genres

	if data.Data.PriceOverview != nil {
		po := data.Data.PriceOverview
		game.Price = po.FinalFormatted
		game.OriginalPrice = po.InitialFormatted
		game.Discount = po.DiscountPercent
	} else if data.Data.IsFree {
		game.Price = "免费"
	}

	return game, nil
}

func main() {
	log.Println("🎮 Steam Picks - 开始获取今日精选游戏...")

	// Get featured game IDs
	appIDs, err := getFeaturedGames()
	if err != nil {
		log.Fatalf("获取精选列表失败: %v", err)
	}
	log.Printf("获取到 %d 个精选游戏ID", len(appIDs))

	// Limit to 10
	if len(appIDs) > 10 {
		appIDs = appIDs[:10]
	}

	// Fetch details for each game
	var games []Game
	for i, id := range appIDs {
		log.Printf("获取游戏详情 [%d/%d]: AppID %d", i+1, len(appIDs), id)
		game, err := getAppDetails(id)
		if err != nil {
			log.Printf("跳过 AppID %d: %v", id, err)
			continue
		}
		games = append(games, *game)
		// Rate limit: Steam API needs ~1s between requests
		time.Sleep(800 * time.Millisecond)
	}

	log.Printf("成功获取 %d 款游戏详情", len(games))

	// Save JSON data
	outputDir := "docs"
	os.MkdirAll(outputDir, 0755)

	jsonData, _ := json.MarshalIndent(games, "", "  ")
	jsonPath := filepath.Join(outputDir, "data.json")
	os.WriteFile(jsonPath, jsonData, 0644)

	// Generate HTML
	generateHTML(games, outputDir)
	log.Println("✅ 网站生成完成！输出目录: docs/")
}

func generateHTML(games []Game, outputDir string) {
	now := time.Now().In(time.FixedZone("CST", 8*3600))
	dateStr := now.Format("2006年01月02日")

	tmpl := template.Must(template.New("index").Funcs(template.FuncMap{
		"join": func(s []string) string { return strings.Join(s, " / ") },
	}).Parse(htmlTemplate))

	f, err := os.Create(filepath.Join(outputDir, "index.html"))
	if err != nil {
		log.Fatalf("创建HTML文件失败: %v", err)
	}
	defer f.Close()

	data := struct {
		Date  string
		Games []Game
		Year  int
	}{
		Date:  dateStr,
		Games: games,
		Year:  now.Year(),
	}

	if err := tmpl.Execute(f, data); err != nil {
		log.Fatalf("渲染模板失败: %v", err)
	}
}

const htmlTemplate = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Steam 每日精选 - {{.Date}}</title>
    <meta name="description" content="每日精选Steam热门游戏推荐，发现你的下一款游戏">
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Arial, sans-serif;
            background: #1b2838;
            color: #c7d5e0;
            min-height: 100vh;
        }

        .header {
            background: linear-gradient(135deg, #1b2838 0%, #2a475e 100%);
            padding: 40px 20px;
            text-align: center;
            border-bottom: 2px solid #66c0f4;
        }

        .header h1 {
            font-size: 2.5em;
            color: #fff;
            margin-bottom: 8px;
            text-shadow: 0 2px 10px rgba(102, 192, 244, 0.3);
        }

        .header h1 span { color: #66c0f4; }

        .header .date {
            font-size: 1.1em;
            color: #8f98a0;
        }

        .header .subtitle {
            font-size: 0.95em;
            color: #56707f;
            margin-top: 4px;
        }

        .container {
            max-width: 1200px;
            margin: 0 auto;
            padding: 30px 20px;
        }

        .game-grid {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(340px, 1fr));
            gap: 24px;
        }

        .game-card {
            background: #16202d;
            border-radius: 8px;
            overflow: hidden;
            transition: transform 0.2s, box-shadow 0.2s;
            border: 1px solid #2a475e;
        }

        .game-card:hover {
            transform: translateY(-4px);
            box-shadow: 0 8px 25px rgba(0,0,0,0.4);
            border-color: #66c0f4;
        }

        .game-card a {
            text-decoration: none;
            color: inherit;
            display: block;
        }

        .game-image {
            width: 100%;
            aspect-ratio: 460/215;
            object-fit: cover;
            display: block;
        }

        .game-info {
            padding: 16px;
        }

        .game-title {
            font-size: 1.2em;
            color: #fff;
            margin-bottom: 8px;
            line-height: 1.3;
        }

        .game-desc {
            font-size: 0.85em;
            color: #8f98a0;
            line-height: 1.5;
            margin-bottom: 12px;
            display: -webkit-box;
            -webkit-line-clamp: 3;
            -webkit-box-orient: vertical;
            overflow: hidden;
        }

        .game-meta {
            display: flex;
            justify-content: space-between;
            align-items: center;
            flex-wrap: wrap;
            gap: 8px;
        }

        .game-genres {
            font-size: 0.8em;
            color: #56707f;
        }

        .game-price {
            display: flex;
            align-items: center;
            gap: 8px;
        }

        .discount-badge {
            background: #4c6b22;
            color: #a4d007;
            padding: 2px 6px;
            border-radius: 3px;
            font-weight: bold;
            font-size: 0.85em;
        }

        .price-original {
            text-decoration: line-through;
            color: #56707f;
            font-size: 0.85em;
        }

        .price-final {
            color: #acdbf5;
            font-weight: bold;
            font-size: 1em;
        }

        .price-free {
            color: #a4d007;
            font-weight: bold;
        }

        .release-date {
            font-size: 0.78em;
            color: #56707f;
            margin-top: 8px;
        }

        .footer {
            text-align: center;
            padding: 30px;
            color: #56707f;
            font-size: 0.85em;
            border-top: 1px solid #2a475e;
            margin-top: 40px;
        }

        .footer a { color: #66c0f4; text-decoration: none; }
        .footer a:hover { text-decoration: underline; }

        @media (max-width: 600px) {
            .header h1 { font-size: 1.8em; }
            .game-grid { grid-template-columns: 1fr; }
            .container { padding: 16px; }
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>🎮 Steam <span>每日精选</span></h1>
        <div class="date">{{.Date}}</div>
        <div class="subtitle">每天为你精选热门好游戏</div>
    </div>

    <div class="container">
        <div class="game-grid">
            {{range .Games}}
            <div class="game-card">
                <a href="{{.StoreURL}}" target="_blank" rel="noopener">
                    <img class="game-image" src="{{.HeaderImage}}" alt="{{.Name}}" loading="lazy">
                    <div class="game-info">
                        <h2 class="game-title">{{.Name}}</h2>
                        <p class="game-desc">{{.ShortDescription}}</p>
                        <div class="game-meta">
                            <span class="game-genres">{{join .Genres}}</span>
                            <div class="game-price">
                                {{if gt .Discount 0}}
                                    <span class="discount-badge">-{{.Discount}}%</span>
                                    <span class="price-original">{{.OriginalPrice}}</span>
                                {{end}}
                                {{if .IsFree}}
                                    <span class="price-free">免费游玩</span>
                                {{else}}
                                    <span class="price-final">{{.Price}}</span>
                                {{end}}
                            </div>
                        </div>
                        {{if .ReleaseDate}}
                        <div class="release-date">📅 {{.ReleaseDate}}</div>
                        {{end}}
                    </div>
                </a>
            </div>
            {{end}}
        </div>
    </div>

    <div class="footer">
        <p>数据来源: <a href="https://store.steampowered.com" target="_blank">Steam</a> | 每日自动更新</p>
        <p style="margin-top:6px;">© {{.Year}} Steam 每日精选 | Powered by GitHub Pages</p>
    </div>
</body>
</html>`
