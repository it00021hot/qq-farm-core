package command

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/MQEnergy/go-skeleton/internal/app/model"
	"github.com/MQEnergy/go-skeleton/internal/bootstrap"
	"github.com/MQEnergy/go-skeleton/internal/farm/logic"
	"github.com/MQEnergy/go-skeleton/internal/vars"
	"github.com/urfave/cli/v2"
)

// FarmImportJSON imports legacy qq-farm-bot accounts.json into cn_farm_account.
func FarmImportJSON() *cli.Command {
	return &cli.Command{
		Name: "farmImport", Usage: "从 qq-farm-bot accounts.json 导入农场账号",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "file", Aliases: []string{"f"}, Required: true, Usage: "accounts.json 路径"},
			&cli.StringFlag{Name: "env", Aliases: []string{"e"}, Value: "dev"},
		},
		Action: func(c *cli.Context) error {
			bootstrap.BootService(bootstrap.SQLiteService)
			raw, err := os.ReadFile(c.String("file"))
			if err != nil {
				return err
			}
			var accounts []struct {
				ID       string `json:"id"`
				Name     string `json:"name"`
				Code     string `json:"code"`
				Platform string `json:"platform"`
				Uin      string `json:"uin"`
				QQ       string `json:"qq"`
				Avatar   string `json:"avatar"`
			}
			if err := json.Unmarshal(raw, &accounts); err != nil {
				var wrap struct {
					Accounts []struct {
						ID       string `json:"id"`
						Name     string `json:"name"`
						Code     string `json:"code"`
						Platform string `json:"platform"`
						Uin      string `json:"uin"`
						QQ       string `json:"qq"`
						Avatar   string `json:"avatar"`
					} `json:"accounts"`
				}
				if err2 := json.Unmarshal(raw, &wrap); err2 != nil {
					return err
				}
				accounts = wrap.Accounts
			}
			now, n := uint(time.Now().Unix()), 0
			for _, a := range accounts {
				code := a.Code
				if code == "" {
					code = a.ID
				}
				acc := model.FarmAccount{Name: a.Name, Code: code, Platform: a.Platform, Uin: a.Uin, QQ: a.QQ, Avatar: a.Avatar, Status: vars.StatusNormal, CreatedAt: now, UpdatedAt: now}
				if acc.Platform == "" {
					acc.Platform = "qq"
				}
				if err := vars.DB.Where("code = ?", acc.Code).FirstOrCreate(&acc).Error; err != nil {
					return err
				}
				cfgJSON, _ := json.Marshal(logic.DefaultAccountConfig())
				cfg := model.FarmAccountConfig{AccountID: acc.ID, ConfigJSON: string(cfgJSON), CreatedAt: now, UpdatedAt: now}
				_ = vars.DB.Where("account_id = ?", acc.ID).FirstOrCreate(&cfg).Error
				n++
			}
			fmt.Printf("imported %d accounts\n", n)
			return nil
		},
	}
}
