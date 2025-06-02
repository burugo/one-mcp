# Custom Service Description Enhancement

## Background and Motivation
The current description for custom services in the UI (e.g., on service cards) is generic (e.g., "Custom sse service"). The user desires a more informative description that reflects the service's configuration:
- For `stdio` services: `command/args (stdio)`
- For `sse` or `http` services: `url (sse/http)`

This will provide users with a clearer at-a-glance understanding of each custom service's nature.

## Issue Analysis: Why Services Show "unknown" Status

**Root Cause**: Custom services created via `CreateCustomService` are not automatically registered with the `ServiceManager`, so they remain in "unknown" health status.

**Current Flow**:
1. `CreateCustomService` creates a new `MCPService` record in the database with `HealthStatus: "unknown"`
2. The service is **NOT** automatically registered with `ServiceManager`
3. `ServiceManager` is responsible for:
   - Creating service instances via `ServiceFactory`
   - Running health checks via `HealthChecker`
   - Updating database health status
4. Without registration, the service remains in "unknown" state forever

**Solution Options**:
1. **Auto-register on creation** (recommended): Modify `CreateCustomService` to register the service with `ServiceManager` after successful database creation
2. **Manual registration trigger**: Add an endpoint to manually register services
3. **Auto-discovery**: Modify `ServiceManager.Initialize()` to auto-register unregistered enabled services

## Key Challenges and Analysis
- The primary change will be in the backend, specifically in the `CreateCustomService` function within `backend/api/handler/market.go`, where the `Description` field of the `model.MCPService` object is populated.
- The frontend component `frontend/src/components/market/ServiceCard.tsx` already displays the `service.description` field, so no frontend changes are anticipated for the primary display, assuming the new description is well-formatted.
- Need to handle cases where `command`, `args`, or `url` might be empty or very long.
- The `Args` for stdio services are stored as a string in `model.MCPService.Args` and received as `requestBody.Arguments` (string). The display format needs to be concise.

## High-level Task Breakdown
1.  Modify backend logic to generate the new dynamic description.
2.  Test thoroughly with different service types and configurations.

## Project Status Board
- Project progress summary: Planning

## Completed Tasks
- [x] Backend description generation logic implemented `enhancement`
- [x] Frontend API URL fix for custom service creation `bug-fix`
- [x] Analysis of "unknown" status issue `investigation`

## In Progress Tasks
- [ ] Implement service auto-registration after custom service creation.

## Future Tasks
- [ ] Comprehensive testing of new descriptions and health status updates.

## Implementation Plan

### 1. Backend Modification (`backend/api/handler/market.go`)

**File**: `backend/api/handler/market.go`
**Function**: `CreateCustomService`

**Current Logic for Description**:
```go
newService := model.MCPService{
    // ...
    Description:           fmt.Sprintf("Custom %s service", requestBody.Type),
    // ...
}
```

**Proposed Change**:
Modify the logic for setting `newService.Description` based on `requestBody.Type`:

```go
var description string
serviceTypeForDisplay := strings.ToLower(string(requestBody.Type)) // e.g., "stdio", "sse"

switch requestBody.Type {
case model.ServiceTypeStdio:
    cmdDisplay := requestBody.Command
    if len(cmdDisplay) > 50 { // Truncate long commands
        cmdDisplay = cmdDisplay[:47] + "..."
    }
    argsDisplay := requestBody.Arguments
    if argsDisplay == "" {
        argsDisplay = "no args"
    } else if len(argsDisplay) > 30 { // Truncate long arguments string
        argsDisplay = argsDisplay[:27] + "..."
    }
    description = fmt.Sprintf("%s/%s (stdio)", cmdDisplay, argsDisplay)
case model.ServiceTypeSSE, model.ServiceTypeStreamableHTTP: // Assuming ServiceTypeStreamableHTTP is the correct model constant
    urlDisplay := requestBody.URL
    if len(urlDisplay) > 80 { // Truncate long URLs
        urlDisplay = urlDisplay[:77] + "..."
    }
    if urlDisplay == "" {
        description = fmt.Sprintf("URL not set (%s)", serviceTypeForDisplay)
    } else {
        description = fmt.Sprintf("%s (%s)", urlDisplay, serviceTypeForDisplay)
    }
default:
    // Fallback for any unknown types, though the previous type check should catch invalid ones.
    description = fmt.Sprintf("Custom service (%s)", serviceTypeForDisplay)
}

newService := model.MCPService{
    Name:                  requestBody.Name,
    DisplayName:           requestBody.Name,
    Description:           description, // Use the new dynamic description
    Category:              model.CategoryUtil,
    Type:                  serviceType, // This is already correctly set to model.ServiceType
    ClientConfigTemplates: "{}",
    Enabled:               true,
    HealthStatus:          "unknown",
}

// Ensure model.ServiceTypeStreamableHTTP is the correct constant.
// It's seen as model.ServiceTypeStreamableHTTP in codebase_search results (proxy/service.go)
// and as requestBody.Type == "streamableHttp" (lowercase 'h') in market.go input validation.
// The MCPService.Type field will hold the proper constant like model.ServiceTypeSSE.
// The requestBody.Type is a string 'sse', 'stdio', 'streamableHttp'.
```

