import React, { useEffect } from 'react'
import { BrowserRouter, Routes, Route, Link, Outlet, useLocation, Navigate } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Toaster } from '@/components/ui/toaster'
import { useToast } from '@/hooks/use-toast'
import { Settings, User, Home, BarChart, Globe, Package, Users, FileText } from 'lucide-react'
import { LoginDialog } from './components/ui/login-dialog'
import { ThemeToggle } from './components/ui/theme-toggle'
import { MarketPage } from './pages/MarketPage'
import { DashboardPage } from './pages/DashboardPage'
import { ServicesPage } from './pages/ServicesPage'
import { AnalyticsPage } from './pages/AnalyticsPage'
import { ProfilePage } from './pages/ProfilePage'
import { PreferencesPage } from './pages/PreferencesPage'
import { UsersPage } from './pages/UsersPage'
import LogsPage from './pages/LogsPage'
import Login from '@/pages/Login'
import { OAuthCallback } from './pages/OAuthCallback'
import { toastEmitter } from '@/utils/api'
import { AuthProvider, useAuth } from './contexts/AuthContext'
import { useTranslation } from 'react-i18next'
import { LanguageSwitcher } from './components/ui/language-switcher'

// Props that might be passed down from AppLayout to pages via Outlet context
export interface PageOutletContext {
  setIsOpen: React.Dispatch<React.SetStateAction<boolean>>
}

const AppLayout = () => {
  const { toast } = useToast()
  const [isOpen, setIsOpen] = React.useState(false)
  const [showLoginDialog, setShowLoginDialog] = React.useState(false)
  const location = useLocation()
  const { currentUser, logout, isLoading } = useAuth()
  const { t } = useTranslation()
  // const navigate = useNavigate() // 暂时未使用

  const NavLink = ({ to, children, isTopNav }: { to: string, children: React.ReactNode, isTopNav?: boolean }) => {
    const isActive = location.pathname === to || (to === '/' && location.pathname === '/dashboard')

    let className = "text-sm font-medium transition-colors duration-200 "
    if (isTopNav) {
      className += isActive ? "text-primary" : "text-muted-foreground hover:text-foreground"
    } else {
      className += `flex items-center gap-3 px-4 py-2.5 rounded-md ${isActive ? 'bg-primary/10 text-primary font-medium' : 'hover:bg-muted/50 text-muted-foreground hover:text-foreground'}`
    }

    return (
      <Link to={to} className={className}>
        {children}
      </Link>
    )
  }

  // Handle logout navigation
  const handleLogout = () => {
    logout()
  }



  if (isLoading) {
    return <div>Loading application...</div>
  }

  return (
    <div className="min-h-screen flex flex-col bg-background text-foreground">
      <header className="border-b border-border bg-background sticky top-0 z-10">
        <div className="container mx-auto flex items-center h-16 px-6">
          <Link to="/" className="flex items-center gap-2">
            <Globe className="h-6 w-6 text-primary" />
            <h1 className="text-xl font-semibold">One MCP</h1>
          </Link>
          <div className="flex items-center ml-auto gap-6">
            <nav className="hidden md:flex items-center gap-6">
              <NavLink to="/" isTopNav>{t('nav.dashboard')}</NavLink>
              <NavLink to="/docs" isTopNav>{t('nav.docs')}</NavLink>
            </nav>
            <LanguageSwitcher />
            <ThemeToggle />
            {currentUser ? (
              <div className="flex items-center gap-2">
                <span className="text-sm">{currentUser.display_name || currentUser.username}</span>
                <Button variant="outline" size="sm" onClick={handleLogout}>{t('auth.logoutButton')}</Button>
              </div>
            ) : (
              <Button
                size="sm"
                className="rounded-full transition-all duration-200 hover:opacity-90 bg-[#7c3aed] hover:bg-[#7c3aed]/90"
                onClick={() => setShowLoginDialog(true)}
              >
                {t('auth.loginButton')}
              </Button>
            )}
          </div>
        </div>
      </header>
      <div className="flex flex-1 overflow-hidden">
        <aside className="w-64 border-r border-border hidden md:block overflow-y-auto flex-shrink-0 bg-background/80">
          <nav className="p-4 space-y-1">
            <NavLink to="/">
              <Home className={`h-4 w-4 ${location.pathname === '/' || location.pathname.startsWith('/dashboard') ? 'text-primary' : 'text-muted-foreground'}`} />
              <span className={`${location.pathname === '/' || location.pathname.startsWith('/dashboard') ? 'text-primary' : 'text-muted-foreground'}`}>{t('nav.dashboard')}</span>
            </NavLink>
            <NavLink to="/services">
              <Globe className={`h-4 w-4 ${location.pathname.startsWith('/services') ? 'text-primary' : 'text-muted-foreground'}`} />
              <span className={`${location.pathname.startsWith('/services') ? 'text-primary' : 'text-muted-foreground'}`}>{t('nav.services')}</span>
            </NavLink>
            {currentUser && currentUser.role && currentUser.role >= 10 && (
              <NavLink to="/market">
                <Package className={`h-4 w-4 ${location.pathname.startsWith('/market') ? 'text-primary' : 'text-muted-foreground'}`} />
                <span className={`${location.pathname.startsWith('/market') ? 'text-primary' : 'text-muted-foreground'}`}>{t('nav.market')}</span>
              </NavLink>
            )}
            <NavLink to="/analytics">
              <BarChart className={`h-4 w-4 ${location.pathname.startsWith('/analytics') ? 'text-primary' : 'text-muted-foreground'}`} />
              <span className={`${location.pathname.startsWith('/analytics') ? 'text-primary' : 'text-muted-foreground'}`}>{t('nav.analytics')}</span>
            </NavLink>
            {/* 用户管理 - 仅管理员可见 */}
            {currentUser && currentUser.role && currentUser.role >= 10 && (
              <NavLink to="/users">
                <Users className={`h-4 w-4 ${location.pathname.startsWith('/users') ? 'text-primary' : 'text-muted-foreground'}`} />
                <span className={`${location.pathname.startsWith('/users') ? 'text-primary' : 'text-muted-foreground'}`}>{t('nav.users')}</span>
              </NavLink>
            )}
            {/* 日志查看 - 仅管理员可见 */}
            {currentUser && currentUser.role && currentUser.role >= 10 && (
              <NavLink to="/logs">
                <FileText className={`h-4 w-4 ${location.pathname.startsWith('/logs') ? 'text-primary' : 'text-muted-foreground'}`} />
                <span className={`${location.pathname.startsWith('/logs') ? 'text-primary' : 'text-muted-foreground'}`}>{t('nav.logs')}</span>
              </NavLink>
            )}
            <div className="my-4 border-t border-border"></div>
            <NavLink to="/profile">
              <User className={`h-4 w-4 ${location.pathname.startsWith('/profile') ? 'text-primary' : 'text-muted-foreground'}`} />
              <span className={`${location.pathname.startsWith('/profile') ? 'text-primary' : 'text-muted-foreground'}`}>{t('nav.profile')}</span>
            </NavLink>
            {currentUser && currentUser.role && currentUser.role >= 10 && (
              <NavLink to="/preferences">
                <Settings className={`h-4 w-4 ${location.pathname.startsWith('/preferences') ? 'text-primary' : 'text-muted-foreground'}`} />
                <span className={`${location.pathname.startsWith('/preferences') ? 'text-primary' : 'text-muted-foreground'}`}>{t('nav.preferences')}</span>
              </NavLink>
            )}
          </nav>
        </aside>
        <main className="flex-1 p-6 overflow-y-auto h-[calc(100vh-64px)] bg-background/50">
          <div className="container mx-auto">
            <Outlet context={{ setIsOpen }} />
          </div>
        </main>
      </div>
      <Dialog open={isOpen} onOpenChange={setIsOpen}>
        <DialogContent className="sm:max-w-[425px]">
          <DialogHeader>
            <DialogTitle>Service Configuration</DialogTitle>
            <DialogDescription>
              Adjust the settings for this service. Click save when you're done.
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid grid-cols-4 items-center gap-4">
              <label htmlFor="api-key" className="text-right text-sm font-medium col-span-1">
                API Key
              </label>
              <Input id="api-key" value="••••••••••••••••" className="col-span-3" />
            </div>
            <div className="grid grid-cols-4 items-center gap-4">
              <label htmlFor="endpoint" className="text-right text-sm font-medium col-span-1">
                Endpoint
              </label>
              <Input id="endpoint" defaultValue="https://api.example.com/v1" className="col-span-3" />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setIsOpen(false)}>Cancel</Button>
            <Button onClick={() => {
              toast({
                title: "Configuration saved",
                description: "Your service settings have been updated."
              });
              setIsOpen(false);
            }}>Save changes</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
      <LoginDialog isOpen={showLoginDialog} onClose={() => setShowLoginDialog(false)} />
      <Toaster />
    </div>
  )
}

