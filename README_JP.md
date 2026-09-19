# ai-dev-sop

**AI 開発 SOP とスキル**

AI 支援開発のための包括的な標準作業手順（SOP）リポジトリ。開発ワークフロー、スキルツール、自動化テスト、mem0 長期メモリシステムを備えています。

## 概要

このリポジトリは **business-platform**（業務プラットフォーム）の完全な開発 SOP であり、バックエンド開発者、テストエンジニア、プロジェクトマネージャー向けです。「一言の必要条件」から「コードデリバリー」までの完全な RD チェーンの標準操作ガイドを提供します。

## 主な機能

- **7段階パイプライン**：インストール → 理解 → プロンプト生成 → シナリオ選択 → テスト自動化 → ドキュメント自動化 → 運用要件自動化
- **スキルクラスター**：superpower、gstack、get-shit-done、OpenSpec などの AI 開発スキル
- **mem0 長期メモリ**：セルフホスト mem0（ベクトル + グラフメモリ）、MCP による全 AI IDE 統合
- **シナリオベースパイプライン**：スキャフォールド、一言必要条件、アップグレード、複製、Web/App クローン
- **QA 自動化**：`/qa-dev` スキルによるエンドツーエンドブラウザ自動化テスト
- **マルチ IDE サポート**：Cursor、Qoder、CodeBuddy、Claude Code

## アーキテクチャ図

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        AI 開発パイプライン                              │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  インストール  →  理解  →  プロンプト生成  →  シナリオ選択              │
│        ↓                                                       ↓         │
│        └─────────  テスト自動化  ←──  ドキュメント自動化  ←──────────┘   │
│        ↓                                                               │
│        └───────────────  運用要件自動化（FDE フィードバックループ）        │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

## 主要コンポーネント

### 1. 開発 SOP（`docs/cn/`）

コアドキュメント `develop-sop.md` の内容：

| 章 | 内容 |
|------|------|
| §0 | ドキュメント概要 - 目標、役割、フローシート、アーキテクチャ |
| §1 | 準備フェーズ - プロジェクトの読解、スキルのインストール、IDE 設定 |
| §2 | プロンプト生成 - テストプロンプト、開発プロンプト、シナリオテンプレート |
| §3 | テストと開発 - エンドツーエンド実行、バッチ処理 |
| §4 | シナリオ SOP - スキャフォールド、一言必要条件、アップグレード、複製パイプライン |
| §5 | 長期メモリ - mem0 メモリシステム（保存/検索/チーム共有） |
| §6 | ドキュメント作成 - 製品研究、市場調査テンプレート |

### 2. スキル（`skills/`）

- **qa-dev**: テスト+開発自動化スキル、`/qa-dev` コマンドをサポート
- バッチ実行、無人モード、回帰テストをサポート

### 3. メモリサービス（`deploy/mem0/`）

- **mem0**: セルフホスト長期メモリサービス（API :8888 / MCP :8080 / Dashboard :3001）、
  PostgreSQL(pgvector) + Neo4j 基盤、CodeBuddy / Qoder / Cursor 等へ MCP 統合

### 4. サンプルプロジェクト（`example/`）

- **ai-coding-boot**: ruoyi-vue-pro アーキテクチャに基づく参照実装

## 7段階の開発フェーズ

| フェーズ | 説明 | 成果物 |
|------|------|--------|
| 1. インストール | スキル、ツールチェーンのインストール、IDE 設定 | スキルセット、設計規範 |
| 2. 理解 | `/gsd-map-codebase` によるコードベースマッピング | 7つの認知ファイル |
| 3. プロンプト生成 | テンプレートからのテスト/開発プロンプト生成 | シーンドキュメント、PRD |
| 4. シナリオ選択 | 対応するシナリオパイプラインへのルーティング | コード、コミット、レポート |
| 5. テスト自動化 | エンドツーエンドブラウザ自動化テスト | テストレポート、バグリスト |
| 6. ドキュメント自動化 | プロジェクトドキュメントの自動生成 | API ドキュメント、運用マニュアル |
| 7. 運用自動化 | 顧客要件 → FDE 実行 | フィードバック記録 |

## シナリオパイプライン

| パイプライン | ユースケース |
|----------|----------|
| `framework-pipeline.md` | ベースラインスキャフォールド開発 |
| `one-sentence-pipeline.md` | 一言必要条件からプロトタイプまで |
| `docs-pipeline.md` | プロジェクトドキュメント自動化 |
| `copy-web-pipeline.md` | 既存の Web プロジェクトをクローン |
| `copy-app-pipeline.md` | モバイルアプリのクローン（uniapp） |
| `java-upgrade-pipeline.md` | テクノロジース택 마이グレーション |

