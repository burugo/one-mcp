# Frontend Routing and Structure Refactor

This task involves refactoring the frontend application to use `react-router-dom` for navigation and to organize page-specific code into separate files under a `pages` directory. This will improve code organization, maintainability, and enable URL-based navigation.

## Completed Tasks

- [x] **Install `react-router-dom`**: Install the `react-router-dom` library using `cnpm`. `Task Type: ref-struct`
- [x] **Create `pages` Directory Structure**: Create a `frontend/src/pages` directory. `Task Type: ref-struct`
- [x] **Create Page Components**:
    - [x] Create `frontend/src/pages/DashboardPage.tsx`. `Task Type: ref-struct`
    - [x] Create `frontend/src/pages/ServicesPage.tsx`. `Task Type: ref-struct`
    - [x] Create `frontend/src/pages/MarketPage.tsx` (move and adapt existing `MarketPage` component). `Task Type: ref-struct`
    - [x] Create `frontend/src/pages/AnalyticsPage.tsx`. `Task Type: ref-struct`
    - [x] Create `frontend/src/pages/ProfilePage.tsx`. `Task Type: ref-struct`
    - [x] Create `frontend/src/pages/PreferencesPage.tsx`. `Task Type: ref-struct`
- [x] **Migrate Page Content & Update Page Logic**:
    - [x] `DashboardPage.tsx`: JSX migrated. Adapted to use `useNavigate` for navigation. `Task Type: ref-struct`
    - [x] `ServicesPage.tsx`: JSX migrated. Adapted to use router context for `setIsOpen` and `useNavigate` for navigation. `Task Type: ref-struct`
    - [x] `MarketPage.tsx`: JSX migrated. Import paths corrected. No further context/hook changes identified as necessary for this page. `Task Type: ref-struct`
    - [x] `AnalyticsPage.tsx`: Skeleton page adapted to connect to `PageOutletContext`. `Task Type: ref-struct`
    - [x] `ProfilePage.tsx`: Skeleton page adapted to connect to `PageOutletContext`. `Task Type: ref-struct`
    - [x] `PreferencesPage.tsx`: Skeleton page adapted to connect to `PageOutletContext`. `Task Type: ref-struct`
- [x] **Setup Routing in `App.tsx`**:
    - [x] Import necessary components from `react-router-dom` (`BrowserRouter`, `Routes`, `Route`, `Outlet`). `Task Type: ref-struct`
    - [x] Wrap the main layout in `<BrowserRouter>`. `Task Type: ref-struct`
    - [x] Define `<Routes>` within the main content area of `App.tsx`. `Task Type: ref-struct`
    - [x] Create a `<Route>` for each page component, specifying its path and element. `Task Type: ref-struct`
    - [x] Use `<Outlet />` in `App.tsx` where the page content should be rendered and pass context (`setIsOpen`). `Task Type: ref-struct`
- [x] **Update Navigation Links in `AppLayout`**:
    - [x] Sidebar and header navigation links in `AppLayout` (in `App.tsx`) use `react-router-dom`'s `<Link>` component and `useLocation` for active states. `Task Type: ref-struct`
- [x] **Remove `currentPage` State**: `currentPage` state and related logic fully removed from `App.tsx` and page components. `Task Type: ref-struct`
- [x] **Clean Up Unused Imports**: Remove unused imports in the refactored files to keep the code clean. `Task Type: ref-struct`
    - [x] Remove unused imports from `App.tsx`. `Task Type: ref-struct`
    - [x] Remove unused imports from `DashboardPage.tsx`. `Task Type: ref-struct`
    - [x] Check and clean other component files. `Task Type: ref-struct`
- [x] **Fix Services Page Tab Width Issue**: Add `w-full` to page containers and change `overflow-y-auto` to `overflow-y-scroll` on main content area. `Task Type: bug-fix`

## In Progress Tasks

- [ ] **Testing Verification**: `Task Type: ref-struct` (Moved to Future Tasks due to new UI bugs taking priority)
    - [x] Manual testing of navigation between pages. `Task Type: ref-struct`
    - [x] Set up Jest for frontend testing. `Task Type: ref-struct`
    - [ ] Write Jest tests for routing and component interactions. `Task Type: ref-struct` (Postponed)

## Future Tasks

- [ ] **Complete Jest Tests for Routing and Components**: Write Jest tests for routing and component interactions. `Task Type: ref-struct` (Postponed from In Progress)

## Implementation Plan

The refactoring will be done incrementally:
1. Install dependencies.
2. Create the new directory and files.
3. Migrate one page at a time and set up its route.
4. Update navigation for that page.
5. Test thoroughly after each page migration.

### Relevant Files

- `frontend/src/App.tsx` - Main layout and routing container. ✅
- `frontend/src/pages/ServicesPage.tsx` - Refactored to use router context/hooks. ✅
- `frontend/src/pages/DashboardPage.tsx` - Refactored to use navigation hooks. ✅
- `frontend/src/pages/MarketPage.tsx` - Import paths corrected. ✅
- `frontend/src/pages/AnalyticsPage.tsx` - Connected to router context. ✅
- `frontend/src/pages/ProfilePage.tsx` - Connected to router context. ✅
- `frontend/src/pages/PreferencesPage.tsx` - Connected to router context. ✅ 
- `frontend/src/components/market/ServiceMarketplace.tsx` - No changes needed. ✅
- `frontend/src/components/market/ServiceDetails.tsx` - No changes needed. ✅
- `frontend/src/store/marketStore.ts` - No changes needed. ✅ 