package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/lukaszbudnik/auditor/model"
	"github.com/lukaszbudnik/auditor/store"
	"github.com/lukaszbudnik/migrator/common"
	"gopkg.in/validator.v2"
)

const (
	defaultPort     string = "8080"
	requestIDHeader string = "X-Request-Id"
)

func getLimit(r *http.Request) int64 {
	s := r.URL.Query().Get("limit")
	limit, err := strconv.ParseInt(s, 10, 64)
	if err != nil || limit < 0 {
		return int64(100)
	}
	return limit
}

func getLastBlock(r *http.Request, result interface{}) {
	t := r.URL.Query().Get("sort")
	time, err := time.Parse(time.RFC3339Nano, t)
	if err == nil {
		fields := model.GetFieldsTaggedWith(result, "sort")
		model.SetFieldValue(result, fields[0], &time)
	}
	fields := model.GetFieldsTaggedWith(result, "dynamodb_partition")
	if len(fields) > 0 {
		partition := r.URL.Query().Get(fields[0].Name)
		model.SetFieldValue(result, fields[0], partition)
	}
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

func makeHandler(handler func(http.ResponseWriter, *http.Request, store.Store), store store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handler(w, r, store)
	}
}

func auditHandler(w http.ResponseWriter, r *http.Request, store store.Store) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		common.LogError(r.Context(), "Wrong method: %v", r.Method)
		errorDefaultResponse(w, http.StatusMethodNotAllowed)
		return
	}
	common.LogInfo(r.Context(), "Start")
	if r.Method == http.MethodGet {
		auditGetHandler(w, r, store)
	}
	if r.Method == http.MethodPost {
		auditPostHandler(w, r, store)
	}
}

func auditGetHandler(w http.ResponseWriter, r *http.Request, store store.Store) {
	limit := getLimit(r)

	lastBlock := &model.Block{}
	getLastBlock(r, lastBlock)

	audit := []model.Block{}
	err := store.Read(&audit, limit, lastBlock)
	if err != nil {
		errorInternalServerErrorResponse(w, err)
		return
	}

	jsonResponse(w, audit)
}

func auditPostHandler(w http.ResponseWriter, r *http.Request, store store.Store) {
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

	err = store.Save(block)
	if err != nil {
		errorInternalServerErrorResponse(w, err)
		return
	}

	hashField := model.GetFieldsTaggedWith(block, "hash")
	hash := model.GetFieldStringValue(block, hashField[0])

	previousHashField := model.GetFieldsTaggedWith(block, "previoushash")
	previousHash := model.GetFieldStringValue(block, previousHashField[0])

	okResponseWithMessage(w, hash, previousHash)
}

func registerHandlers(store store.Store) *http.ServeMux {
	router := http.NewServeMux()
	router.Handle("/", http.NotFoundHandler())
	router.Handle("/audit", makeHandler(auditHandler, store))
	return router
}

// Start starts simple Auditor API
func Start(store store.Store) (*http.Server, error) {
	log.Printf("INFO auditor starting on port %s", defaultPort)

	router := registerHandlers(store)

	server := &http.Server{
		Addr:    ":" + defaultPort,
		Handler: tracing(router),
	}

	err := server.ListenAndServe()

	return server, err
}
