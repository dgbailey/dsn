# dsn

Written to derive DSN keys from[https://golang.org/pkg/net/http/#Request](requests) forwarded from an on prem Sentry (8.13) store endpoint /api/{projectID}/store/.

# implementation
```
import "github.com/dgbailey/dsn"

func myFunc(r *http.Request){
    //some request handler
	dsn, err := dsn.FromRequest(r)
	if err != nil {
		//handle err
	} else {
        //check dsn length + other logic
	}
}

```

# run tests

```go test --v```

# Limitations:
1. Currently requests sent to the legacy /api/store/ will return a DSN struct with URL as empty ""
2. Module will currently not handle forwarded requests to the sentry API: /api/0/ 
3. Module does not rewrite auth headers.





    