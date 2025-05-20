# Login Refactor Tasks

Standardize the UI and logic for the standalone login page (`/login`) and the login modal, ensuring correct token persistence.

## Completed Tasks

- [x] **Task 1: Create a reusable login form component (`LoginFormCommon.tsx`)** `ref-struct`
    - **Objective**: Extract the login form UI (including social logins, email/password inputs, show/hide password, "Continue" button, "Sign up" link), state management (email, password, loading, showPassword), input handlers (`setEmail`, `setPassword`), and core `handleSubmit` logic (API call, `useAuth().login` call, toast notifications) from `frontend/src/components/ui/login-dialog.tsx` into a new common component `frontend/src/components/ui/LoginFormCommon.tsx`.
    - **Interface**: This component should accept `onSuccess: () => void` as a prop, to be called after successful login. It might also need an `isDialogMode?: boolean` prop for minor style/behavior adjustments if necessary.
    - **Key Logic Migration**: The `handleSubmit` function including `api.post('/auth/login', ...)` call, `authLogin(...)` call, `localStorage.setItem('refresh_token', ...)` call, and `toast(...)` calls.
    - **UI Element Migration**: Social login button group, "or" separator, Email input field, Password input field (with show/hide toggle), Continue button, and "Don't have an account? Sign up" link.
    - **Success Criteria**: `LoginFormCommon.tsx` contains the complete login form UI and core login logic, without the `Dialog` wrapper. Its behavior after success can be controlled via props.

- [x] **Task 2: Refactor the login dialog (`login-dialog.tsx`)** `ref-struct`
    - **Objective**: Modify `frontend/src/components/ui/login-dialog.tsx` to use the newly created `LoginFormCommon.tsx` component for rendering the form and handling login logic.
    - **Logic to Retain**: `LoginDialog` will primarily be responsible for the `Dialog` and `DialogContent` wrappers, and handling the `isOpen` and `onClose` props.
    - **Integration**: Render `<LoginFormCommon onSuccess={handleDialogSuccess} isDialogMode />` within `DialogContent`. The `handleDialogSuccess` function will call `onClose()` and `navigate('/')` (or just `onClose()`, letting the caller handle navigation).
    - **Success Criteria**: The login dialog functions identically to before, but its core form and login logic are provided by `LoginFormCommon.tsx`.

- [x] **Task 3: Refactor the standalone login page (`Login.tsx`)** `new-feat`
    - **Objective**: Modify `frontend/src/pages/Login.tsx`.
        - **Remove**: Its existing entire `<form>` UI and the `handleSubmit`, `handleInputChange` logic.
        - **Introduce**: Import and use the `useAuth` hook (implicitly used via `LoginFormCommon`).
        - **Integrate**: Render `<LoginFormCommon onSuccess={handlePageSuccess} />` on the page. The `handlePageSuccess` function should call `navigate('/')` (or the desired redirect path).
        - **Styling**: Ensure that when `LoginFormCommon` is rendered on the `Login.tsx` page, its appearance matches the form part of the modal shown in the screenshot (right side). `Login.tsx` might need to provide some outer containers and styles to center `LoginFormCommon` and make it look like a standalone login form, not a dialog section.
        - **Token**: Ensure that after successful login, via `authLogin` called within `LoginFormCommon`, the token and user information are correctly updated in `AuthContext` and `localStorage`.
    - **Success Criteria**: The `/login` page UI is identical to the login modal's form section. Login functionality is normal, Token and user state are correctly saved and globally synchronized, and the page redirects correctly.

- [x] **Task 4: Testing and Verification** `bug-fix`
    - **Objective**: Comprehensively test both login methods.
        - **Modal Login**: Check if functionality, UI, token handling, error messages, and successful redirection are consistent with the pre-refactor state.
        - **Standalone Page Login (`/login`)**: Check if UI is consistent with the modal form, and if login functionality, token handling, error messages, and successful redirection meet expectations.
        - **Logout and Re-login**: Test logging out and then logging back in through both methods, ensuring the state is correct.
        - **Page Refresh**: After logging in, refresh the page to check if the login state is persisted.
    - **Success Criteria**: All test scenarios pass, UI and functionality meet user requirements, and no regressions are found.

## In Progress Tasks

## Implementation Plan

1.  **Create `LoginFormCommon.tsx`**: Carefully extract UI and logic from `login-dialog.tsx`.
2.  **Modify `login-dialog.tsx`**: Import `LoginFormCommon` and adjust.
3.  **Modify `Login.tsx`**: Remove old code, import `LoginFormCommon`, ensure `useAuth` is available via context, and adjust page structure for the new component.
4.  **Comprehensive Testing**.

### Relevant Files

- `frontend/src/components/ui/login-dialog.tsx` (Source for common part, now refactored)
- `frontend/src/pages/Login.tsx` (Now refactored)
- `frontend/src/contexts/AuthContext.tsx` (Used by `LoginFormCommon` for `authLogin`)
- `frontend/src/components/ui/LoginFormCommon.tsx` (Created) 