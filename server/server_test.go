package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/lukaszbudnik/auditor/model"
	"github.com/lukaszbudnik/auditor/store"
	"github.com/lukaszbudnik/migrator/common"
	"github.com/stretchr/testify/assert"
)

type errReader int

func (errReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("trouble maker")
}

func newTestRequest(method, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return req, err
	}
	ctx := req.Context()
	ctx = context.WithValue(ctx, common.RequestIDKey{}, "123")
	action := fmt.Sprintf("%v %v", method, strings.Replace(url, "http://example.com", "", -1))
	ctx = context.WithValue(ctx, common.ActionKey{}, action)
	return req.WithContext(ctx), err
}

func newMockStore() store.Store {
	return newMockStoreWithError(-1)()
}

func newMockStoreWithAudit(audit []model.Block) func() store.Store {
	return newMockStoreWithErrorAndAudit(-1, audit)
}

func newMockStoreWithError(threshold int) func() store.Store {
	return newMockStoreWithErrorAndAudit(threshold, []model.Block{})
}

func newMockStoreWithErrorAndAudit(threshold int, audit []model.Block) func() store.Store {
	return func() store.Store {
		return &mockStore{errorThreshold: threshold, counter: 1, audit: audit}
	}
}

func newJSONInput() *bytes.Buffer {
	time := time.Now().Format(time.RFC3339Nano)
	input := fmt.Sprintf(`{"Event": "new event", "Timestamp": "%v"}`, time)
	return bytes.NewBufferString(input)
}

func TestGetLimit(t *testing.T) {
	request, err := newTestRequest(http.MethodGet, "http://example.com/?limit=1234", nil)
	assert.Nil(t, err)

	limit := getLimit(request)
	assert.Equal(t, int64(1234), limit)
}

func TestGetLimitError(t *testing.T) {
	invalid := []string{"asdad", "-12323", "213.213", ""}
	for _, i := range invalid {
		request, err := newTestRequest(http.MethodGet, fmt.Sprintf("http://example.com/?limit=%v", i), nil)
		assert.Nil(t, err)

		limit := getLimit(request)
		// should always default to 100
		assert.Equal(t, int64(100), limit)
	}
}

func TestGetLastBlock(t *testing.T) {
	request, err := newTestRequest(http.MethodGet, "http://example.com/?sort=2019-01-01T12:39:01.999999999%2B01:00", nil)
	assert.Nil(t, err)
	lastBlock := &model.Block{}
	getLastBlock(request, lastBlock)
	assert.Equal(t, "2019-01-01 11:39:01 +0000 UTC", lastBlock.Timestamp.UTC().Truncate(time.Second).String())
}

func TestGetLastBlockWithDynamodbPartition(t *testing.T) {
	request, err := newTestRequest(http.MethodGet, "http://example.com/?sort=2019-01-01T12:39:01.999999999%2B01:00&Customer=abc", nil)
	assert.Nil(t, err)
	lastBlock := &model.Block{}
	getLastBlock(request, lastBlock)
	assert.Equal(t, "2019-01-01 11:39:01 +0000 UTC", lastBlock.Timestamp.UTC().Truncate(time.Second).String())
	assert.Equal(t, "abc", lastBlock.Customer)
}

func TestGetLastBlockError(t *testing.T) {
	invalid := []string{"asdad", ""}
	for _, i := range invalid {
		request, err := newTestRequest(http.MethodGet, fmt.Sprintf("http://example.com/?sort=%v", i), nil)
		assert.Nil(t, err)

		lastBlock := &model.Block{}
		getLastBlock(request, lastBlock)
		assert.Nil(t, lastBlock.Timestamp)
	}
}

func TestRegisterHandlers(t *testing.T) {
	mockStore := newMockStore()
	router := registerHandlers(mockStore)
	assert.NotNil(t, router)
}

