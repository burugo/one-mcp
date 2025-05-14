# UI Polish and Bug Fixing

This file tracks tasks related to fixing UI bugs and implementing minor UI enhancements discovered after the main routing refactor and successful login implementation.

## Completed Tasks

- [x] 修复 AppContent 测试因多个 'One MCP' 元素导致的断言报错（test 修正） `bug-fix`

## In Progress Tasks

- [ ] **Fix Login Navigation Update Issue**: `bug-fix`
    - **Description**: After a user successfully logs in, the top navigation bar (specifically the areaManaged by `AppLayout` in `App.tsx`) does not update to reflect the logged-in state. It continues to show the "Login" button instead of, for example, a user avatar and a logout option.
    - **Analysis**: This likely involves checking how the authentication state (e.g., presence of a token, user data from context or store) is being used to conditionally render the navigation elements in `AppLayout`. The logic might be missing, incorrect, or not re-rendering upon state change.
    - **Success Criteria**: After login, the "Login" button is replaced with appropriate UI elements for a logged-in user (e.g., user avatar/dropdown, logout button).

- [ ] **Fix Theme Switching Style Inconsistencies**: `bug-fix`
    - **Description**: When the operating system is set to dark mode, and the website theme is manually switched to light mode (or vice-versa), there are visual inconsistencies. For example, some elements might retain dark theme styles while the overall page is light, or vice-versa, leading to poor contrast and a disjointed look as seen in the screenshot (e.g., cards or backgrounds).
    - **Analysis**: This could be due to:
        - Incorrect application or overriding of theme-specific CSS variables or Tailwind classes.
        - CSS specificity issues where OS-level preferences (via `prefers-color-scheme`) are unintentionally taking precedence over the manually selected application theme.
        - Shadcn UI theme provider (`ThemeProvider`) or a custom theme switching logic not correctly toggling all relevant classes or variables on the `html` or `body` elements, or on specific components.
    - **Success Criteria**: The UI consistently reflects the selected application theme (light or dark) across all components, regardless of the OS theme preference. No mixed-theme elements should be visible.

## Future Tasks

- [ ] (Placeholder for future UI enhancements)

## Implementation Plan

1.  **Login Navigation Update**:
    *   Inspect `AppLayout` in `App.tsx` and how it determines whether to show the "Login" button or logged-in user UI.
    *   Verify that the authentication state (e.g., from `AuthContext`, Zustand store, or `localStorage` token check) is correctly accessed and causes a re-render.
    *   Implement the correct conditional rendering logic for the navigation bar.
2.  **Theme Switching Styles**:
    *   Review the `ThemeProvider` setup (likely from Shadcn UI) and how theme classes (`.dark`, `.light`) are applied to the root HTML element.
    *   Inspect the CSS/Tailwind classes of the problematic components seen in the screenshot (e.g., cards, backgrounds, text) in both theme states, especially when OS and app themes differ.
    *   Ensure that theme-specific styles are correctly defined and applied, and that OS `prefers-color-scheme` is either respected or correctly overridden by the application's theme choice.

### Relevant Files

- `frontend/src/App.tsx` (for `AppLayout` and potentially theme provider context)
- `frontend/src/components/ui/login-dialog.tsx` (for login success logic that might trigger state updates)
- `frontend/src/contexts/AuthContext.tsx` (or equivalent auth state management)
- `frontend/src/components/ThemeProvider.tsx` (or wherever theme switching logic resides)
- `frontend/src/index.css` (for global styles and theme variables)
- Potentially affected page components like `frontend/src/pages/DashboardPage.tsx` or `frontend/src/pages/ServicesPage.tsx` (for theme-related styling issues on specific components). 