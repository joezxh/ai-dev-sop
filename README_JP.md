# ai-dev-sop

**AI 開発 SOP とスキル**

AI 支援開発のための包括的な標準作業手順（SOP）リポジトリ。開発ワークフロー、スキルツール、自動化テスト、ダブルトラックメモリフレームワークを備えています。

## 概要

このリポジトリは **mediation-platform**（調停プラットフォーム）の完全な開発 SOP であり、バックエンド開発者、テストエンジニア、プロジェクトマネージャー向けです。「一言の必要条件」から「コードデリバリー」までの完全な RD チェーンの標準操作ガイドを提供します。

## 主な機能

- **7段階パイプライン**：インストール → 理解 → プロンプト生成 → シナリオ選択 → テスト自動化 → ドキュメント自動化 → 運用要件自動化
- **スキルクラスター**：superpower、gstack、get-shit-done、OpenSpec などの AI 開発スキル
- **ダブルトラックメモリフレームワーク**：MemPalace（セマンティックメモリ）× codebase-mem-mcp（コード構造メモリ）
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
| §5 | ダブルトラックメモリ - MemPalace × codebase-mem-mcp フレームワーク |
| §6 | ドキュメント作成 - 製品研究、市場調査テンプレート |

### 2. スキル（`skills/`）

- **qa-dev**: テスト+開発自動化スキル、`/qa-dev` コマンドをサポート
- バッチ実行、無人モード、回帰テストをサポート

### 3. ツール（`tools/`）

- **cbmem-team**: codebase-memory-mcp の HTTP マルチユーザー ラッパー、チームコラボレーション用

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

## ダブルトラックメモリフレームワーク

自然言語メモリとコード構造メモリを組み合わせます：

| トラック | テクノロジー | 目的 |
|------|------|------|
| **トラック A** | MemPalace | チームセマンティックメモリ、意思決定、顧客要件 |
| **トラック B** | codebase-mem-mcp | コード構造、アーキテクチャグラフ、コールチェーン |

### デプロイ

```bash
# トラック A：MemPalace（Docker）
docker run -d --name mempalace \
  -p 8080:8080 \
  -v ~/.mempalace:/data \
  -e MP_VECTOR_BACKEND=chromadb \
  mempalace/mempalace:0.8.3

# トラック B：codebase-memory-mcp
npm install -g codebase-memory-mcp
codebase-memory-mcp install
```

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

**Cursor:** `playwright` と `memplace` MCP サーバーをインストールします。

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
@mediation-web/docs/scene/scene-template.md のテンプレートに従って...]
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
│   │   ├── mempalace-codebase-mem-framework.md
│   │   └── benchmark/       # 評価データセット
│   │       ├── DS-Decision.md
│   │       ├── DS-CallPath.md
│   │       ├── DS-Cross.md
│   │       ├── DS-ADR.md
│   │       ├── DS-Customer.md
│   │       └── DS-DeadCode.md
│   └── scene/
├── skills/
│   └── qa-dev/
├── tools/
│   └── cbmem-team/       # codebase-memory-mcp チームラッパー
│       ├── cmd/
│       │   ├── cbmem-team/
│       │   └── cbmem-mint-token/
│       └── internal/
│           ├── pool/
│           ├── mcp/
│           ├── auth/
│           └── store/
└── example/
    └── ai-coding-boot/
```

## リソースリンク

- [MemPalace GitHub](https://github.com/MemPalace/mempalace)
- [MemPalace ドキュメント](https://mempalaceofficial.com/)
- [codebase-memory-mcp GitHub](https://github.com/DeusData/codebase-memory-mcp)
- [codebase-memory-mcp npm](https://www.npmjs.com/package/codebase-memory-mcp)

## ライセンス

MIT License

## コントリビューション

新しいシナリオ、パイプライン、スキルのコントリビューションを歓迎します。コントリビューション時は SOP ガイドラインに従ってください。
