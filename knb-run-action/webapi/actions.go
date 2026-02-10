package webapi

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/akristianlopez/action"
	"github.com/akristianlopez/action/ast"
	"github.com/akristianlopez/action/object"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql" // Import du driver MySQL/MariaDB
	"github.com/goccy/go-json"
	"github.com/golang-jwt/jwt/v5"
	"github.com/hashicorp/consul/api"
	_ "github.com/lib/pq"           // Driver PostgreSQL
	_ "github.com/mattn/go-sqlite3" // Import du driver SQLite
)

var ErrNotFound = errors.New("not found")
var Db_connect_params *Db_access_params
var Running_mode string
var ExistingService func() ([]*api.ServiceEntry, error)
var IsServiceExists func(entries []*api.ServiceEntry, name string) *api.ServiceEntry
var ExtContext *gin.Context
var rw sync.RWMutex
var Emit func(url, subj, message, token string) (bool, error)
var Brokers []BrokerInfo
var microservices []*api.ServiceEntry
var StdAction *Action
var ReadSecret func(name string) (string, error)

func serviceExists(name string) bool {
	if ExistingService == nil || IsServiceExists == nil {
		return false
	}
	if microservices == nil {
		srv, err := ExistingService()
		if err != nil {
			slog.Error("Invalid service name", "service name", name, "error", err)
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
func findURL(topic string) (string, error) {
	if topic == "" {
		return "", errors.New("Empty topic. Please fill it and reply")
	}
	if len(Brokers) == 0 {
		return "", errors.New("No avalaible brokers")
	}
	url := ""
	for _, broker := range Brokers {
		t := strings.Split(broker.Topic, ".")
		b := strings.Split(topic, ".")
		if strings.EqualFold(t[0], b[0]) {
			url = broker.URL
			break
		}
	}
	if url == "" {
		return "", errors.New("Topic '%s' not found in the list of available brokers")
	}
	return url, nil
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
	claims     *Claims
	filter     map[string]ast.Expression
	excluded   map[string]bool
	context    map[string]map[string][]string
	isInitMode bool
}

func (sec *security) IsHandlabledField(ctx *gin.Context, table, field string) bool {
	if sec.isInitMode {
		return true
	}
	req := &RequestData{}
	if err := ctx.BindJSON(req); err != nil {
		return false
	}
	rw.RLock()
	defer rw.RUnlock()
	if _, ok := sec.excluded[strings.ToLower(fmt.Sprintf("%s.%s.%s.%s.%s", req.Role, req.Proc, req.Knowledge, table, field))]; ok {
		return false
	}
	return true
}
func (sec *security) IsHandlabled(ctx *gin.Context, table, field, operation string) (bool, string) {
	if sec.isInitMode {
		return true, ""
	}
	req := &RequestData{}
	if err := ctx.BindJSON(req); err != nil {
		return false, ""
	}
	if sec.IsHandlabledField(ctx, table, field) {
		rw.RLock()
		defer rw.RUnlock()
		if val, ok := sec.context[strings.ToLower(fmt.Sprintf("%s.%s.%s", req.Role, req.Proc, req.Knowledge))]; ok {
			if array, ok := val[strings.ToLower(table)]; ok {
				for _, v := range array {
					if strings.EqualFold(v, operation) {
						return true, ""
					}
				}
			}
		}
		return false, ""
	}
	return false, ""
}
func (sec *security) hasFilter(ctx *gin.Context, table string) bool {
	if sec.isInitMode {
		return false
	}

	req := &RequestData{}
	if err := ctx.BindJSON(req); err != nil {
		return false
	}
	rw.RLock()
	defer rw.RUnlock()
	if _, ok := sec.filter[strings.ToLower(fmt.Sprintf("%s.%s.%s.%s", req.Role, req.Proc, req.Knowledge, table))]; ok {
		return true
	}
	return false
}
func (sec *security) replace(expr ast.Expression, table, newName string) ast.Expression {
	switch val := expr.(type) {
	case *ast.TypeMember:
		if v, ok := val.Left.(*ast.Identifier); ok {
			if strings.EqualFold(v.Value, table) {
				return &ast.TypeMember{
					Left:  &ast.Identifier{Token: v.Token, Value: newName},
					Right: sec.replace(val.Right, table, newName),
				}
			}
		}
		return &ast.TypeMember{
			Left:  sec.replace(val.Left, table, newName),
			Right: sec.replace(val.Right, table, newName),
		}
	case *ast.InfixExpression:
		return &ast.InfixExpression{
			Left:     sec.replace(val.Left, table, newName),
			Operator: val.Operator,
			Right:    sec.replace(val.Right, table, newName),
		}
	case *ast.PrefixExpression:
		return &ast.PrefixExpression{
			Operator: val.Operator,
			Right:    sec.replace(val.Right, table, newName),
		}
	default:
		return expr
	}
}
func (sec *security) getFilter(ctx *gin.Context, table, newName string) (ast.Expression, bool) {
	req := &RequestData{}
	if err := ctx.BindJSON(req); err != nil {
		return nil, false
	}
	rw.RLock()
	defer rw.RUnlock()
	if val, ok := sec.filter[strings.ToLower(fmt.Sprintf("%s.%s.%s.%s", req.Role, req.Proc, req.Knowledge, table))]; ok {
		if newName == "" {
			return val, true
		}
		return sec.replace(val, table, newName), true
	}
	return nil, false
}
func (sec *security) removeDuplicates(input []string) []string {
	sort.Strings(input)
	seen := make(map[string]bool) // Tracks seen strings
	result := []string{}

	for _, val := range input {
		if _, exists := seen[val]; !exists {
			seen[val] = true
			result = append(result, val)
		}
	}
	return result
}

func (sec *security) load() (bool, error) {
	// loading of the context
	db, err := sql.Open(Db_connect_params.Kind, getConnectionString())
	if err != nil {
		return false, fmt.Errorf("Nsina: Error when trying to retrieve the action from the database : %v", err)
	}
	defer db.Close()
	sec.isInitMode = false
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	act := action.NewAction(c, db, Db_connect_params.Kind)
	// check wether the sysobject exist in the database, otherwise reinitialize the database
	// two databases are allowed : postgresql and mariadb
	sql := ""
	switch strings.ToLower(Db_connect_params.Kind) {
	case "postgres":
		sql = fmt.Sprintf("select table_name from information_schema.tables where upper(table_name) in ('ROLE','PROCESS','KNOWLEDGE','TRANSACTION','CONTEXT','RULE','FILTER','EXCLUDED','IDS','LAN','LABEL','CONTRACT','EVENT') and table_catalog ='%s'", Db_connect_params.Name)
	case "mariadb":
		sql = fmt.Sprintf("select table_name from information_schema.tables where upper(table_name) in ('ROLE','PROCESS','KNOWLEDGE','TRANSACTION','CONTEXT','RULE','FILTER','EXCLUDED','IDS','LAN','LABEL','CONTRACT','EVENT') and table_catalog ='%s'", Db_connect_params.Name)
	default:
		slog.Error("Invalid database kind", "kind", Db_connect_params.Kind)
		os.Exit(1)
	}
	rows, err := db.Query(sql)
	if err != nil {
		return false, err
	}
	cpt := 0
	for rows.Next() {
		cpt++
	}
	if cpt < 1 || cpt != 13 {
		sec.isInitMode = true
		sql = genScriptInit()
		val, err := act.Interpret(sql, sec.IsHandlabled, sec.hasFilter, sec.getFilter, nil, false, false,
			serviceExists, serviceSignature, nil, emit)
		if len(err) > 0 {
			v, _ := json.Marshal(err)
			slog.Error("Initialization of the system security", "source", sql, "error", string(v))
			return false, fmt.Errorf("Too many error occured")
		}
		if val.Type() == object.ERROR_OBJ {
			slog.Error("Initialization of the system security", "source", sql, "error", val.Inspect())
			return false, fmt.Errorf("Initialization of the system security : %s", val.Inspect())
		}
		return true, nil
	}
	rows, err = db.Query("SELECT P.ROLE,C.PROC,C.GOAL,C.OBJECT,C.TRANS FROM CONTEXT C INNER JOIN PROCESS P ON (P.CODE= C.PROC)")
	if err != nil {
		return false, err
	}
	var role, proc, goal, object, trans string
	for rows.Next() {
		err = rows.Scan(&role, &proc, &goal, &object, &trans)
		if err != nil {
			return false, err
		}
		if _, ok := sec.context[strings.ToLower(fmt.Sprintf("%s.%s.%s", role, proc, goal))]; !ok {
			sec.context[strings.ToLower(fmt.Sprintf("%s.%s.%s", role, proc, goal))] = make(map[string][]string)
		}
		if _, ok := sec.context[strings.ToLower(fmt.Sprintf("%s.%s.%s", role, proc, goal))][strings.ToLower(object)]; !ok {
			sec.context[strings.ToLower(fmt.Sprintf("%s.%s.%s", role, proc, goal))][strings.ToLower(object)] = make([]string, 0)
		}
		rw.Lock()
		sec.context[strings.ToLower(fmt.Sprintf("%s.%s.%s", role, proc, goal))][strings.ToLower(object)] = append(sec.context[strings.ToLower(fmt.Sprintf("%s.%s.%s", role, proc, goal))][strings.ToLower(object)], trans)
		rw.Unlock()
	}

	for key := range sec.context {
		rw.Lock()
		for name := range sec.context[key] {
			sec.context[key][name] = sec.removeDuplicates(sec.context[key][name])
		}
		rw.Unlock()
	}

	// load filter
	rows, err = db.Query("SELECT P.ROLE, F.PROC, F.GOAL, F.OBJECT, F.EXPRESSSION FROM FILTER F INNER JOIN PROCESS P ON (P.CODE= F.PROC)")
	if err != nil {
		return false, err
	}
	sec.filter = make(map[string]ast.Expression)
	for rows.Next() {
		err = rows.Scan(&role, &proc, &goal, &object, &trans)
		if err != nil {
			return false, err
		}
		expr, erList := act.Expression(trans, object, "", sec.IsHandlabled)
		if len(erList) > 0 {
			// sec.filter[strings.ToLower(strings.ToLower(fmt.Sprintf("%s.%s.%s", role, proc, goal)))] = nil
			continue
		}
		if _, ok := sec.filter[strings.ToLower(strings.ToLower(fmt.Sprintf("%s.%s.%s.%s", role, proc, goal, object)))]; !ok {
			rw.Lock()
			sec.filter[strings.ToLower(strings.ToLower(fmt.Sprintf("%s.%s.%s.%s", role, proc, goal, object)))] = expr
			rw.Unlock()
			continue
		}
	}

	// loading of excluded fields
	rows, err = db.Query("SELECT P.ROLE, E.PROC, E.GOAL, E.OBJECT, E.FIELD FROM EXCLUDED E INNER JOIN PROCESS P ON (P.CODE= E.PROC)")
	if err != nil {
		return false, err
	}
	sec.excluded = make(map[string]bool)
	for rows.Next() {
		err = rows.Scan(&role, &proc, &goal, &object, &trans)
		if err != nil {
			continue
		}
		if _, ok := sec.excluded[strings.ToLower(strings.ToLower(fmt.Sprintf("%s.%s.%s.%s", role, proc, goal, object)))]; !ok {
			rw.Lock()
			sec.excluded[strings.ToLower(strings.ToLower(fmt.Sprintf("%s.%s.%s.%s", role, proc, goal, object)))] = true
			rw.Unlock()
			continue
		}
	}
	return true, nil
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
	knb        map[string]bool
	screen     map[string]map[string]map[string]string
}

func newAction() *Action {
	s := &Action{
		secu:       &security{},
		contracts:  make(map[string]*ast.Action),
		signatures: make(map[string]*Signatures),
		action:     make(map[string]map[string]map[string]*ast.Action),
		screen:     make(map[string]map[string]map[string]string),
	}
	_, err := s.secu.load()
	if err != nil {
		slog.Error("Error at the initialization of the step", "error", err.Error())
		os.Exit(1)
	}
	return s
}
func (a *Action) getSecretKey() (string, error) {
	return ReadSecret("jwt_key")
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

func IsTokenValid(tok, key string) bool {
	return true
}
func ValidateToken(token, key string) bool {
	// 1. Tenter de valider avec la clé actuelle en mémoire
	if IsTokenValid(token, ConfigClient.Params["jwt_key"].(string)) {
		return true
	}

	// 2. Si ça échoue, on vérifie si le fichier sur le disque a été mis à jour
	// (Utile uniquement si vous utilisez un volume partagé ou un agent externe)
	switch ConfigClient.Params["jwt_key"].(string) {
	case "swarm":
		freshKey, _ := ReadSecret("jwt_key")
		if freshKey != ConfigClient.Params["jwt_key"].(string) {
			slog.Info("🔄 Nouvelle clé détectée sur le disque, mise à jour de la mémoire...")
			ConfigClient.Params["jwt_key"] = freshKey
			return IsTokenValid(token, ConfigClient.Params["jwt_key"].(string))
		}
	case "standalone":
		return IsTokenValid(token, ConfigClient.Params["jwt_key"].(string))
	}
	return false
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
		serviceExists, serviceSignature, a.eval, emit)
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
				serviceExists, serviceSignature, a.eval, emit)
			return &ResponseData{Error: 0, Data: map[string]interface{}{"result": val}}, nil
		}
		qry := fmt.Sprintf(`Action "execute a query"()\n start RETURN SELECT ACTION.ACTION,KNOWLEDGE.REFRESHABLE
				FROM KNOWLEDGE INNER JOIN ACTION ON (KNOWLEDGE.GOAL=ACTION.GOAL AND KNOWLEDGE.PROC=ACTION.PROC)
					INNER JOIN PROCESS ON (PROCESS.CODE==KNOWLEDGE.PROC)
				WHERE (KNOWLEDGE.GOAL=='%s' AND KNOWLEDGE.PROC=='%s' AND PROCESS.ROLE=='%s')\nstop`,
			req.Knowledge, req.Proc, req.Role)
		flag := 0
		val, erro := act.Interpret(qry, a.secu.IsHandlabled, a.secu.hasFilter, a.secu.getFilter, nil, true, true,
			serviceExists, serviceSignature, a.eval, emit)
		if erro != nil {
			return &ResponseData{}, err
		}
		if val == object.NULL {
			return &ResponseData{}, ErrNotFound
		}
		if rows, ok := val.(*object.SQLResult); ok {
			for rows.Rows.Next() {
				rows.Rows.Scan(&qry, &flag)
			}
			if _, ok := a.action[strings.ToLower(req.Role)]; !ok {
				a.action[strings.ToLower(req.Role)] = make(map[string]map[string]*ast.Action)
			}
			if _, ok := a.action[strings.ToLower(req.Role)][strings.ToLower(req.Proc)]; !ok {
				a.action[strings.ToLower(req.Role)][strings.ToLower(req.Proc)] = make(map[string]*ast.Action)
			}
			a.knb[strings.ToLower(fmt.Sprintf("%s.%s", req.Proc, req.Knowledge))] = flag > 0
			val, erro = act.Interpret(qry, a.secu.IsHandlabled, a.secu.hasFilter, a.secu.getFilter, nil, true, true,
				serviceExists, serviceSignature, a.eval, emit)
			if flag > 0 {
				go a.secu.load()
			}
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
			serviceExists, serviceSignature, a.eval, emit)
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
						slog.Error("Error encoding JSON", "error", err)
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
		serviceExists, serviceSignature, a.eval, emit)
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
		slog.Error("Getting signature cancled by the user")
		resp.Error = 1
		resp.Data["msg"] = fmt.Sprintf("Getting signature cancled by the user on : %v", time.Now())
		return resp
	default:
		if microservices == nil {
			slog.Error("The microservices list is not initialized.")
			resp.Error = 1
			resp.Data["msg"] = fmt.Sprintf("The microservices list is not initialized. %v", time.Now())
			return resp
		}
		service := req.Data["service"].(string) // name of the microservice
		if !strings.EqualFold(ConfigClient.Params["service_name"].(string), service) {
			slog.Error(fmt.Sprintf("microservice name '%s' doesn't match to '%s'",
				ConfigClient.Params["service_name"].(string), service))
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
				serviceExists, serviceSignature, a.eval, emit), true
		}
		str := fmt.Sprintf(`Action "execute a query"()\n start RETURN SELECT RULE.ACTION
				FROM CONTRACT INNER JOIN RULE ON (CONTRACT.GOAL=RULE.GOAL AND CONTRACT.PROC=RULE.PROC)
						INNER JOIN PROCESS ON (PROCESS.CODE==RULE.PROC)
				WHERE (CONTRACT.NAME=='%s' And CONTRACT.PROC=='%s' And CONTRACT.GOAL=='%s' And PROCESS.ROLE=='%s')\n stop`,
			req.Data["contract"], req.Proc, req.Knowledge, req.Role)
		val, err := act.Interpret(str, a.secu.IsHandlabled, a.secu.hasFilter, a.secu.getFilter, args, true, false,
			serviceExists, serviceSignature, a.eval, emit)
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
		slog.Error("The parameters involved in the connection to the database are not defined.")
		os.Exit(1)
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
func HandleBrokerMessage(url, topic, subj, token string, data []byte) bool {
	// find the right knowled which matches with the subject
	// load it and transmit to it the data
	// run it and right eventually the errors status in the logfile

	db, err := sql.Open(Db_connect_params.Kind, getConnectionString())
	if err != nil {
		slog.Error("Nsina: Error when trying to retrieve the action from the database", "error", err)
		return false
	}
	defer db.Close()

	cfg := make(map[string]interface{})
	err = json.Unmarshal(data, &cfg)
	if err != nil {
		slog.Error("Nsina: Error when trying to consume data from the broker", "broker", url, "data", string(data), "error", err)
		return false
	}
	v := make(map[string]object.Object)
	for key, val := range cfg {
		if obj, ok := val.(object.Object); ok {
			v[key] = obj
		}
	}

	rows, err := db.Query(fmt.Sprintf(`SELECT R.ACTION, R.PROC, R.GOAL, P.ROLE 
	 FROM EVENT EV INNER JOIN RULE R ON (EV.PROC==R.PROC AND EV.GOAL==R.GOAL )
	 	  INNER JOIN PROCESS P ON (R.PROC==P.CODE)
	 WHERE (EV.URL=='%s' AND EV.SUBJECT=='%s)`, strings.ToLower(url), strings.ToLower(topic)))

	if err != nil {
		slog.Error("Consume data: No contract found", "url", url, "subject", topic, "error", err)
		return false
	}
	if rows.Next() {
		qry := ""
		req := &RequestData{}
		rows.Scan(&qry, &req.Proc, &req.Knowledge, &req.Role)
		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)
		ctx.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		body := new(bytes.Buffer)
		mw := multipart.NewWriter(body)
		defer mw.Close()
		mw.WriteField("proc", req.Proc)
		mw.WriteField("role", req.Role)
		mw.WriteField("knowledge", req.Knowledge)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/event", nil)
		ctx.Request.Header.Set("Content-Type", "encoding/json")

		act := action.NewAction(ctx, db, Db_connect_params.Kind)
		result, erro := act.Interpret(qry, StdAction.secu.IsHandlabled, StdAction.secu.hasFilter,
			StdAction.secu.getFilter, v, false, false,
			serviceExists, serviceSignature, StdAction.eval, emit)
		if erro != nil {
			slog.Error("Consume data: Too many errors occured while trying to treat the subject",
				"url", url, "subject", topic, "error", erro)
			return false
		}
		if result.Type() == object.ERROR_OBJ {
			slog.Error("Consume data: Interpretor error", "url", url, "subject", topic,
				"error", result.Inspect())
			return false
		}
		return true
	}
	slog.Warn("Consume data: No subject handler found", "url", url, "subject", topic)
	return false
}
func emit(ctx *gin.Context, subject string, message any) bool {
	if Emit == nil {
		return false
	}
	select {
	case <-ctx.Done():
		slog.Warn("Publish message cancled by the user", "subject", subject)
		return false
	default:
		url, err := findURL(subject)
		if err != nil {
			slog.Error("Publish message: Can't find url", "subject", subject, "error", err)
			return false
		}
		var msg []byte
		if t, ok := message.(*object.Struct); ok {
			msg, err = json.Marshal(t)
			if err != nil {
				slog.Error("Publish message: can't treat data", "subject", subject, "data", message, "error", err)
				return false
			}
		}
		if t, ok := message.(*object.Array); ok {
			msg, err = json.Marshal(t)
			if err != nil {
				slog.Error("Publish message: can't treat data", "subject", subject, "data", message, "error", err)
				return false
			}
		}
		switch t := message.(type) {
		case *object.String, *object.Integer, *object.Float,
			*object.Boolean, *object.Date, *object.Duration:
			msg, err = json.Marshal(t)
			if err != nil {
				slog.Error("Publish message: can't treat data", "subject", subject, "data", message, "error", err)
				return false
			}
		default:
			slog.Error("Publish message: Invalid broker data type", "subject", subject, "data", message, "error", err)
			return false
		}
		token, err := authMiddleware(ctx)
		if err != nil {
			slog.Error("Publish message:", "subject", subject, "data", message, "error", err)
			return false
		}
		ok, err := Emit(url, subject, string(msg), token)
		if err != nil {
			slog.Error("Publish message:", "subject", subject, "data", message, "error", err)
			return false
		}
		return ok
	}
}

func authMiddleware(c *gin.Context) (string, error) {
	// 1. Get the header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("Authorization header required")
	}
	// 2. Remove "Bearer " prefix
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == "" {
		return "", fmt.Errorf("Invalid Authorization header format")
	}
	return tokenString, nil
}
