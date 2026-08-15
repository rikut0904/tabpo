# tabpo

PostgreSQL向けのデスクトップデータベースクライアントです。Wails + React/TypeScript + Go `database/sql`で構成し、macOS・Windowsでの動作を対象にしています。

## 開発環境

- Go 1.26+
- Node.js / npm
- Wails v2 CLI
- PostgreSQL（接続テスト時）

Windows版のビルドには、Windows 10/11、WebView2 Runtime、Go、Node.js/npm、Wails CLIが必要です。Linux版のビルドには、Go、Node.js/npm、Wails CLIに加えてGTK 3とWebKitGTKの開発パッケージが必要です。

## 初回セットアップ

環境に合わせて初期化します。Wails CLIのインストール、フロントエンド依存関係の導入、Linuxに必要なGTK/WebKitGTKの導入を行います。このコマンドで書くプラットフォームを自動判定するためwindows,mac,linuxいずれでも同じコマンドで実行可能です。

```sh
make init
```

`wails: command not found`になる場合は、GoのbinディレクトリをPATHへ追加してください。毎回設定したくない場合は、`~/.zshrc`へ追加します。

```sh
echo 'export PATH="$(go env GOPATH)/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

フロントエンドの依存関係をインストールします。

```sh
make install
```

## 起動

プロジェクトルートで、以下を実行します。

```sh
make up
```

`make up`は、フロントエンドをビルドしてからWailsの開発アプリを起動します。アプリを終了する場合は、ターミナルで`Ctrl-C`を押してください。

## 各プラットフォームでのダウンロード

```bash
make build
```

を実行することでどのプラットフォームかを自動判別して自動でビルドが行われます。ビルド後、以下のディレクトリに生成された実行ファイルを起動してご利用ください。

- Windows: `build/win-amd64/tabpo.exe`
- macOS: `build/mac-arm64/tabpo.app`
- Linux: `build/linux-amd64/tabpo`

## 接続の流れ

1. 未接続時に表示される接続フォームへPostgreSQLのホスト、ポート、接続先DB、ユーザー名、パスワードを入力する
2. 「接続」を押して指定したDBへ接続する
3. テーブルを選択してデータを閲覧する
4. 主キーがあるテーブルでは、セルをクリックして値を変更する

## 現在の実装範囲

- PostgreSQLへの接続・切断
- 接続設定フォーム
- 接続設定で指定したPostgreSQLデータベースへの接続
- スキーマ内のテーブル一覧表示
- 最大100行のテーブルデータ閲覧
- 主キーを持つテーブルのセル編集・保存
- 任意SQLは実行せず、内部で固定クエリのみを利用
- WailsのGo Bindingによるフロントエンド連携
- 保存済み接続プロファイルの保存・再利用・削除
- 接続プロファイルの設定値はSQLite、パスワードはOSの資格情報ストアへ保存
- OS標準のファイルオープナーで第三者ライセンスを表示
- アプリメニューの「操作」から接続・切断・更新を実行

ローカル保存データはOSのユーザー設定ディレクトリに`DB Access/profiles.db`として作成されます。パスワードはこのSQLiteファイルには保存されず、OSの資格情報ストアへ保存されます。

## ライセンス

Copyright (c) 2026 rikut0904

本ソフトウェアは、コンパイル済みアプリケーションを無料で利用できる独自ライセンスで公開しています。ソースコードの利用、改変、再配布および派生作品の公開は許可していません。詳細は[LICENSE](./LICENSE)を参照してください。

第三者依存ライブラリのライセンス一覧は[THIRD_PARTY_NOTICES.md](./THIRD_PARTY_NOTICES.md)に記載しています。配布時はこのファイルもDMGに同梱してください。
