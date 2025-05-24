    # Fix SSEProxyHandler Path and Query Handling

    This plan addresses a bug in `SSEProxyHandler` where the request path and query string are not correctly separated when proxying requests, leading to test failures.

    ## Completed Tasks
    
    - [ ] None

    ## In Progress Tasks

    - [ ] Task 1: Analyze `SSEProxyHandler` and `TestSSEProxyHandler_MCPProtocolFlow` to confirm the root cause of the 404 error. `analysis`
        - Status: Completed during planning phase. The root cause is identified as `c.Request.URL.Path` being set to the full `action` string including query parameters.

    ## Future Tasks

    - [ ] Task 2: Modify `SSEProxyHandler` in `backend/api/handler/proxy_handler.go`. `bug-fix`
        - Description:
            - The `action := c.Param("action")` currently captures the path segment including any query string (e.g., "/message?sessionId=test").
            - This `action` string needs to be parsed.
            - The part of `action` before '?' should be set to `c.Request.URL.Path`.
            - The part of `action` after '?' (if it exists) should be set to `c.Request.URL.RawQuery`.
            - Use `strings.SplitN(action, "?", 2)` or `net/url.ParseRequestURI` (after prefixing with a dummy scheme if necessary, though `SplitN` is simpler here as we already have the path component from `action`).
        - Success Criteria: The `SSEProxyHandler` correctly forwards the path and query string separately to the underlying `sseSvc`.

    - [ ] Task 3: Run existing unit tests to verify the fix. `test`
        - Description: Execute `go test ./backend/api/handler/...` or specifically `TestSSEProxyHandler_MCPProtocolFlow`.
        - Success Criteria: `TestSSEProxyHandler_MCPProtocolFlow` and all other relevant tests pass. The 404 error is resolved, and the POST request receives a 202 with the correct content type and body.

    - [ ] Task 4: (Optional) Add new test cases if current coverage is insufficient. `test`
        - Description: Consider adding a specific test case for `SSEProxyHandler` that proxies to an endpoint *without* query parameters, and one with multiple query parameters, to ensure robustness, if not already covered.
        - Success Criteria: New tests pass and provide additional confidence in the fix.

    ## Implementation Plan

    The core change will be in `backend/api/handler/proxy_handler.go` within the `SSEProxyHandler` function.

    Current problematic section:
    ```go
    // ...
    action := c.Param("action")
    // ...
    // (logic to prepend "/" if missing)
    // ...
    c.Request.URL.Path = action // This is the line to change
    sseSvc.ServeHTTP(c.Writer, c.Request)
    ```

    Proposed change:
    ```go
    // ...
    rawAction := c.Param("action") // e.g., "/message?sessionId=test" or "/justpath"

    // Prepend "/" if action is not empty and doesn't start with "/"
    // This existing logic primarily handles cases like "action" vs "/action"
    // if the *action param could capture something without a leading slash.
    // Given that action is /api/sse/:serviceName/*action, action will always start with / or be empty.
    // Let's assume it always starts with / if not empty, or is empty (becomes "/" later).
    // The main concern is the query string.

    var pathPart string
    var queryPart string

    if queryIndex := strings.Index(rawAction, "?"); queryIndex != -1 {
        pathPart = rawAction[:queryIndex]
        queryPart = rawAction[queryIndex+1:]
    } else {
        pathPart = rawAction
    }
    
    // Ensure pathPart logic from existing code is applied
	if pathPart == "" {
		pathPart = "/" 
	}
	if pathPart != "" && !strings.HasPrefix(pathPart, "/") { // Should not happen if action is from /*foo
		pathPart = "/" + pathPart
	}

    c.Request.URL.Path = pathPart
    c.Request.URL.RawQuery = queryPart // Set RawQuery

    sseSvc.ServeHTTP(c.Writer, c.Request)
    ```
    This parsing logic correctly separates the path and query.

    ### Relevant Files

    - `backend/api/handler/proxy_handler.go` - To be modified.
    - `backend/api/handler/proxy_handler_test.go` - To verify the fix.