package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed frontend/dist
var assets embed.FS

type App struct {
	ctx      context.Context
	service  *DatabaseService
	profiles *ProfileStore
}

func NewApp() *App {
	profiles, err := NewProfileStore()
	if err != nil {
		log.Printf("ローカル設定DBを初期化できません: %v", err)
	}
	return &App{service: NewDatabaseService(), profiles: profiles}
}

// Ping confirms that the Wails bridge is available.
func (a *App) Ping() string { return "ok" }

func (a *App) ConnectJSON(configuration string) (string, error) {
	var config ConnectionConfiguration
	if err := json.Unmarshal([]byte(configuration), &config); err != nil {
		return "", err
	}
	result, err := a.service.Connect(config)
	if err != nil {
		return "", err
	}
	return marshalJSON(result)
}

func (a *App) DisconnectDB() error { return a.service.Disconnect() }

func (a *App) ListProfilesJSON() (string, error) {
	if a.profiles == nil {
		return "[]", nil
	}
	profiles, err := a.profiles.List()
	if err != nil {
		return "", err
	}
	return marshalJSON(profiles)
}

func (a *App) SaveProfileJSON(profileJSON, password string) (string, error) {
	if a.profiles == nil {
		return "", fmt.Errorf("ローカル設定DBを利用できません")
	}
	var profile ConnectionProfile
	if err := json.Unmarshal([]byte(profileJSON), &profile); err != nil {
		return "", err
	}
	saved, err := a.profiles.Save(profile, password)
	if err != nil {
		return "", err
	}
	return marshalJSON(saved)
}

func (a *App) DeleteProfile(id string) error {
	if a.profiles == nil {
		return nil
	}
	return a.profiles.Delete(id)
}

func (a *App) ConnectProfile(id string) (string, error) {
	if a.profiles == nil {
		return "", fmt.Errorf("ローカル設定DBを利用できません")
	}
	config, err := a.profiles.LoadConfig(id)
	if err != nil {
		return "", err
	}
	info, err := a.service.Connect(config)
	if err != nil {
		return "", err
	}
	return marshalJSON(info)
}

func (a *App) ListDatabasesJSON() (string, error) {
	result, err := a.service.ListDatabases()
	if err != nil {
		return "", err
	}
	return marshalJSON(result)
}

func (a *App) ListTablesJSON() (string, error) {
	result, err := a.service.ListTables()
	if err != nil {
		return "", err
	}
	return marshalJSON(result)
}

func (a *App) LoadTableJSON(schema, table string, maxRows int) (result string, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			result = ""
			err = fmt.Errorf("テーブルの読み込み中に予期しないエラーが発生しました: %v", recovered)
		}
	}()
	data, err := a.service.LoadTable(schema, table, maxRows)
	if err != nil {
		return "", err
	}
	return marshalJSON(data)
}

func (a *App) UpdateCellJSON(schema, table, column, primaryKey, primaryKeyValue, value string) error {
	return a.service.UpdateCell(schema, table, column, primaryKey, primaryKeyValue, value)
}

func marshalJSON(value interface{}) (string, error) {
	data, err := json.Marshal(value)
	return string(data), err
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.service.SetContext(ctx)
}

func (a *App) shutdown(ctx context.Context) {
	a.service.Close()
	if a.profiles != nil {
		_ = a.profiles.Close()
	}
}

func (a *App) showError(message string) {
	if a.ctx != nil {
		runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
			Title:   "DB Access",
			Message: message,
			Type:    runtime.ErrorDialog,
		})
	}
}

func main() {
	app := NewApp()
	if err := wails.Run(&options.App{
		Title:                    "DB Access",
		Width:                    1440,
		Height:                   900,
		MinWidth:                 980,
		MinHeight:                640,
		AssetServer:              &assetserver.Options{Assets: assets},
		BackgroundColour:         &options.RGBA{R: 18, G: 18, B: 20, A: 1},
		OnStartup:                app.startup,
		OnShutdown:               app.shutdown,
		Bind:                     []interface{}{app},
		Frameless:                false,
		EnableDefaultContextMenu: true,
	}); err != nil {
		log.Fatal(err)
	}
}
