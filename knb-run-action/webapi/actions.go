package webapi

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/akristianlopez/action"
	"github.com/akristianlopez/action/ast"
	"github.com/akristianlopez/action/object"
	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"
	"github.com/golang-jwt/jwt/v5"
	"github.com/hashicorp/consul/api"
	_ "github.com/lib/pq" // Driver PostgreSQL
)

var ErrNotFound = errors.New("not found")
var Db_connect_params *Db_access_params
var Running_mode string
var ExistingService func() ([]*api.ServiceEntry, error)
var IsServiceExists func(entries []*api.ServiceEntry, name string) *api.ServiceEntry
var ExtContext *gin.Context

var microservices []*api.ServiceEntry

func serviceExists(name string) bool {
	if ExistingService == nil || IsServiceExists == nil {
		return false
	}
	if microservices == nil {
		srv, err := ExistingService()
		if err != nil {
			log.Println(err)
			return false
		}
		microservices = srv
	}
	return IsServiceExists(microservices, name) != nil
}
func serviceSignature(ctx *gin.Context, service, name string) ([]*ast.StructField, *ast.TypeAnnotation, error) {
	if ExistingService == nil || IsServiceExists == nil {
		return nil, nil, nil
	}
	req := &RequestData{}
	if err := ctx.BindJSON(req); err != nil {
		return nil, nil, err
	}
	entry := IsServiceExists(microservices, service)
	if entry != nil {
		url := fmt.Sprintf("%s:%d/action/contract/%s/%s/%s/%s/%s", entry.Service.Address, entry.Service.Port,
			service, name, req.Proc, req.Knowledge, req.Role)
		val, err := send2getSignature(ctx, url)
		if err != nil {
			return nil, nil, err
		}
		if val.Error != 0 {
			return nil, nil, fmt.Errorf("Error from eval server: %v", val.Data["msg"])
		}
		args := []*ast.StructField{}
		err = json.Unmarshal([]byte(val.Data["arguments"].(string)), &args)
		if err != nil {
			return nil, nil, err
		}
		ret := &ast.TypeAnnotation{}
		err = json.Unmarshal([]byte(val.Data["returnType"].(string)), ret)
		if err != nil {
			return nil, nil, err
		}
		return args, ret, nil
	}
	return nil, nil, nil
}

type Program interface {
	GetInterface(req RequestData) (string, error)
	Run(req RequestData) (ResponseData, error)
	Fetch(req RequestData) (ResponseData, error)
	Check(req RequestData, id string) (ResponseData, error)
}

type Claims struct {
	Proc     string   `json:"proc"`
	Role     []string `json:"role"`
	Language string   `json:"language"`
	jwt.Claims
}

type security struct {
	claims *Claims
}

func (sec *security) IsHandlabled(table, field, operation string) (bool, string) {
	return true, ""
}
func (sec *security) hasFilter(table string) bool {
	return false
}
func (sec *security) getFilter(table, newName string) (ast.Expression, bool) {
	return nil, false
}

type Signatures struct {
	args    []*ast.StructField
	retType *ast.TypeAnnotation
}
type Action struct {
	secu       *security
	contracts  map[string]*ast.Action
	signatures map[string]*Signatures
	action     map[string]map[string]map[string]*ast.Action
	screen     map[string]map[string]map[string]string
}