## mem0 長期メモリシステム

MCP プロトコルですべての AI 開発ツールに長期メモリを提供します：

| 機能 | 説明 |
|------|------|
| 長期メモリ | 会話/事実/意思決定/設定、ベクトル検索（pgvector） |
| グラフメモリ | エンティティ関係グラフ（Neo4j）、`GRAPH_ENABLED=true` |
| チーム共有 | `git_remote` → `project_id` 共有プール、クロスユーザー検索 |
| 会話ログ | 毎ターンの Q/A 原文アーカイブ、`session_id` でグループ化 |

### デプロイ

```bash
# ワンクリックインストール（mem0 サービス）
./scripts/install-all.sh          # Linux/macOS
./scripts/install-all.ps1         # Windows

# または手動
cd deploy/mem0 && docker compose up -d
```

### IDE 統合

```json
{
  "mcpServers": {
    "mem0-local": {
      "type": "http",
      "url": "http://127.0.0.1:8080/mcp"
    }
  }
}
```

詳細は [docs/quick-ref/mem0-manual.md](docs/quick-ref/mem0-manual.md) を参照。

## クイックスタート

### 1. スキルのインストール

```bash
# npx skills を使用
npx skills add obra/superpowers -a qoder --global
npx skills add garrytan/gstack -a qoder --global
npx get-shit-done-cc@latest --qoder --global
npm install -g @fission-ai/openspec@latest

# design-md をクローン
git clone https://github.com/VoltAgent/awesome-design-md.git temp-design
cp temp-design/design-md/vercel/DESIGN.md ./DESIGN.md
rm -rf temp-design
```

### 2. IDE の設定

**Qoder:**
```json
{
  "general": {
    "defaultPermissionMode": "auto"
  }
}
```

**Cursor:** `mem0` MCP サーバーをインストールします（[docs/ide-config/ide-mcp-templates.md](docs/ide-config/ide-mcp-templates.md) 参照）。

### 3. コードベースのマップ

```bash
/gsd-map-codebase
```

`.planning/codecase/` に 7つの認知ファイルを生成：
- ARCHITECTURE.md
- CONCERNS.md
- CONVENTIONS.md
- INTEGRATIONS.md
- STACK.md
- STRUCTURE.md
- TESTING.md

### 4. シーんプロンプトの生成

```bash
# 既存コードからテスト+開発プロンプトを生成
/brainstorm
@business-web/docs/scene/scene-template.md のテンプレートに従って...]
```

### 5. 開発の実行

```bash
# 完全フロー：テスト → レポート → 修正 → 開発 → 検証
/qa-dev --file docs/scene/system-scene.md --module SYS-01

# バッチ実行（無人モード）
/qa-dev --batch --unattended --file docs/scene/system-scene.md
```

## 推奨 AI モデル

| IDE | 推奨モデル |
|-----|----------|
| Cursor | composer-2.5 |
| Qoder | Lite / QWen3.7-MAX |
| CodeBuddy | Hy3 |

## プロジェクト構造

```
ai-dev-sop/
├── README.md           # 英語版
├── README_CN.md        # 中国語版
├── README_JP.md         # 日本語版
├── docs/
│   ├── cn/               # 中国語ドキュメント
│   │   ├── develop-sop.md
│   │   ├── scene-template.md
│   │   ├── one-sentence-pipeline.md
│   │   ├── framework-pipeline.md
│   │   ├── copy-web-pipeline.md
│   │   ├── copy-app-pipeline.md
│   │   ├── java-upgrade-pipeline.md
│   │   ├── docs-pipeline.md
│   │   ├── qa-dev-sop.md
│   │   └── benchmark/       # 評価データセット（履歴、旧メモリシステムと共にアーカイブ）
│   │       ├── DS-Decision.md
│   │       ├── DS-CallPath.md
│   │       ├── DS-Cross.md
│   │       ├── DS-ADR.md
│   │       ├── DS-Customer.md
│   │       └── DS-DeadCode.md
│   └── scene/
├── skills/
│   ├── qa-dev/
│   └── mem0-longterm-memory/   # mem0 メモリスキル
└── deploy/
    └── mem0/               # セルフホスト mem0（API/MCP/Dashboard）
```

## リソースリンク

- [mem0 ドキュメント](https://docs.mem0.ai/)
- [mem0 MCP 統合ガイド](https://docs.mem0.ai/platform/mem0-mcp)
- [Mem0 Cloud Dashboard](https://app.mem0.ai/)

## ライセンス

MIT License

## コントリビューション

新しいシナリオ、パイプライン、スキルのコントリビューションを歓迎します。コントリビューション時は SOP ガイドラインに従ってください。
