package webapi

var ConfigClient ClientConfig = ClientConfig{Port: 0, Params: make(map[string]interface{})}
var Cfg_srv_addr_map *map[string]interface{}
var Deregister_waiting_time int = 20
