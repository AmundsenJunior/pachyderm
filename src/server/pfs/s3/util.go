package s3

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// The S3 storage class that all PFS content will be reported to be stored in
const storageClass = "STANDARD"

// The S3 user associated with all PFS content
var defaultUser = User{ID: "00000000000000000000000000000000", DisplayName: "pachyderm"}

// User is an XML-encodable representation of an S3 user
type User struct {
	ID          string `xml:"ID"`
	DisplayName string `xml:"DisplayName"`
}

// writeXML serializes a struct to a response as XML
func writeXML(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(code)
	encoder := xml.NewEncoder(w)
	if err := encoder.Encode(v); err != nil {
		// just log a message since a status code - and maybe part of
		logrus.Errorf("s3gateway: could not enocde xml response: %v", err)
	}
}

// intFormValue extracts an int value from a request's form values, ensuring
// it's within specified bounds. If the value is not specified, is not an int,
// or is not within the specified bounds, it defaults to `def`.
func intFormValue(r *http.Request, name string, min int, max int, def int) int {
	s := r.FormValue(name)
	if s == "" {
		return def
	}

	i, err := strconv.Atoi(s)
	if err != nil || i < min || i > max {
		return def
	}
	return i
}

// withBodyReader calls the provided callback with a reader for the HTTP
// request body. This also verifies the body against the `Content-MD5` header.
//
// The callback should return whether or not it succeeded. If it does not
// succeed, it is assumed that the callback wrote an appropriate failure
// response to the client.
//
// This function will return whether it succeeded.
func withBodyReader(w http.ResponseWriter, r *http.Request, f func(io.Reader) bool) bool {
	expectedHash := r.Header.Get("Content-MD5")

	if expectedHash != "" {
		expectedHashBytes, err := base64.StdEncoding.DecodeString(expectedHash)
		if err != nil || len(expectedHashBytes) != 16 {
			invalidDigestError(w, r)
			return false
		}

		hasher := md5.New()
		reader := io.TeeReader(r.Body, hasher)

		succeeded := f(reader)
		if !succeeded {
			return false
		}

		actualHash := hasher.Sum(nil)
		if !bytes.Equal(expectedHashBytes, actualHash) {
			badDigestError(w, r)
			return false
		}

		w.WriteHeader(http.StatusOK)
		return true
	}

	succeeded := f(r.Body)
	if succeeded {
		w.WriteHeader(http.StatusOK)
	}
	return true
}

func bucketArgs(w http.ResponseWriter, r *http.Request) (string, string, bool) {
	vars := mux.Vars(r)
	bucket := vars["bucket"]
	lastDashIndex := strings.LastIndex(bucket, "-")

	if lastDashIndex >= 0 {
		repo := bucket[:lastDashIndex]
		branch := bucket[lastDashIndex+1:]

		if branch == "master" {
			 parts := strings.SplitN(r.URL.Path, "/", 3)
			 filepath := parts[2]
			 w.Header().Set("Location", fmt.Sprintf("/%s/%s", repo, filepath))
			 permanentRedirectError(w, r)
			 return repo, branch, false
		}

		return repo, branch, true
	}
		
	return bucket, "master", true
}

func objectArgs(w http.ResponseWriter, r *http.Request) (string, string, string, bool) {
	vars := mux.Vars(r)
	file := vars["file"]
	repo, branch, ok := bucketArgs(w, r)
	return repo, branch, file, ok
}