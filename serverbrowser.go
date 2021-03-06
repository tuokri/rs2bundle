package main

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"encoding/json"

	"github.com/henkman/steamwebapi"
	"github.com/ip2location/ip2location-go"
	"github.com/zserge/webview"
)

type Native struct {
	w    webview.WebView
	key  string
	game string
	dir  string
}

func jsonCleanString(raw string) string {
	return strings.Replace(
		strings.Replace(raw, `\`, `\\`, -1),
		`"`, `\"`, -1)
}

func jsonClean(raw []byte) []byte {
	return bytes.Replace(
		bytes.Replace(raw, []byte(`\`), []byte(`\\`), -1),
		[]byte(`"`), []byte(`\"`), -1)
}

func (n *Native) GetServerList(cb string, showempty bool) {
	go func() {
		filter := `\gamedir\` + n.game
		if !showempty {
			filter += `\empty\1`
		}
		servers, err := steamwebapi.GetServerList(n.key, 5000, filter)
		if err != nil {
			n.w.Dispatch(func() {
				n.w.Eval(fmt.Sprintf(`(function(x) {%s})("%s");`, cb,
					jsonCleanString(`{"error":"`+err.Error()+`"}`)))
			})
			return
		}
		type JSONServer struct {
			Name       string `json:"name"`
			Addr       string `json:"addr"`
			Map        string `json:"map"`
			Players    int    `json:"players"`
			MaxPlayers int    `json:"max_players"`
			Steamid    string `json:"steamid"`
			Country    string `json:"country"`
		}
		jsservers := make([]JSONServer, len(servers))
		for i, server := range servers {
			s := strings.Split(server.Addr, ":")
			country := "un"
			if len(s) > 0 {
				location := ip2location.Get_country_short(s[0])
				country = strings.ToLower(location.Country_short)
			}
			jsservers[i] = JSONServer{
				Name:       server.Name,
				Addr:       server.Addr,
				Map:        server.Map,
				Players:    server.Players,
				MaxPlayers: server.MaxPlayers,
				Steamid:    server.Steamid,
				Country:    country,
			}
		}
		raw, err := json.Marshal(jsservers)
		if err != nil {
			n.w.Dispatch(func() {
				n.w.Eval(fmt.Sprintf(`(function(x) {%s})("%s");`, cb,
					jsonCleanString(`{"error":"`+err.Error()+`"}`)))
			})
			return
		}
		n.w.Dispatch(func() {
			n.w.Eval(fmt.Sprintf(`(function(x) {%s})("%s");`, cb,
				string(jsonClean(raw))))
		})
	}()
}

func (n *Native) RunShowStats(server string) {
	prog := filepath.Join(n.dir, "serverstats.exe")
	cmd := exec.Command("cmd", "/C", "start", prog, "-s", server)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	cmd.Start()
}

func errorPopup(msg string) {
	const HTML = `<!doctype html><html><body><h1>%s</h1></body></html>`
	w := webview.New(webview.Settings{
		Title: "error",
		URL: `data:text/html,` +
			url.PathEscape(fmt.Sprintf(HTML, msg)),
	})
	w.Run()
	w.Exit()
}

func main() {
	var config struct {
		Key  string `json:"key"`
		Game string `json:"game"`
	}
	{
		fd, err := os.Open("serverbrowser.json")
		if err != nil {
			errorPopup(err.Error())
			return
		}
		err = json.NewDecoder(fd).Decode(&config)
		fd.Close()
		if err != nil {
			errorPopup(err.Error())
			return
		}
	}
	ex, err := os.Executable()
	if err != nil {
		errorPopup(err.Error())
		return
	}
	dir := filepath.Dir(ex)
	ip2location.Open(filepath.Join(dir, `IP2LOCATION-LITE-DB1.BIN`))
	url := "file:///" + filepath.ToSlash(dir) + "/r/serverbrowser.html"
	w := webview.New(webview.Settings{
		Title:     "browser",
		URL:       url,
		Width:     1280,
		Height:    680,
		Resizable: true,
	})
	defer w.Exit()
	w.Dispatch(func() {
		w.Bind("native", &Native{w: w,
			key:  config.Key,
			game: config.Game,
			dir:  dir})
	})
	w.Run()
}