func TestTracing(t *testing.T) {
	r, _ := newTestRequest(http.MethodGet, "http://example.com/sdsdf", nil)

	w := httptest.NewRecorder()
	handler := tracing(http.NotFoundHandler())
	handler.ServeHTTP(w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAuditMethodNotAllowed(t *testing.T) {
	httpMethods := []string{http.MethodHead, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodConnect, http.MethodOptions, http.MethodTrace}

	for _, httpMethod := range httpMethods {
		req, _ := newTestRequest(httpMethod, "http://example.com/audit", nil)

		w := httptest.NewRecorder()
		handler := makeHandler(auditHandler, newMockStore())
		handler(w, req)

		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	}

}

func TestAuditGet(t *testing.T) {
	time, _ := time.Parse(time.RFC3339Nano, "2019-01-03T08:09:09.611985+01:00")
	audit := []model.Block{}
	audit = append(audit, model.Block{Customer: "a", Timestamp: &time, Event: "some event", Category: "cat", Subcategory: "subcat", Hash: "1234567890abcdef", PreviousHash: "0987654321xyzghj"})
	handler := makeHandler(auditHandler, newMockStoreWithAudit(audit)())

	req, _ := newTestRequest(http.MethodGet, "http://example.com/audit", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.HeaderMap["Content-Type"][0])
	assert.Equal(t, `[{"Customer":"a","Timestamp":"2019-01-03T08:09:09.611985+01:00","Category":"cat","Subcategory":"subcat","Event":"some event","Hash":"1234567890abcdef","PreviousHash":"0987654321xyzghj"}]`, strings.TrimSpace(w.Body.String()))
}

func TestAuditGetReadError(t *testing.T) {
	handler := makeHandler(auditHandler, newMockStoreWithError(1)())

	req, _ := newTestRequest(http.MethodGet, "http://example.com/audit", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "application/json", w.HeaderMap["Content-Type"][0])
	assert.Equal(t, `{"ErrorMessage":"Error 1"}`, strings.TrimSpace(w.Body.String()))
}

func TestAuditPost(t *testing.T) {
	json := newJSONInput()
	req, _ := newTestRequest(http.MethodPost, "http://example.com/audit", json)

	w := httptest.NewRecorder()
	handler := makeHandler(auditHandler, newMockStore())
	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.HeaderMap["Content-Type"][0])
}

func TestAuditPostPreviousHash(t *testing.T) {
	audit := []model.Block{}
	audit = append(audit, model.Block{Hash: "1234567890abcdef"})
	handler := makeHandler(auditHandler, newMockStoreWithAudit(audit)())

	json := newJSONInput()
	req, _ := newTestRequest(http.MethodPost, "http://example.com/audit", json)
	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.HeaderMap["Content-Type"][0])
	assert.Contains(t, strings.TrimSpace(w.Body.String()), `"PreviousHash":"1234567890abcdef"`)
}

func TestAuditPostRequestIOError(t *testing.T) {
	req, _ := newTestRequest(http.MethodPost, "http://example.com/audit", errReader(0))

	w := httptest.NewRecorder()
	handler := makeHandler(auditHandler, newMockStore())
	handler(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "application/json", w.HeaderMap["Content-Type"][0])
	assert.Equal(t, `{"ErrorMessage":"trouble maker"}`, strings.TrimSpace(w.Body.String()))
}

func TestAuditPostJSONError(t *testing.T) {
	json := `{"event"]`
	req, _ := newTestRequest(http.MethodPost, "http://example.com/audit", bytes.NewBufferString(json))

	w := httptest.NewRecorder()
	handler := makeHandler(auditHandler, newMockStore())
	handler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "application/json", w.HeaderMap["Content-Type"][0])
	assert.Equal(t, `{"ErrorMessage":"invalid character ']' after object key"}`, strings.TrimSpace(w.Body.String()))
}

func TestAuditPostValidationError(t *testing.T) {
	// Timestamp is required
	json := fmt.Sprintf(`{"Event": "new event"}`)
	req, _ := newTestRequest(http.MethodPost, "http://example.com/audit", bytes.NewBufferString(json))

	w := httptest.NewRecorder()
	handler := makeHandler(auditHandler, newMockStore())
	handler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "application/json", w.HeaderMap["Content-Type"][0])
	assert.Equal(t, `{"ErrorMessage":"Timestamp: zero value"}`, strings.TrimSpace(w.Body.String()))
}

func TestAuditPostStoreReadError(t *testing.T) {
	json := newJSONInput()
	req, _ := newTestRequest(http.MethodPost, "http://example.com/audit", json)

	w := httptest.NewRecorder()
	handler := makeHandler(auditHandler, newMockStoreWithError(1)())
	handler(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "application/json", w.HeaderMap["Content-Type"][0])
	assert.Equal(t, `{"ErrorMessage":"Error 1"}`, strings.TrimSpace(w.Body.String()))
}

func TestAuditPostStoreSaveError(t *testing.T) {
	json := newJSONInput()
	req, _ := newTestRequest(http.MethodPost, "http://example.com/audit", json)

	w := httptest.NewRecorder()
	handler := makeHandler(auditHandler, newMockStoreWithError(1)())
	handler(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "application/json", w.HeaderMap["Content-Type"][0])
	assert.Equal(t, `{"ErrorMessage":"Error 1"}`, strings.TrimSpace(w.Body.String()))
}
