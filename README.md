# dsn

Written to derive DSN keys from originating -forwarded- requests for an on prem Sentry (8.13)
store endpoint /api/{projectID}/store/.

# Limitations:
1. Currently requests sent to the legacy /api/store/ endpoint will not have project ids. 
2. Module will currently not handle forwarded requests to the sentry API: /api/0/ 
3. Module does not rewrite auth headers.





    