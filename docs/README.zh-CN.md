<p align="center">
  <img src="../imgs/logo.png" alt="Astrolabe 星盘面板 Logo" width="140" />
</p>

<h1 align="center">Astrolabe 星盘面板</h1>

<p align="center">
  组件化的 <strong>NAS 主页</strong>、<strong>Homelab 看板</strong>、<strong>个人导航</strong> 与 <strong>监控画布</strong>——像做 PPT 一样排版</p>

<p align="center">
  组件化 · 所见即所得 · 拖拽配置 · 磁吸网格 · 宿主机 / Docker / Netdata 指标
</p>

<p align="center">
  <a href="https://github.com/ThingOfNull/astrolabe_panel/actions/workflows/ci.yml"><img alt="CI" src="https://img.shields.io/github/actions/workflow/status/ThingOfNull/astrolabe_panel/ci.yml?label=ci&logo=github"></a>
  <a href="https://github.com/ThingOfNull/astrolabe_panel/releases"><img alt="Release" src="https://img.shields.io/github/v/release/ThingOfNull/astrolabe_panel?label=release&logo=github"></a>
  <a href="https://github.com/ThingOfNull/astrolabe_panel/stargazers"><img alt="Stars" src="https://img.shields.io/github/stars/ThingOfNull/astrolabe_panel?style=flat&logo=github"></a>
</p>

<p align="center">
  <img src="../imgs/index.png" alt="Astrolabe 首页看板" width="92%" />
</p>

<p align="center"><sub><strong>首页</strong> — 画布上的书签与组件</sub></p>

<p align="center">
  <img src="../imgs/setting.png" alt="Astrolabe 设置页" width="92%" />
</p>

<p align="center"><sub><strong>设置</strong> — 编排组件与看板偏好</sub></p>

**Astrolabe 星盘面板**是一款**高度自定义**的主页与轻量看板：磁吸网格画布上拖拽小组件，探活与书签并列，指标可来自**宿主机**、**Docker** 或 **Netdata REST**，把导航和日常监控收拢到同一套轻量界面里。

[**English documentation**](../README.md)

## 星盘面板和其他home panel的区别？

传统导航页擅长静态链接，但 Homelab 用户往往还要在书签、状态页和各类指标界面之间来回跳；传统home panel布局固定，但是我们更想要所见即所得。星盘把首页做成**可编排、可数据驱动**的画布：实时刷新，书签带探活，图表与仪表盘与链接同屏；改布局更像改看板，而不是在配置文件里写代码。

项目坚持**轻量交付**：前端打进二进制，发布一个 **`astrolabe`** 即可在 NAS 或小虚拟机上长期运行；颜色、图标、壁纸均可调，无需为换肤维护一套独立栈。

## 核心功能

- **组件化布局：** 所见即所得、拖拽配置；磁吸网格对齐，排版省心。
- **轻量化运行：** `make build` 后**单文件**交付；常态内存约 **30MB 量级**（随组件与采集频率变化）。
- **高度可定制：** 深/浅主题与 CSS 变量；Iconify 图标；可选壁纸与玻璃拟态等样式。
- **实时指标：** 宿主机（CPU、内存、负载、磁盘、网络）、**Docker**、**Netdata**；折线/柱状图、仪表盘、状态矩阵等。
- **不止导航：** 全局聚合搜索（**Ctrl+K**）、时钟、天气(因api限制，目前仅中国大陆地区)、文本、分割线；支持带探活的书签类组件。
- **配置可迁移：** 支持导出/导入看板、数据源与组件。

## 🌟 功能概览

### 🧩 看板画布

> **像做幻灯片一样摆组件——所见即所得。**

- ✨ **磁吸网格：** 少纠结坐标，对齐更整齐。
- 🔄 **实时同步：** WebSocket 推送，编辑与数据刷新跟手。
- 📦 **快照：** JSON 导入导出，便于备份与迁移。

### 📊 指标与数据源

> **宿主机、容器与 Netdata 接到同一块画布。**

- 🖥️ **本地与 Docker：** 不额外搭服务也能看核心资源信号。
- 📈 **Netdata REST：** 可直接接入已有netdata，获取丰富指标。


### 🎨 外观与资源

> **主题与素材一体化，不必另开设计工具链。**

- 🌙 **颜色可自定义：** 所有元素颜色均可自定义。
- 🖼️ **壁纸与图标：** 可自主上传图标、壁纸，也可使用 Iconify 图标。

## 快速开始

### Docker

```bash
docker compose up -d --build
# 浏览器访问 http://localhost:8080
```

容器内数据持久化目录一般为 `/data/.astrolabe_panel/`（由 compose 卷挂载）。

### 本地构建

环境要求：**Go 1.25+**、**Node 22**、**pnpm 10+**、**GNU make**。

```bash
make build     # vite 构建 + go 构建，生成 ./astrolabe
./astrolabe    # 默认监听 8080
```

### 前后端分离开发

```bash
make dev-back  # 后端 API + WebSocket，:8080
make dev-web   # Vite :5173，将 WebSocket 代理到后端
```

## 配置说明

配置文件查找顺序：

1. `--config /path/to/config.json`
2. 环境变量 `ASTROLABE_CONFIG`
3. 默认 `~/.astrolabe_panel/config.json`（Windows：`%USERPROFILE%\.astrolabe_panel\config.json`）

若文件不存在，首次启动会写入默认配置。

## 常用命令

```bash
make test          # go test ./...
make lint          # go vet + 前端 ESLint
make smoke         # WebSocket JSON-RPC 烟测客户端
cd web && pnpm e2e # Playwright（需先安装浏览器）
```

## 技术栈

**后端：** Go、Gin、GORM（SQLite）、Docker 客户端、gopsutil、slog。  
**前端：** Vue 3、TypeScript、Pinia、vue-i18n、ECharts、Tailwind CSS。

## 🙏 致谢

感谢以下开源项目（及全体贡献者）：

- [Vue.js](https://github.com/vuejs/core) — 前端框架  
- [Vite](https://github.com/vitejs/vite) — 构建与开发服务  
- [Gin](https://github.com/gin-gonic/gin) — Go Web 框架  
- [GORM](https://github.com/go-gorm/gorm) — SQLite 持久化  
- [ECharts](https://github.com/apache/echarts) — 图表可视化  
- [Tailwind CSS](https://github.com/tailwindlabs/tailwindcss) — 样式工具集  

## 许可证

MIT
