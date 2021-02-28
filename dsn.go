package dsn

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

var (
	// ErrMissing User Thrown if we are missing the public key that comprises {PROTOCOL}://{PUBLIC_KEY}:{SECRET_KEY}@{HOST}{PATH}/{PROJECT_ID}
	ErrMissingUser = errors.New("sentry:  missing public key")
)

func FromRequest(r *http.Request) (*DSN, error) {
	/*
		Critical assumption here is that User information (sentry_key and optionally sentry_secret) will come from either
		request headers or the request query string. You will never use both to fill each of these values.

		We parse headers first to find User info. This will return pk, sk, both or err if no pk is found.
		If we err using headers we proceed to the QS. An Err here throws for the entire parse request operation.
		Returns the DSN struct which offers the original DSN with myDSN.originalDSN
	*/
	var user *User
	u := r.URL //represents a fully parsed url
	h := r.Header.Values("X-Sentry-Auth")

	host := u.Host

	// parse headers first
	usingHeader, err := parseHeaders(h)
	if err != nil {
		//try to gather DSN info from query string
		usingQs, qerr := parseQueryString(u)

		if qerr != nil {
			return nil, ErrMissingUser
		} else {
			user = usingQs
		}
	} else {
		user = usingHeader
	}
	// parse project
	p, err := checkPath(u)
	if err != nil {
		return nil, err
	}
	// complete DSN
	dsn := createDSN(user, host, p)

	return dsn, nil

}

type DSN struct {
	originalDSN string //original dsn for incoming request
}
type User struct {
	PublicKey string //public key for DSN
	SecretKey string //private key for DSN if necessary
}

//perhaps we can assume then that it will never be both
//we either are using q string or headers (mutually exclusive?)

func createDSN(d *User, host string, projectID string) *DSN {
	//Assumes either both keys are present or just public key. Other cases are caught earlier in processing incoming requests
	var myDSN string
	prefix := "https://"
	if len(d.PublicKey) > 0 && len(d.SecretKey) == 0 {
		myDSN = prefix + d.PublicKey + "@" + host + "/" + projectID

	} else if len(d.PublicKey) > 0 && len(d.SecretKey) > 0 {
		myDSN = prefix + d.PublicKey + ":" + d.SecretKey + "@" + host + "/" + projectID
	}
	return &DSN{originalDSN: myDSN}
}
func parseHeaders(h []string) (*User, error) {
	/*
		Parses values from X-SENTRY-AUTH header. Searches for both pk and sk values.
		Throws error if nothing is found for pk as this is critical.
		Returns user struct with appropriate values or empty strings.
	*/
	var sentryPublic string
	var sentrySecret string
	toArray := strings.Split(h[0], ",")

	for _, v := range toArray {
		fmt.Println(v)
		//should errors be thrown here?
		foundPublic, _ := regexp.MatchString(`sentry_key=([a-f0-9]{32})`, v)
		foundPrivate, _ := regexp.MatchString(`sentry_secret=([a-f0-9]{32})`, v)
		if foundPublic {
			sentryPublic = strings.Split(v, "=")[1]
			fmt.Println("HEADER:Match found PK")

		}
		if foundPrivate {
			sentrySecret = strings.Split(v, "=")[1]
			fmt.Println("HEADER:Match found Secret")

		}
	}
	if len(sentryPublic) == 0 {
		return nil, ErrMissingUser

	}
	return &User{PublicKey: sentryPublic, SecretKey: sentrySecret}, nil

}

func parseQueryString(u *url.URL) (*User, error) {
	/*
	   We need to check query string for DSN values as they may reside here and not in headers.
	   Looks for both pk and sk if applicable.
	   Throws if we are missing pk as this is critical.
	   Returns user struct with appropriate values or empty strings.
	*/

	pk := u.Query().Get("sentry_key")
	if len(pk) == 0 {
		return nil, ErrMissingUser
	}
	sk := u.Query().Get("sentry_secret")

	return &User{PublicKey: pk, SecretKey: sk}, nil

}

func checkPath(u *url.URL) (string, error) {
	/* assumes /api/<project_id>/store/
	the legacy /api/store endpoint does not include project id.
	This is usually where public key could be used to lookup project. Seems like a streth in this context.
	Currently this is handled by nothing and needs to be addressed
	We are acting optimistically here in terms of uri normalization.

	https://develop.sentry.dev/sdk/store
	*/
	path := u.Path
	isValid, _ := regexp.MatchString(`\/api\/\d+\/store\/`, path)

	if !isValid {
		return "", fmt.Errorf("Missing project ID. Attempted to parse project from: %s", path)

	}
	pathItems := strings.Split(path, "/")
	//with leading + trailing splits array has deterministic length of 5
	return pathItems[2], nil

}