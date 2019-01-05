package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/lukaszbudnik/auditor/hash"
	"github.com/lukaszbudnik/auditor/model"
	"github.com/lukaszbudnik/auditor/store"
	"github.com/lukaszbudnik/auditor/store/provider"
	"github.com/lukaszbudnik/migrator/common"
	"gopkg.in/validator.v2"
)

const (
	defaultPort     string = "8080"
	requestIDHeader string = "X-Request-Id"
)

var lock = &sync.Mutex{}

func newStore() (store.Store, error) {
	return provider.NewStore()
}

func getLimit(r *http.Request) int64 {
	s := r.URL.Query().Get("limit")
	limit, err := strconv.ParseInt(s, 10, 64)
	if err != nil || limit < 0 {
		return int64(100)
	}
	return limit
}

func getLastBlock(r *http.Request) *model.Block {
	s := r.URL.Query().Get("timestamp")
	time, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return nil
	}
	return &model.Block{Timestamp: &time}
}

func errorResponse(w http.ResponseWriter, errorStatus int, response interface{}) {
	w.WriteHeader(errorStatus)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func errorResponseWithStatusAndErrorMessage(w http.ResponseWriter, errorStatus int, errorMessage string) {
	errorResponse(w, errorStatus, struct{ ErrorMessage string }{errorMessage})
}

func errorDefaultResponse(w http.ResponseWriter, errorStatus int) {
	errorResponseWithStatusAndErrorMessage(w, errorStatus, http.StatusText(errorStatus))
}

func errorInternalServerErrorResponse(w http.ResponseWriter, err error) {
	errorResponseWithStatusAndErrorMessage(w, http.StatusInternalServerError, err.Error())
}

func jsonResponse(w http.ResponseWriter, response interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func okResponseWithMessage(w http.ResponseWriter, hash, previousHash string) {
	jsonResponse(w, struct {
		Hash         string
		PreviousHash string
	}{hash, previousHash})
}

func tracing(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// requestID
		requestID := r.Header.Get(requestIDHeader)
		if requestID == "" {
			requestID = fmt.Sprintf("%d", time.Now().UnixNano())
		}
		ctx := context.WithValue(r.Context(), common.RequestIDKey{}, requestID)
		// action
		action := fmt.Sprintf("%v %v", r.Method, r.RequestURI)
		ctx = context.WithValue(ctx, common.ActionKey{}, action)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func makeHandler(handler func(http.ResponseWriter, *http.Request, func() (store.Store, error), func(object interface{}) ([]byte, error)), newStore func() (store.Store, error), serialize func(object interface{}) ([]byte, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handler(w, r, newStore, serialize)
	}
}

func auditHandler(w http.ResponseWriter, r *http.Request, newStore func() (store.Store, error), serialize func(object interface{}) ([]byte, error)) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		common.LogError(r.Context(), "Wrong method: %v", r.Method)
		errorDefaultResponse(w, http.StatusMethodNotAllowed)
		return
	}
	common.LogInfo(r.Context(), "Start")
	if r.Method == http.MethodGet {
		auditGetHandler(w, r, newStore)
	}
	if r.Method == http.MethodPost {
		auditPostHandler(w, r, newStore, serialize)
	}
}

func auditGetHandler(w http.ResponseWriter, r *http.Request, newStore func() (store.Store, error)) {
	store, err := newStore()
	if err != nil {
		common.LogError(r.Context(), "Internal server error - could not connect to backend store: %v", err.Error())
		errorInternalServerErrorResponse(w, err)
		return
	}
	defer store.Close()
	limit := getLimit(r)
	lastBlock := getLastBlock(r)

	audit := []model.Block{}
	err = store.Read(&audit, limit, &lastBlock)
	if err != nil {
		errorInternalServerErrorResponse(w, err)
		return
	}

	jsonResponse(w, audit)
}

func auditPostHandler(w http.ResponseWriter, r *http.Request, newStore func() (store.Store, error), serialize func(object interface{}) ([]byte, error)) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		common.LogError(r.Context(), "Error reading request: %v", err.Error())
		errorInternalServerErrorResponse(w, err)
		return
	}

	block := &model.Block{}
	err = json.Unmarshal(body, block)
	if err != nil {
		common.LogError(r.Context(), "Bad request: %v", err.Error())
		errorResponseWithStatusAndErrorMessage(w, http.StatusBadRequest, err.Error())
		return
	}
	err = validator.Validate(block)
	if err != nil {
		common.LogError(r.Context(), "Validation error: %v", err.Error())
		errorResponseWithStatusAndErrorMessage(w, http.StatusBadRequest, err.Error())
		return
	}

	store, err := newStore()
	if err != nil {
		common.LogError(r.Context(), "Internal server error - could not connect to backend store: %v", err.Error())
		errorInternalServerErrorResponse(w, err)
		return
	}
	defer store.Close()

	lock.Lock()
	defer lock.Unlock()

	audit := []model.Block{}
	err = store.Read(&audit, 1, nil)
	if err != nil {
		errorInternalServerErrorResponse(w, err)
		return
	}

	if len(audit) == 1 {
		previousBlock := &audit[0]
		block.PreviousHash = previousBlock.Hash
	}

	hash, err := hash.ComputeHashWithSerialize(block, serialize)
	if err != nil {
		errorInternalServerErrorResponse(w, err)
		return
	}
	block.Hash = hash

	err = store.Save(block)
	if err != nil {
		errorInternalServerErrorResponse(w, err)
		return
	}

	okResponseWithMessage(w, block.Hash, block.PreviousHash)
}

func registerHandlers() *http.ServeMux {
	router := http.NewServeMux()
	router.Handle("/", http.NotFoundHandler())
	router.Handle("/audit", makeHandler(auditHandler, newStore, hash.Serialize))
	return router
}

// Start starts simple Auditor API
func Start() (*http.Server, error) {
	log.Printf("INFO auditor starting on port %s", defaultPort)

	router := registerHandlers()

	server := &http.Server{
		Addr:    ":" + defaultPort,
		Handler: tracing(router),
	}

	err := server.ListenAndServe()

	return server, err
}
