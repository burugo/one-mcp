import { useState } from 'react'
import './App.css'
import { BrowserRouter, Routes, Route, Link, Outlet, useLocation } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Toaster } from '@/components/ui/toaster'
import { useToast } from '@/hooks/use-toast'
import { Settings, Search, User, Home, BarChart, PlusCircle, Globe, Menu, Package, Server, Activity, Clock, Database, AlertCircle, CheckCircle } from 'lucide-react'
import { LoginDialog } from './components/ui/login-dialog'
import { ThemeToggle } from './components/ui/theme-toggle'
import { MarketPage } from './pages/MarketPage'
import { DashboardPage } from './pages/DashboardPage'
import { ServicesPage } from './pages/ServicesPage'
import { AnalyticsPage } from './pages/AnalyticsPage'
import { ProfilePage } from './pages/ProfilePage'
import { PreferencesPage } from './pages/PreferencesPage'

// Props that might be passed down from AppLayout to pages via Outlet context
export interface PageOutletContext {
  setIsOpen: React.Dispatch<React.SetStateAction<boolean>>
}

const AppLayout = () => {
  const { toast } = useToast()
  const [isOpen, setIsOpen] = useState(false)
  const [showLoginDialog, setShowLoginDialog] = useState(false)
  const location = useLocation()

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

  return (
    <div className="min-h-screen flex flex-col bg-background text-foreground">
      <header className="border-b border-border bg-background sticky top-0 z-10">
        <div className="flex items-center h-16 px-6">
          <Link to="/" className="flex items-center gap-2">
            <Globe className="h-6 w-6 text-primary" />
            <h1 className="text-xl font-semibold">One MCP</h1>
          </Link>
          <div className="mx-auto max-w-md w-full hidden md:block px-4">
            <div className="relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                className="pl-10 bg-muted/40 border-muted rounded-md"
                placeholder="Search services..."
              />
            </div>
          </div>
          <div className="flex items-center ml-auto gap-6">
            <nav className="hidden md:flex items-center gap-6">
              <NavLink to="/api" isTopNav>API</NavLink>
              <NavLink to="/models" isTopNav>Models</NavLink>
              <NavLink to="/" isTopNav>Dashboard</NavLink>
              <NavLink to="/docs" isTopNav>Docs</NavLink>
            </nav>
            <ThemeToggle />
            <Button
              size="sm"
              className="rounded-full transition-all duration-200 hover:opacity-90 bg-[#7c3aed] hover:bg-[#7c3aed]/90"
              onClick={() => setShowLoginDialog(true)}
            >
              Login
            </Button>
          </div>
        </div>
      </header>
      <div className="flex flex-1 overflow-hidden">
        <aside className="w-64 border-r border-border hidden md:block overflow-y-auto flex-shrink-0 bg-background/80">
          <nav className="p-4 space-y-1">
            <NavLink to="/">
              <Home className={`h-4 w-4 ${location.pathname === '/' || location.pathname.startsWith('/dashboard') ? 'text-primary' : 'text-muted-foreground'}`} />
              <span className={`${location.pathname === '/' || location.pathname.startsWith('/dashboard') ? 'text-primary' : 'text-muted-foreground'}`}>Dashboard</span>
            </NavLink>
            <NavLink to="/services">
              <Globe className={`h-4 w-4 ${location.pathname.startsWith('/services') ? 'text-primary' : 'text-muted-foreground'}`} />
              <span className={`${location.pathname.startsWith('/services') ? 'text-primary' : 'text-muted-foreground'}`}>Services</span>
            </NavLink>
            <NavLink to="/market">
              <Package className={`h-4 w-4 ${location.pathname.startsWith('/market') ? 'text-primary' : 'text-muted-foreground'}`} />
              <span className={`${location.pathname.startsWith('/market') ? 'text-primary' : 'text-muted-foreground'}`}>Service Market</span>
            </NavLink>
            <NavLink to="/analytics">
              <BarChart className={`h-4 w-4 ${location.pathname.startsWith('/analytics') ? 'text-primary' : 'text-muted-foreground'}`} />
              <span className={`${location.pathname.startsWith('/analytics') ? 'text-primary' : 'text-muted-foreground'}`}>Analytics</span>
            </NavLink>
            <div className="my-4 border-t border-border"></div>
            <NavLink to="/profile">
              <User className={`h-4 w-4 ${location.pathname.startsWith('/profile') ? 'text-primary' : 'text-muted-foreground'}`} />
              <span className={`${location.pathname.startsWith('/profile') ? 'text-primary' : 'text-muted-foreground'}`}>Profile</span>
            </NavLink>
            <NavLink to="/preferences">
              <Settings className={`h-4 w-4 ${location.pathname.startsWith('/preferences') ? 'text-primary' : 'text-muted-foreground'}`} />
              <span className={`${location.pathname.startsWith('/preferences') ? 'text-primary' : 'text-muted-foreground'}`}>Preferences</span>
            </NavLink>
          </nav>
        </aside>
        <main className="flex-1 p-6 overflow-y-scroll h-[calc(100vh-64px)] bg-background/50 max-w-7xl mx-auto w-full">
          <Outlet context={{ setIsOpen }} />
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
  return (
    <Routes>
      <Route element={<AppLayout />}>
        <Route index element={<DashboardPage />} />
        <Route path="dashboard" element={<DashboardPage />} />
        <Route path="services" element={<ServicesPage />} />
        <Route path="market" element={<MarketPage />} />
        <Route path="analytics" element={<AnalyticsPage />} />
        <Route path="profile" element={<ProfilePage />} />
        <Route path="preferences" element={<PreferencesPage />} />
        <Route path="api" element={<div>API Page Content</div>} />
        <Route path="models" element={<div>Models Page Content</div>} />
        <Route path="docs" element={<div>Docs Page Content</div>} />
      </Route>
    </Routes>
  );
}

function App() {
  return (
    <BrowserRouter>
      <AppContent />
    </BrowserRouter>
  );
}

export default App;
// Export AppContent for testing purposes
export { AppContent };
