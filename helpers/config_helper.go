package helpers

import (
	"fmt"

	beego "github.com/beego/beego/v2/server/web"
)

func ConfigString(key string) string {
	value, err := beego.AppConfig.String(key)
	if err != nil {
		panic(fmt.Errorf("configuración %q no disponible: %w", key, err))
	}
	return value
}