func newAction() *Action {
	s := &Action{
		contracts:  make(map[string]*ast.Action),
		signatures: make(map[string]*Signatures),
		action:     make(map[string]map[string]map[string]*ast.Action),
		screen:     make(map[string]map[string]map[string]string),
	}
	return s
}
func (a *Action) getSecretKey() (*[]byte, error) {
	secretKey := []byte("votre_cle_secrete_tres_securisee") // Remplacez par votre clé secrète
	return &secretKey, nil
}
func (a *Action) FillToken(ctx *gin.Context) (bool, error) {
	tokenString, err := getJwt(ctx)
	if err != nil {
		return false, err
	}
	secretKey, _ := a.getSecretKey()

	keyFunc := func(token *jwt.Token) (interface{}, error) {
		// Vérifie que l'algorithme de signature est celui attendu (HS256)
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secretKey, nil
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, keyFunc)
	if err != nil {
		return false, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		a.secu = &security{claims: claims}
		a.fillSecurity(ctx)
		return true, nil
	}
	return false, errors.New("Invalid token")
}
func (a *Action) fillSecurity(ctx *gin.Context) (string, error) {
	// a.secu = &security{}
	return "", nil
}
func (a *Action) IsTokenValid(tok, key string) (bool, error) {
	return true, nil
}
func (a *Action) GetInterface(ctx *gin.Context, req RequestData) (string, error) {
	if qry, ok := a.screen[strings.ToLower(req.Role)][strings.ToLower(req.Proc)][strings.ToLower(req.Knowledge)]; !ok {
		return qry, nil
	}
	qry := fmt.Sprintf(`Action "execute a query"()\n start RETURN SELECT ACTION.SCREEN 
			FROM KNOWLEDGE INNER JOIN ACTION ON (KNOWLEDGE.GOAL=ACTION.GOAL AND KNOWLEDGE.PROC=ACTION.PROC)
				INNER JOIN PROCESS ON (PROCESS.CODE==KNOWLEDGE.PROC)
			WHERE (KNOWLEDGE.GOAL=='%s' AND KNOWLEDGE.PROC=='%s' AND PROCESS.ROLE=='%s')\nstop`,
		req.Knowledge, req.Proc, req.Role)
	db, err := sql.Open(Db_connect_params.Kind, getConnectionString())
	if err != nil {
		return "", fmt.Errorf("Nsina: Error when trying to retrieve the screen from the database : %v", err)
	}
	defer db.Close()
	act := action.NewAction(ctx, db, Db_connect_params.Kind)
	val, erro := act.Interpret(qry, a.secu.IsHandlabled, a.secu.hasFilter, a.secu.getFilter, nil, true, true,
		serviceExists, serviceSignature, a.eval)
	if erro != nil {
		return "", err
	}
	if val == object.NULL {
		return "", ErrNotFound
	}
	if rows, ok := val.(*object.SQLResult); ok {
		for rows.Rows.Next() {
			rows.Rows.Scan(&qry)
		}
		if _, ok := a.screen[strings.ToLower(req.Role)]; !ok {
			a.screen[strings.ToLower(req.Role)] = make(map[string]map[string]string)
		}
		if _, ok := a.screen[strings.ToLower(req.Role)][strings.ToLower(req.Proc)]; !ok {
			a.screen[strings.ToLower(req.Role)][strings.ToLower(req.Proc)] = make(map[string]string)
		}
		if _, ok := a.screen[strings.ToLower(req.Role)][strings.ToLower(req.Proc)][strings.ToLower(req.Knowledge)]; !ok {
			a.screen[strings.ToLower(req.Role)][strings.ToLower(req.Proc)][strings.ToLower(req.Knowledge)] = qry
		}
		return qry, nil
	}
	return "", ErrNotFound
}
func (a *Action) Run(ctx *gin.Context, req RequestData) (*ResponseData, error) {
	db, err := sql.Open(Db_connect_params.Kind, getConnectionString())
	if err != nil {
		return &ResponseData{}, fmt.Errorf("Nsina: Error when trying to retrieve the action from the database : %v", err)
	}
	defer db.Close()
	if ar, ok := req.Data["arguments"]; ok {
		args := (ar.(map[string]interface{}))[req.Data["service"].(string)].(map[string]interface{})[req.Data["contract"].(string)].(map[string]object.Object)
		act := action.NewAction(ctx, db, Db_connect_params.Kind)
		if prog, ok := a.action[strings.ToLower(req.Role)][strings.ToLower(req.Proc)][strings.ToLower(req.Knowledge)]; !ok {
			// Execute prog
			val := act.Execute(prog, a.secu.hasFilter, a.secu.getFilter, args, true, true,
				serviceExists, serviceSignature, a.eval)
			return &ResponseData{Error: 0, Data: map[string]interface{}{"result": val}}, nil
		}
		qry := fmt.Sprintf(`Action "execute a query"()\n start RETURN SELECT ACTION.ACTION 
				FROM KNOWLEDGE INNER JOIN ACTION ON (KNOWLEDGE.GOAL=ACTION.GOAL AND KNOWLEDGE.PROC=ACTION.PROC)
					INNER JOIN PROCESS ON (PROCESS.CODE==KNOWLEDGE.PROC)
				WHERE (KNOWLEDGE.GOAL=='%s' AND KNOWLEDGE.PROC=='%s' AND PROCESS.ROLE=='%s')\nstop`,
			req.Knowledge, req.Proc, req.Role)
		val, erro := act.Interpret(qry, a.secu.IsHandlabled, a.secu.hasFilter, a.secu.getFilter, nil, true, true,
			serviceExists, serviceSignature, a.eval)
		if erro != nil {
			return &ResponseData{}, err
		}
		if val == object.NULL {
			return &ResponseData{}, ErrNotFound
		}
		if rows, ok := val.(*object.SQLResult); ok {
			for rows.Rows.Next() {
				rows.Rows.Scan(&qry)
			}
			if _, ok := a.action[strings.ToLower(req.Role)]; !ok {
				a.action[strings.ToLower(req.Role)] = make(map[string]map[string]*ast.Action)
			}
			if _, ok := a.action[strings.ToLower(req.Role)][strings.ToLower(req.Proc)]; !ok {
				a.action[strings.ToLower(req.Role)][strings.ToLower(req.Proc)] = make(map[string]*ast.Action)
			}
			val, erro = act.Interpret(qry, a.secu.IsHandlabled, a.secu.hasFilter, a.secu.getFilter, nil, true, true,
				serviceExists, serviceSignature, a.eval)

			if erro != nil {
				return &ResponseData{}, fmt.Errorf("Too many errors occured while trying to interpret the action of '%s' from '%s'", req.Knowledge, req.Proc)
			}
			if _, ok := a.action[strings.ToLower(req.Role)][strings.ToLower(req.Proc)][strings.ToLower(req.Knowledge)]; !ok {
				prog, _ := act.Generate(qry, a.secu.IsHandlabled, serviceExists, serviceSignature)
				a.action[strings.ToLower(req.Role)][strings.ToLower(req.Proc)][strings.ToLower(req.Knowledge)] = prog
			}
			return &ResponseData{Error: 0, Data: map[string]interface{}{"result": val}}, nil
		}
	}
	return &ResponseData{}, ErrNotFound
}
func (a *Action) Fetch(ctx *gin.Context, req RequestData) (*ResponseData, error) {
	if qry, ok := req.Data["query"]; ok {
		src := fmt.Sprintf(`Action "execute a query"()\n start RETURN %s)\nstop`, qry.(string))
		db, err := sql.Open(Db_connect_params.Kind, getConnectionString())
		if err != nil {
			return &ResponseData{Error: 1, Data: map[string]interface{}{"msg": fmt.Errorf("Nsina: Error when trying to retrieve the screen from the database : %v", err.Error())}},
				fmt.Errorf("Nsina: Error when trying to retrieve the screen from the database : %v", err)
		}
		defer db.Close()
		act := action.NewAction(ctx, db, Db_connect_params.Kind)
		val, erro := act.Interpret(src, a.secu.IsHandlabled, a.secu.hasFilter, a.secu.getFilter, nil, true, true,
			serviceExists, serviceSignature, a.eval)
		if erro != nil {
			return &ResponseData{Error: 1, Data: map[string]interface{}{"msg": "Too many errors occured while trying to retrieve the query data"}}, errors.New("Too many errors occured while trying to retrieve the query data")
		}
		if val == object.NULL {
			return &ResponseData{Error: 1, Data: map[string]interface{}{"msg": ErrNotFound.Error()}}, ErrNotFound
		}
		if rows, ok := val.(*object.SQLResult); ok {
			_, err1 := rows.Rows.Columns()
			cols, err2 := rows.Rows.ColumnTypes()
			if err1 != nil {
				return &ResponseData{Error: 1, Data: map[string]interface{}{"msg": fmt.Sprintf("Nsina; %s", err1.Error())}}, fmt.Errorf("Nsina; %s", err1.Error())
			}
			if err2 != nil {

				return &ResponseData{Error: 1, Data: map[string]interface{}{"msg": fmt.Sprintf("Nsina; %s", err2.Error())}}, fmt.Errorf("Nsina; %s", err2.Error())
			}

			flusher, ok := ctx.Writer.(http.Flusher)
			if ok {
				ctx.Writer.Header().Set("Content-Type", "text/event-stream")
				ctx.Writer.Header().Set("Cache-Control", "no-cache")
				ctx.Writer.Header().Set("Connection", "keep-alive")
				ctx.Writer.Header().Set("Access-Control-Allow-Origin", "*")
				ctx.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
				for rows.Rows.Next() {
					args := make([]any, 0)
					for k := range rows.Columns {
						args = append(args, act.GetDefaultSQLValueAddress(cols[k].DatabaseTypeName()))
					}
					rows.Rows.Scan(args...)
					row := make(map[string]interface{})
					for k, val := range args {
						row[rows.Columns[k]] = val
					}
					data, err := json.Marshal(row)
					if err != nil {
						log.Println("Error encoding JSON:", err)
						continue
					}
					// Write JSON followed by newline for easy parsing
					fmt.Fprintf(ctx.Writer, "%s\n", data)
					// Flush to send immediately
					flusher.Flush()
				}
				// Optionally send a completion message
				fmt.Fprintln(ctx.Writer, `{"status":"done"}`)
				flusher.Flush()
				return nil, nil
			}
			resp := &ResponseData{Error: 0, Data: map[string]interface{}{"result": make([]map[string]interface{}, 0)}}
			for rows.Rows.Next() {
				args := make([]any, 0)
				for k := range rows.Columns {
					args = append(args, act.GetDefaultSQLValueAddress(cols[k].DatabaseTypeName()))
				}
				rows.Rows.Scan(args...)
				row := make(map[string]interface{})
				for k, val := range args {
					row[rows.Columns[k]] = val
				}
				resp.Data["result"] = append(resp.Data["result"].([]map[string]interface{}), row)
			}
			return resp, nil
		}
	}
	return &ResponseData{Error: 1, Data: map[string]interface{}{"msg": "Query not found"}}, ErrNotFound
}
func (a *Action) Check(ctx *gin.Context, req RequestData, id, table, newName string) (*ResponseData, *[]string, error) {
	result := &ResponseData{Data: make(map[string]interface{})}
	result.Error = 0
	db, err := sql.Open(Db_connect_params.Kind, getConnectionString())
	if err != nil {
		result.Error = 1
		result.Data["Error"] = err.Error()
		// log.Fatalf("Nsina: Error when trying to connect to the database : %v", err)
		return result, nil, err
	}
	defer db.Close()
	if src, ok := req.Data["source"]; ok {
		act := action.NewAction(ctx, db, Db_connect_params.Kind)
		ok, err := act.Check(src.(string), id, table, newName, a.secu.IsHandlabled, serviceExists, serviceSignature)
		if ok {
			return result, nil, nil
		}
		result.Error = 1
		result.Data["Error"] = err
		return result, &err, nil
	}
	result.Error = 1
	result.Data["Error"] = ErrNotFound.Error()
	//ctx context.Context, db *sql.DB, dbname string
	return result, nil, ErrNotFound
}
func (a *Action) getSignature(ctx *gin.Context, proc, goal, role, srv, name string) ([]*ast.StructField, *ast.TypeAnnotation, []string, error) {
	// The usage of this method required to call before the function ExistingService.
	// This function helps to initialize the local variable 'microservices'
	// getSignature returns the signature of the local contract for sending to the consumer

	// Retrieves data about the contract. Espacially the service name and the public contract reference
	service := srv // name of the microservice
	if !strings.EqualFold(ConfigClient.Params["service_name"].(string), service) {
		return nil, nil, nil, fmt.Errorf("microservice name '%s' doesn't match to '%s'",
			ConfigClient.Params["service_name"].(string), service)
	}
	contract := name // name of the public contract
	// check if the req allows this consumer to access to the contract
	// if not return nil, nil, else load the action source code attached to the public contract
	// and extracts the signature.
	db, err := sql.Open(Db_connect_params.Kind, getConnectionString())
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Nsina: Error when trying to connect to the database : %v", err)
	}
	act := action.NewAction(ctx, db, Db_connect_params.Kind)
	defer db.Close()
	src, err := a.getContractText(contract, proc, goal, role, act)
	if err != nil {
		return nil, nil, nil, err
	}
	args, reType, chkErr := act.Signature(src)
	if chkErr != nil {
		return nil, nil, chkErr, fmt.Errorf("Too many errors occured while checking the contract '%s'", contract)
	}
	return args, reType, chkErr, nil
}
func (a *Action) getContractText(name, proc, goal, role string, act *action.Action) (string, error) {
	// This function retrieves the source code of the contract identified by 'name'
	// from the local storage (database, file, etc.)
	str := fmt.Sprintf(`Action "execute a query"()\n start RETURN SELECT RULE.ACTION
			  FROM CONTRACT INNER JOIN RULE ON (CONTRACT.GOAL=RULE.GOAL AND CONTRACT.PROC=RULE.PROC)
			  		INNER JOIN PROCESS ON (PROCESS.CODE==RULE.PROC)
			  WHERE (CONTRACT.NAME=='%s' And CONTRACT.PROC=='%s' And CONTRACT.GOAL=='%s' And PROCESS.ROLE=='%s')\n stop`,
		name, proc, goal, role)
	val, err := act.Interpret(str, a.secu.IsHandlabled, a.secu.hasFilter, a.secu.getFilter, nil, true, true,
		serviceExists, serviceSignature, a.eval)
	if len(err) > 0 {
		return "", fmt.Errorf("Too many errors occured while interpreting the contract '%s'", name)
	}
	if val == object.NULL {
		return "", fmt.Errorf("The contract '%s' doesn't exist", name)
	}
	if qry, ok := val.(*object.SQLResult); ok {
		src := ""
		for qry.Rows.Next() {
			qry.Rows.Scan(&src)
		}
		if src == "" {
			return "", fmt.Errorf("The contract '%s' has no associated action", name)
		}
		return src, nil
	}
	return "", fmt.Errorf("The contract '%s' doesn't exist", name)
}
func (a *Action) Signature(ctx *gin.Context, req RequestData) ResponseData {
	// The usage of this method required to call before the function ExistingService.
	// This function helps to initialize the local variable 'microservices'
	// Signature contacts the remote microservice to get a signature of the contract
	// and transferts it to the consumer
	resp := ResponseData{Error: 0, Data: make(map[string]interface{})}
	select {
	case <-ctx.Done():
		log.Println("Getting signature cancled by the user on : ", time.Now())
		resp.Error = 1
		resp.Data["msg"] = fmt.Sprintf("Getting signature cancled by the user on : %v", time.Now())
		return resp
	default:
		if microservices == nil {
			log.Println("The microservices list is not initialized.", time.Now())
			resp.Error = 1
			resp.Data["msg"] = fmt.Sprintf("The microservices list is not initialized. %v", time.Now())
			return resp
		}
		service := req.Data["service"].(string) // name of the microservice
		if !strings.EqualFold(ConfigClient.Params["service_name"].(string), service) {
			log.Printf("microservice name '%s' doesn't match to '%s'",
				ConfigClient.Params["service_name"].(string), service)
			resp.Error = 1
			resp.Data["msg"] = fmt.Sprintf("microservice name '%s' doesn't match to '%s'",
				ConfigClient.Params["service_name"].(string), service)
			return resp
		}
		contract := req.Data["contract"].(string) // name of the public contract
		if sign, ok := a.signatures[contract]; ok {
			jsData, err := json.MarshalNoEscape(sign.args)
			if err == nil {
				resp.Data["arguments"] = jsData
				jsData, err = json.MarshalNoEscape(sign.retType)
				if err != nil {
					resp.Error = 1
					resp.Data["msg"] = err.Error()
					resp.Data["errors"] = ""
					return resp
				}
				resp.Data["returnType"] = jsData
				return resp
			}
		}
		args, ret, chk, err := a.getSignature(ctx, req.Proc, req.Knowledge, req.Role, service, contract)
		if err != nil {
			resp.Error = 1
			resp.Data["msg"] = err.Error()
			jsData, er := json.MarshalNoEscape(chk)
			if er != nil {
				resp.Data["errors"] = ""
				return resp
			}
			resp.Data["errors"] = string(jsData)
			return resp
		}
		if len(chk) > 0 {
			resp.Error = 1
			resp.Data["msg"] = ""
			jsData, er := json.MarshalNoEscape(chk)
			if er != nil {
				resp.Data["errors"] = ""
				return resp
			}
			resp.Data["errors"] = jsData
			return resp
		}
		jsData, err := json.MarshalNoEscape(args)
		if err != nil {
			resp.Error = 1
			resp.Data["msg"] = err.Error()
			resp.Data["errors"] = ""
			return resp
		}
		resp.Data["arguments"] = jsData
		jsData, err = json.MarshalNoEscape(ret)
		if err != nil {
			resp.Error = 1
			resp.Data["msg"] = err.Error()
			resp.Data["errors"] = ""
			return resp
		}
		resp.Data["returnType"] = jsData
		a.signatures[contract] = &Signatures{args: args, retType: ret}
		return resp
	}
}

