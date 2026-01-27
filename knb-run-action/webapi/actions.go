package webapi

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/akristianlopez/action"
	"github.com/akristianlopez/action/ast"
	"github.com/akristianlopez/action/object"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/hashicorp/consul/api"
	_ "github.com/lib/pq" // Driver PostgreSQL
)

var ErrNotFound = errors.New("not found")
var Db_connect_params *Db_access_params
var Running_mode string
var ExistingService func() ([]*api.ServiceEntry, error)
var IsServiceExists func(entries []*api.ServiceEntry, name string) *api.ServiceEntry

var mocroservices []*api.ServiceEntry

func serviceExists(name string) bool {
	if ExistingService == nil || IsServiceExists == nil {
		return false
	}
	if mocroservices == nil {
		srv, err := ExistingService()
		if err != nil {
			log.Println(err)
			return false
		}
		mocroservices = srv
	}
	return IsServiceExists(mocroservices, name) != nil
}
func serviceSignature(service, name string) ([]*ast.StructField, *ast.TypeAnnotation, error) {
	if ExistingService == nil || IsServiceExists == nil {
		return nil, nil, nil
	}
	// if mocroservices == nil {
	// 	srv, err := ExistingService()
	// }
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

type Action struct {
	secu *security
}

func newAction() *Action {
	s := &Action{}
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
	return "", nil
}
func (a *Action) Run(ctx *gin.Context, req RequestData) (*ResponseData, error) {
	return &ResponseData{}, nil
}
func (a *Action) Fetch(ctx *gin.Context, req RequestData) (*ResponseData, error) {

	return &ResponseData{}, nil
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
		act := action.NewAction(ctx.Request.Context(), db, Db_connect_params.Kind)
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

func (a *Action) Signature(ctx *gin.Context, req RequestData) ([]*ast.StructField, *ast.TypeAnnotation) {
	return nil, nil
}
func (a *Action) ExecContract(ctx *gin.Context, req RequestData) object.Object {
	return nil
}

// func getEnv(key, defaultValue string) string {
// 	if value, exists := os.LookupEnv(key); exists {
// 		return value
// 	}
// 	return defaultValue
// }

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