**Detailed Steps**:
1.  Inside `CreateCustomService` in `backend/api/handler/market.go`, before initializing `newService`.
2.  Declare a `description string` variable.
3.  Use a `switch` statement on `requestBody.Type` (which is a string: "stdio", "sse", "streamableHttp").
    *   **Case "stdio"**:
        *   Get `requestBody.Command`. Truncate if longer than (e.g.) 50 characters.
        *   Get `requestBody.Arguments`. If empty, use a placeholder like "no args". Truncate if longer than (e.g.) 30 characters.
        *   Format as `fmt.Sprintf("%s/%s (stdio)", truncatedCommand, truncatedArgs)`.
    *   **Case "sse" and "streamableHttp"**:
        *   Get `requestBody.URL`. Truncate if longer than (e.g.) 80 characters.
        *   If URL is empty, format as `fmt.Sprintf("URL not set (%s)", requestBody.Type)`.
        *   Otherwise, format as `fmt.Sprintf("%s (%s)", truncatedURL, requestBody.Type)`.
    *   **Default case**: (As a fallback, though type validation should prevent this)
        *   `fmt.Sprintf("Custom service (%s)", requestBody.Type)`
4.  Assign the generated `description` to `newService.Description`.

**Considerations**:
-   The truncation lengths (50, 30, 80) are placeholders and can be adjusted for better UI fit.
-   Confirm the exact string values for `requestBody.Type` for "streamableHttp" (e.g. "streamableHttp" vs "streamableHTTP") and ensure the `model.ServiceTypeStreamableHTTP` constant is used correctly when comparing if needed (though `requestBody.Type` string comparison is fine here). `CreateCustomService` already validates `requestBody.Type` and converts it to `serviceType model.ServiceType`.

### 2. Fix "unknown" Status Issue

**Root Issue**: Services created by `CreateCustomService` are not automatically registered with `ServiceManager`.

**Solution**: Add service registration after successful database creation in `CreateCustomService`.

**Implementation**:
```go
// After successful model.CreateService(&newService)
serviceManager := proxy.GetServiceManager()
ctx := c.Request.Context()
if err := serviceManager.RegisterService(ctx, &newService); err != nil {
    // Log error but don't fail the API call since the service was created
    log.Printf("Warning: Failed to register custom service %s (ID: %d) with ServiceManager: %v", newService.Name, newService.ID, err)
    // Optionally return a warning in the response
} else {
    log.Printf("Successfully registered custom service %s (ID: %d) with ServiceManager", newService.Name, newService.ID)
}
```

**Benefits**:
- Service will be automatically health-checked
- Status will update from "unknown" to actual health state
- Service will be available for proxy operations

**Alternative**: Modify `ServiceManager.Initialize()` to auto-discover and register unregistered enabled services.

### 3. Testing
-   Create custom services of type `stdio` with:
    -   Short command, short args.
    -   Long command, short args.
    -   Short command, long args.
    -   Long command, long args.
    -   Command, no args.
    -   No command (if possible, though likely blocked by validation).
-   Create custom services of type `sse` with:
    -   Short URL.
    -   Long URL.
    -   No URL (if possible, though likely blocked by validation).
-   Create custom services of type `streamableHttp` with:
    -   Short URL.
    -   Long URL.
    -   No URL (if possible, though likely blocked by validation).
-   Verify the description displays correctly on the service cards in the UI.
-   Verify that if essential parts (like command for stdio, URL for sse/http) are missing (and somehow bypass validation), the description is still reasonable.
-   **Health Status Testing**: Verify that services transition from "unknown" to proper health status after registration.

### Relevant Files
-   `backend/api/handler/market.go` - Main file for modification.
-   `frontend/src/components/market/ServiceCard.tsx` - UI component that will display the new description.
-   `frontend/src/pages/ServicesPage.tsx` - Another page that displays service descriptions.
-   `backend/model/mcp_service.go` - Definition of `MCPService` struct.
-   `frontend/src/components/market/CustomServiceModal.tsx` - Modal for creating custom services, source of `requestBody`.
-   `backend/library/proxy/manager.go` - ServiceManager for service registration.
-   `backend/library/proxy/service.go` - ServiceFactory and health checking logic.

### 3. Frontend URL Fix (`frontend/src/pages/ServicesPage.tsx`)

**Issue**: The frontend was making API requests with duplicated `/api` prefix, causing 404 errors.

**Root Cause**: 
- `frontend/src/utils/api.ts` configures Axios with `baseURL: '/api'`
- `ServicesPage.tsx` was calling `api.post('/api/mcp_market/custom_service', ...)` 
- This resulted in URLs like `/api/api/mcp_market/custom_service`

**Fix Applied**:
Changed API calls in `frontend/src/pages/ServicesPage.tsx`:
```typescript
// Before:
res = await api.post('/api/mcp_market/custom_service', serviceData) as APIResponse<any>;
res = await api.post('/api/mcp_market/install_or_add_service', payload) as APIResponse<any>;

// After:
res = await api.post('/mcp_market/custom_service', serviceData) as APIResponse<any>;
res = await api.post('/mcp_market/install_or_add_service', payload) as APIResponse<any>;
```

**Status**: ✅ Completed

## ACT mode Feedback or Assistance Requests
**Issue Identified**: Custom services remain in "unknown" status because they are not automatically registered with ServiceManager after creation. The CreateCustomService function only creates the database record but doesn't register the service for health checking.

**Recommended Solution**: Add ServiceManager registration after successful database creation in CreateCustomService function. This will ensure services transition from "unknown" to proper health status automatically. 