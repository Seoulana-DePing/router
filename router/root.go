package router

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/router/common/log"
	"github.com/router/config"
	"github.com/router/gping"
	"github.com/router/keystore"
	solclient "github.com/router/network/solana"
	"github.com/router/network/ws"
)

type Router struct {
	engine *gin.Engine
	wsHub  *ws.WSHub
	solanaClient *solclient.SolanaClient
	keyPair *solana.PrivateKey
	port   string
	pendingGeoRequests sync.Map
	pendingRequestIds sync.Map
	gpingClient *gping.GpingClient
	log    log.Logger
}

func NewRouter(cfg *config.Config) *Router {
	// Initialize Solana client
    solanaClient := solclient.NewSolanaClient("https://api.devnet.solana.com") 
	keyPair, err := keystore.LoadKeypair(cfg.KeystorePath, cfg.KeystorePassword)
	if err != nil {
		panic(err)
	}
	gpingClient := gping.NewGpingClient(cfg)
	router := &Router{
		engine: gin.New(),
		wsHub:  ws.NewWsHub(),
		solanaClient: solanaClient,
		keyPair: keyPair,
		port:   fmt.Sprintf(":%s", cfg.Port),
		gpingClient: gpingClient,
		log:    log.New("module", "server"),
	}
	router.engine.Use(gin.Logger())
	router.engine.Use(gin.Recovery())
	router.engine.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH"},
		AllowHeaders:     []string{"ORIGIN", "Content-Length", "Content-Type", "Access-Control-Allow-Headers", "Access-Control-Allow-Origin", "Authorization", "X-Requested-With", "expires"},
		ExposeHeaders:    []string{"ORIGIN", "Content-Length", "Content-Type", "Access-Control-Allow-Headers", "Access-Control-Allow-Origin", "Authorization", "X-Requested-With", "expires"},
		AllowCredentials: true,
		AllowOriginFunc: func(origin string) bool {
			return true
		},
		MaxAge: 12 * time.Hour,
	}))
	router.registerHandler()
	return router
}

func (r *Router) Run() error {
	r.log.Info("Http server started", "port", r.port)
	return r.engine.Run(r.port)
}

func (r *Router) Resp(c *gin.Context, status int, resp interface{}) {
	c.JSON(status, resp)
}

func (r *Router) RespOK(c *gin.Context, resp interface{}) {
	c.JSON(http.StatusOK, resp)
}

func (r *Router) RespError(c *gin.Context, status int, err interface{}) {
	c.JSON(status, err)
}

// RegisterPOSTHandler : post handler api 객체의 api handler함수를 등록할때 호출.
func (r *Router) RegisterPOSTHandler(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes {
	return r.engine.POST(relativePath, handlers...)
}

// RegisterGETHandler : get handler api 객체의 api handler함수를 등록할때 호출.
func (r *Router) RegisterGETHandler(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes {
	return r.engine.GET(relativePath, handlers...)
}

func (r *Router) PostRequest(host, url string, req interface{}) ([]byte, int, error) {

	if requestBody, err := json.Marshal(req); err != nil {
		return nil, 0, err
	} else if resp, err := http.Post(strings.Join([]string{host, url}, ""), "application/json", bytes.NewBuffer(requestBody)); err != nil {
		return nil, 0, err
	} else if resp == nil {
		return nil, 0, fmt.Errorf("response is nil")
	} else {
		defer resp.Body.Close()

		if body, err := ioutil.ReadAll((resp.Body)); err != nil {
			return nil, resp.StatusCode, err
		} else {
			return body, resp.StatusCode, nil
		}
	}
}
