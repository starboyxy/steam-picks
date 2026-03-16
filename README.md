# Steam 每日精选 🎮

每天自动从 Steam 获取精选热门游戏，生成静态网站展示。

## 预览

访问: https://starboyxy.github.io/steam-picks

## 特性

- 🔄 每天北京时间 10:00 自动更新
- 🎯 精选 10 款热门/特惠/好评游戏
- 🌙 Steam 暗色风格界面
- 📱 响应式设计，手机也能看
- 🔗 点击直达 Steam 商店页面
- 💰 显示价格和折扣信息
- 🏷️ 显示游戏类型标签

## 技术栈

- **后端**: Go (数据抓取 + 静态页面生成)
- **前端**: 纯 HTML/CSS (无框架依赖)
- **托管**: GitHub Pages (免费)
- **自动化**: GitHub Actions (每日定时任务)

## 本地运行

```bash
go run main.go
# 生成的网站在 docs/ 目录下
```

## 手动更新

在 GitHub 仓库的 Actions 页面，选择 "Daily Steam Picks Update"，点击 "Run workflow"。

## License

MIT
