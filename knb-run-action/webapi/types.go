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