func (a *Action) eval(ctx *gin.Context, srv, name string, args map[string]object.Object) (object.Object, bool) {
	// TODO: A Definir
	entry := IsServiceExists(microservices, srv)
	if entry != nil {
		url := fmt.Sprintf("%s:%d/action/contract", entry.Service.Address, entry.Service.Port)
		if ExtContext == nil {
			return newError("Invalid context"), false
		}
		payload := &RequestData{}
		if err := ctx.BindJSON(payload); err != nil {
			return newError("Error %v", err), false
		}
		// check if the request data contains the reference to the contract
		proc := ""
		knb := ""
		role := ""
		if val, ok := payload.Data[strings.ToLower(srv)]; ok {
			ctd := make(map[string]interface{})
			err := json.Unmarshal([]byte((val.(string))), &ctd)
			if err != nil {
				return newError("Error %v", err), false
			}
			cont := ctd[name].(map[string]interface{})
			proc = cont["proc"].(string)
			knb = cont["knowledge"].(string)
			role = cont["role"].(string)
			req := RequestData{
				Proc:      proc,
				Knowledge: knb,
				Role:      role,
				Data:      make(map[string]interface{}),
			}
			req.Data["service"] = srv
			req.Data["contract"] = name
			req.Data["arguments"] = args
			val := send2Eval(ctx, url, req)
			return val, val.Type() == object.ERROR_OBJ
		}
		return newError("Invalid contract name '%s'", name), false
	}
	return newError("Invalid microservice name '%s'", srv), false
}
func (a *Action) ExecContract(ctx *gin.Context, req RequestData) (object.Object, bool) {
	// TODO: Taking into account the storage action
	db, err := sql.Open(Db_connect_params.Kind, getConnectionString())
	if err != nil {
		return newError("Error : %v", err.Error()), false
	}
	defer db.Close()
	if ar, ok := req.Data["arguments"]; ok {
		args := (ar.(map[string]interface{}))[req.Data["service"].(string)].(map[string]interface{})[req.Data["contract"].(string)].(map[string]object.Object)
		act := action.NewAction(ctx, db, Db_connect_params.Kind)
		if prog, ok := a.contracts[req.Data["contract"].(string)]; ok {
			return act.Execute(prog, a.secu.hasFilter, a.secu.getFilter, args, true, false,
				serviceExists, serviceSignature, a.eval), true
		}
		str := fmt.Sprintf(`Action "execute a query"()\n start RETURN SELECT RULE.ACTION
				FROM CONTRACT INNER JOIN RULE ON (CONTRACT.GOAL=RULE.GOAL AND CONTRACT.PROC=RULE.PROC)
						INNER JOIN PROCESS ON (PROCESS.CODE==RULE.PROC)
				WHERE (CONTRACT.NAME=='%s' And CONTRACT.PROC=='%s' And CONTRACT.GOAL=='%s' And PROCESS.ROLE=='%s')\n stop`,
			req.Data["contract"], req.Proc, req.Knowledge, req.Role)
		val, err := act.Interpret(str, a.secu.IsHandlabled, a.secu.hasFilter, a.secu.getFilter, args, true, false,
			serviceExists, serviceSignature, a.eval)
		if len(err) > 0 {
			return newError("Too many errors occured while interpreting the contract '%s'", req.Data["contract"]), false
		}
		prog, _ := act.Generate(str, a.secu.IsHandlabled, serviceExists, serviceSignature)
		a.contracts[req.Data["contract"].(string)] = prog
		return val, true
	}
	return object.NULL, false
}
func getConnectionString() string {
	// Paramètres de connexion (à adapter ou mettre dans des variables d'environnement)
	if Db_connect_params == nil {
		log.Fatalf("The parameters involved in the connection to the database are not defined.")
		return ""
	}

	// Chaîne de connexion PostgreSQL
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		Db_connect_params.Address, Db_connect_params.Port, Db_connect_params.Userid,
		Db_connect_params.Password, Db_connect_params.Name)
}
func getJwt(ctx *gin.Context) (string, error) {
	tokenString := ctx.GetHeader("Authorization")
	if tokenString == "" {
		return "", errors.New("Authorization header required")
	}
	// Extraire le token (enlever "Bearer ")
	parts := strings.Split(tokenString, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", errors.New("Invalid Authorization header format")
	}
	tokenString = parts[1]

	return tokenString, nil
}
