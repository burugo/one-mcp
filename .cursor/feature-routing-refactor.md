# Frontend Routing and Structure Refactor

This task involves refactoring the frontend application to use `react-router-dom` for navigation and to organize page-specific code into separate files under a `pages` directory. This will improve code organization, maintainability, and enable URL-based navigation.

## In Progress Tasks

- [x] **Install `react-router-dom`**: Install the `react-router-dom` library using `cnpm`. `Task Type: ref-struct`
- [x] **Create `pages` Directory Structure**: Create a `frontend/src/pages` directory. `Task Type: ref-struct`
- [ ] **Create Page Components**:
    - [x] Create `frontend/src/pages/DashboardPage.tsx`. `Task Type: ref-struct`
    - [x] Create `frontend/src/pages/ServicesPage.tsx`. `Task Type: ref-struct`
    - [x] Create `frontend/src/pages/MarketPage.tsx` (move and adapt existing `MarketPage` component). `Task Type: ref-struct`
    - [x] Create `frontend/src/pages/AnalyticsPage.tsx`. `Task Type: ref-struct`
    - [x] Create `frontend/src/pages/ProfilePage.tsx`. `Task Type: ref-struct`
    - [x] Create `frontend/src/pages/PreferencesPage.tsx`. `Task Type: ref-struct`
- [ ] **Migrate Page Content & Update Page Logic**:
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
- [ ] **Testing**: Verify all routes and navigation links work as expected. `Task Type: ref-struct`

## Future Tasks

- [ ] None at this moment.

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
- `frontend/src/pages/DashboardPage.tsx`
- `frontend/src/pages/MarketPage.tsx`
- `frontend/src/pages/AnalyticsPage.tsx`
- `frontend/src/pages/ProfilePage.tsx`
- `frontend/src/pages/PreferencesPage.tsx`
- `frontend/src/pages/` - New directory for page components. ✅ 