// New component for the routes content
const AppContent = () => {
  const { toast } = useToast()
  const { isLoading: authIsLoading } = useAuth()

  useEffect(() => {
    // 订阅toast事件
    const unsubscribe = toastEmitter.subscribe((toastData) => {
      toast(toastData)
    })

    // 清理订阅
    return () => {
      unsubscribe()
    }
  }, [toast])

  if (authIsLoading) {
    return <div>Loading authentication...</div>
  }

  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route path="/oauth/github" element={<OAuthCallback />} />
      <Route path="/oauth/google" element={<OAuthCallback />} />
      <Route path="/" element={<AppLayout />}>
        <Route index element={<DashboardPage />} />
        <Route path="dashboard" element={<PrivateRoute><DashboardPage /></PrivateRoute>} />
        <Route path="services" element={<PrivateRoute><ServicesPage /></PrivateRoute>} />
        <Route path="market" element={<PrivateRoute><MarketPage /></PrivateRoute>} />
        <Route path="analytics" element={<PrivateRoute><AnalyticsPage /></PrivateRoute>} />
        <Route path="users" element={<PrivateRoute><UsersPage /></PrivateRoute>} />
        <Route path="logs" element={<PrivateRoute><LogsPage /></PrivateRoute>} />
        <Route path="profile" element={<PrivateRoute><ProfilePage /></PrivateRoute>} />
        <Route path="preferences" element={<PrivateRoute><PreferencesPage /></PrivateRoute>} />
        <Route path="api" element={<div>API Page Content</div>} />
        <Route path="models" element={<div>Models Page Content</div>} />
        <Route path="docs" element={<div>Docs Page Content</div>} />
      </Route>
    </Routes>
  )
}

export { AppContent }

function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <AppContent />
      </AuthProvider>
    </BrowserRouter>
  )
}

export default App

// 权限验证组件
const PrivateRoute: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { token, isLoading } = useAuth()
  const location = useLocation()

  if (isLoading) {
    return <div>Loading...</div>
  }

  if (!token) {
    return <Navigate to="/login" state={{ from: location }} replace />
  }

  return <>{children}</>
}
