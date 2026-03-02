Write-Output "--- Execution du service knb ---"

# $Env:APP_PORT = 8050
# $Env:SERVICE_NAME = "app"
# $Env:CONSUL_HTTP_ADDR = "http://consul.wosa.local:8500"
./knb-run-action -c "127.0.0.1:8500" "knb/default" 
