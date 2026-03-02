Write-Output "--- Execution du shell ---"

$Env:PORT = 8080
$Env:SERVICE_NAME = "app"
$Env:CONSUL_ADDR = "http://consul.wosa.local:8500"
$Env:CONFIG_PATH = "knb/services/shell"
$Env:SERVICE_DOMAIN= "wosa.local"
./knb-shell
