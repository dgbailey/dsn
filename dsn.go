package dsn

import (
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

var HTTP_X_SENTRY_AUTH = "X-SENTRY-AUTH"

var (
	// ErrMissing User Thrown if we are missing the public key that comprises {PROTOCOL}://{PUBLIC_KEY}:{SECRET_KEY}@{HOST}{PATH}/{PROJECT_ID}
	ErrMissingUser      = errors.New("sentry:  missing public key")
	ErrMissingProjectID = errors.New("sentry:  Failed attempt to parse project ID from path --")
)

type DSN struct {
	URL       string //original dsn for incoming request
	Host      string
	ProjectID string
	PublicKey string
	SecretKey string
}
type User struct {
	PublicKey string //public key for DSN
	SecretKey string //private key for DSN if necessary
}


func CreateDSN(d *User, host string, projectID string) *DSN {
	/*
	In the case where we encounter the legacy /api/store/ the returned DNS struct will have len(url) == 0
	This will allow for optional checks in case the other parts of the struct (publicKey) are used for projectID lookups
	Remaining conditions assume either both keys are present or just public key. 
	*/
	var url string
	prefix := "https://"
	if len(projectID) == 0{
		url = ""
	}else if len(d.PublicKey) > 0 && len(d.SecretKey) == 0 {
		url = prefix + d.PublicKey + "@" + host + "/" + projectID
	}else if len(d.PublicKey) > 0 && len(d.SecretKey) > 0 {
		url = prefix + d.PublicKey + ":" + d.SecretKey + "@" + host + "/" + projectID
	}
	
	return &DSN{URL: url, ProjectID: projectID, Host: host, PublicKey: d.PublicKey, SecretKey: d.SecretKey}
}
func ParseHeaders(h []string) (*User, error) {
	/*
		Parses values from X-SENTRY-AUTH header. Searches for both pk and sk values.
		Throws error if nothing is found for pk as this is critical.
		Returns user struct with appropriate values or empty strings.
	*/
	var sentryPublic string
	var sentrySecret string
	
	if len(h) == 0 {
		return nil, ErrMissingUser
	}
	
	toArray := strings.Split(strings.Split(h[0]," ")[1],",")
	//Anticipates header: Sentry <start-header-values,...>

	for _, v := range toArray {

		foundPublic, _ := regexp.MatchString(`sentry_key=([a-f0-9]{32})`, v)
		foundPrivate, _ := regexp.MatchString(`sentry_secret=([a-f0-9]{32})`, v)
		if foundPublic {
			sentryPublic = strings.Split(v, "=")[1]
		}
		if foundPrivate {
			sentrySecret = strings.Split(v, "=")[1]
		}
	}
	if len(sentryPublic) == 0 {
		return nil, ErrMissingUser

	}
	return &User{PublicKey: sentryPublic, SecretKey: sentrySecret}, nil

}

func ParseQueryString(u *url.URL) (*User, error) {
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

func CheckPath(u *url.URL) (string, error) {
	/* 
	Assumes /api/<project_id>/store/   OR    \/api\/store\/
	The legacy /api/store/ endpoint does not include project id.

	This is usually where public key could be used to lookup project in Relay. As we are not in relay this is not an option.
	Older clients tested:
		raven-python 5.27.0
		java Raven-Java 7.8.0-31c26
		javascript raven-js 3.10.0
	
	All of these clients utilize the  /api/<project_id>/store/  endpoint.
	Given the test have a higher degree of certainty that we will not encounter the legacy api.
	We currently throw below if we do.

	** Anticipates leading and trailing slashes **
	https://develop.sentry.dev/sdk/store
	*/
	path := u.Path
	isValid, _ := regexp.MatchString(`\/api\/\d+\/store\/`, path)
	isValidLegacy, _ := regexp.MatchString(`\/api\/store\/`, path)

	if !isValid {
		if isValidLegacy {
			return "", nil
		}
		return "", ErrMissingProjectID
	}
	pathItems := strings.Split(path, "/")

	//with leading + trailing splits array has deterministic length of 5
	return pathItems[2], nil

}
func FromRequest(r *http.Request) (*DSN, error) {
	/*
		Critical assumption here is that User information (sentry_key and optionally sentry_secret) will come from either
		request headers or the request query string. You will never use both to fill each of these values.

		We parse headers first to find User info. This will return pk, sk, both or err if no pk is found.
		If we err using headers we proceed to the QS. An Err here throws for the entire parse request operation.
		Returns the DSN struct which offers the original DSN with myDSN.URL
	*/
	var user *User
	u := r.URL //represents a fully parsed url
	h := r.Header.Values(HTTP_X_SENTRY_AUTH)

	host := u.Hostname()
	if len(host) == 0{
		host = r.Host
	}
	//some routers/proxies may strip the host from http.Request.URL so http.Request.Host is useful.

	
	usingHeader, err := ParseHeaders(h)
	if err != nil {
	
		usingQs, qerr := ParseQueryString(u)

		if qerr != nil {
			return nil, ErrMissingUser
		} else {
			user = usingQs
		}
	} else {
		user = usingHeader
	}
	// parse project
	p, err := CheckPath(u)
	if err != nil {
		return nil, err
	}
	// complete DSN
	dsn := CreateDSN(user, host, p)

	return dsn, nil

}
