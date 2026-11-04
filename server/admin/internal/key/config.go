package key

import "github.com/gogu-x/proxy-access-manager/conf"

type Config struct {
	Domain, XrayBin, XrayAPI, XrayTag, HysteriaStats, HysteriaSecret string
	XrayPort, HysteriaPort                                           int
}

func LoadConfig() Config {
	return Config{
		Domain: conf.Domain, XrayBin: conf.XrayBin, XrayAPI: conf.XrayAPI,
		XrayTag: conf.XrayTag, XrayPort: conf.XrayPort,
		HysteriaStats: conf.HysteriaStats, HysteriaSecret: conf.HysteriaSecret,
		HysteriaPort: conf.HysteriaPort,
	}
}
