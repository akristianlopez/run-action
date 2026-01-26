package webapi

type RequestData struct {
	Proc      string                 `json:"proc"`
	Role      string                 `json:"role"`
	Knowledge string                 `json:"Knowledge"`
	Data      map[string]interface{} `json:"data"`
}

type ResponseData struct {
	Error int                    `json:"error"`
	Data  map[string]interface{} `json:"data"`
}

type Db_access_params struct {
	Userid   string
	Password string
	Port     int64
	Address  string
	Name     string
}
type Config struct {
	Database struct {
		Address string `yaml:"url" json:"url"`
		Port    int    `yaml:"port" json:"port"`
		Usrid   string `yaml:"username" json:"username"`
		Name    string `yaml:"database" json:"database"`
		// Password string `yaml:"password" json:"password"`
	} `yaml:"database" json:"database"`
	Kafka struct {
		Brokers []BrokerInfo `yaml:"brokers" json:"brokers"`
	} `yaml:"kafka" json:"kafka"`
	Vault struct {
		URL   string `yaml:"url" json:"url"`
		Path  string `yaml:"path" json:"path"`
		Token string `yaml:"token" json:"token"`
	} `yaml:"vault" json:"vault"`
	Consul struct {
		URL                   string `yaml:"url" json:"address"`
		Health_check_interval int    `yaml:"check_delay" json:"check_delay"`
		Timeout               int    `yaml:"timeout" json:"timeout"`
		Deregistry_delay_time int    `yaml:"deregistry_delay" json:"deregistry_delay"`
	} `yaml:"consul" json:"consul"`
}
type ClientConfig struct {
	Params map[string]interface{}
	Port   int
}
type DefaultConfig struct {
	Params map[string]Default
}

type Default struct {
	Name string `yaml:"name"`
	Path string `yaml:"path"`
	Port int    `yaml:"port"`
} //`yaml:"default" json:"default"`
type BrokerInfo struct {
	URL   string `yaml:"url" json:"url"`
	Topic string `yaml:"topic" json:"topic"`
} //`yaml:"default" json:"default"`
