import { useState } from 'react'
import './App.css'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { NavigationMenu, NavigationMenuContent, NavigationMenuItem, NavigationMenuLink, NavigationMenuList, NavigationMenuTrigger } from '@/components/ui/navigation-menu'
import { Table, TableBody, TableCaption, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Toaster } from '@/components/ui/toaster'
import { useToast } from '@/hooks/use-toast'
import { Settings, Search, User, Home, BarChart, PlusCircle, ChevronDown, Globe, Menu } from 'lucide-react'
import { LoginDialog } from './components/ui/login-dialog'
import { ThemeToggle } from './components/ui/theme-toggle'

function App() {
  const { toast } = useToast()
  const [isOpen, setIsOpen] = useState(false)
  const [showLoginDialog, setShowLoginDialog] = useState(false)

  // Function for links with hover effect
  const NavLink = ({ href, children }: { href: string, children: React.ReactNode }) => (
    <a
      href={href}
      className="text-sm font-medium text-muted-foreground hover:text-foreground transition-colors duration-200"
    >
      {children}
    </a>
  )

  return (
    <div className="min-h-screen bg-background text-foreground">
      {/* Header/Navigation - OpenRouter Style */}
      <header className="border-b border-border bg-background">
        <div className="container flex items-center h-16 px-4">
          <div className="flex items-center gap-2">
            <Globe className="h-6 w-6 text-primary" />
            <h1 className="text-xl font-semibold">One MCP</h1>
          </div>

          {/* Search Bar - OpenRouter Style */}
          <div className="mx-auto max-w-md w-full hidden md:block">
            <div className="relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                className="pl-10 bg-muted/40 border-muted rounded-md"
                placeholder="Search services..."
              />
            </div>
          </div>

          {/* Main Navigation */}
          <div className="flex items-center ml-auto gap-4">
            <nav className="hidden md:flex items-center gap-6">
              <NavLink href="#">API</NavLink>
              <NavLink href="#">Models</NavLink>
              <NavLink href="#">Dashboard</NavLink>
              <NavLink href="#">Docs</NavLink>
            </nav>
            <ThemeToggle />
            <Button
              size="sm"
              className="rounded-full transition-all duration-200 hover:opacity-90"
              onClick={() => setShowLoginDialog(true)}
            >
              Login
            </Button>
            <Button variant="ghost" size="icon" className="md:hidden">
              <Menu className="h-5 w-5" />
            </Button>
          </div>
        </div>
      </header>

      {/* Sidebar and Content */}
      <div className="flex">
        {/* Sidebar - OpenRouter Style */}
        <aside className="w-64 border-r border-border hidden md:block p-4">
          <nav className="space-y-2">
            <div className="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-3">
              Core
            </div>
            <a href="#" className="flex items-center gap-2 text-sm px-3 py-2 rounded-md hover:bg-muted transition-colors">
              <Home className="h-4 w-4 text-muted-foreground" />
              <span>Dashboard</span>
            </a>
            <a href="#" className="flex items-center gap-2 text-sm px-3 py-2 text-primary bg-primary/10 rounded-md font-medium">
              <Globe className="h-4 w-4" />
              <span>Services</span>
            </a>
            <a href="#" className="flex items-center gap-2 text-sm px-3 py-2 rounded-md hover:bg-muted transition-colors">
              <BarChart className="h-4 w-4 text-muted-foreground" />
              <span>Analytics</span>
            </a>

            <div className="text-xs font-semibold text-muted-foreground uppercase tracking-wider mt-6 mb-3">
              Settings
            </div>
            <a href="#" className="flex items-center gap-2 text-sm px-3 py-2 rounded-md hover:bg-muted transition-colors">
              <User className="h-4 w-4 text-muted-foreground" />
              <span>Profile</span>
            </a>
            <a href="#" className="flex items-center gap-2 text-sm px-3 py-2 rounded-md hover:bg-muted transition-colors">
              <Settings className="h-4 w-4 text-muted-foreground" />
              <span>Preferences</span>
            </a>
          </nav>
        </aside>

        {/* Main Content */}
        <main className="flex-1 p-6">
          <div className="flex justify-between items-center mb-8">
            <div>
              <h2 className="text-3xl font-bold tracking-tight">MCP Services</h2>
              <p className="text-muted-foreground mt-1">Manage and configure your multi-cloud platform services</p>
            </div>
            <Button onClick={() => toast({ title: "Action triggered", description: "You would create a new service here." })}>
              <PlusCircle className="w-4 h-4 mr-2" /> Add Service
            </Button>
          </div>

          <Tabs defaultValue="all" className="mb-8">
            <TabsList className="w-full max-w-md grid grid-cols-3">
              <TabsTrigger value="all" className="flex-1">All Services</TabsTrigger>
              <TabsTrigger value="active" className="flex-1">Active</TabsTrigger>
              <TabsTrigger value="inactive" className="flex-1">Inactive</TabsTrigger>
            </TabsList>
            <TabsContent value="all">
              <div className="grid gap-6 grid-cols-1 md:grid-cols-2 lg:grid-cols-3 mt-6">
                {[
                  { id: 1, name: "Search Service", status: "Active", description: "AI-powered semantic search", icon: <Search className="w-6 h-6 text-primary" /> },
                  { id: 2, name: "Analytics", status: "Active", description: "Usage tracking and reporting", icon: <BarChart className="w-6 h-6 text-primary" /> },
                  { id: 3, name: "User Management", status: "Inactive", description: "User access and controls", icon: <User className="w-6 h-6 text-primary" /> }
                ].map(service => (
                  <Card key={service.id} className="border-border shadow-sm hover:shadow transition-shadow duration-200">
                    <CardHeader>
                      <div className="flex items-center justify-between">
                        <div className="flex items-center">
                          <div className="bg-primary/10 p-2 rounded-md mr-3">
                            {service.icon}
                          </div>
                          <div>
                            <CardTitle className="text-lg">{service.name}</CardTitle>
                            <CardDescription>
                              <span className={`inline-flex items-center px-2 py-1 text-xs rounded-full ${service.status === "Active"
                                ? "bg-green-100 text-green-800"
                                : "bg-gray-100 text-gray-800"
                                }`}>
                                {service.status}
                              </span>
                            </CardDescription>
                          </div>
                        </div>
                      </div>
                    </CardHeader>
                    <CardContent>
                      <p className="text-sm text-muted-foreground">{service.description}</p>
                    </CardContent>
                    <CardFooter className="flex justify-between">
                      <Button variant="outline" size="sm" onClick={() => setIsOpen(true)}>Configure</Button>
                      <Button
                        variant={service.status === "Active" ? "outline" : "default"}
                        size="sm"
                        onClick={() => toast({
                          title: `Service ${service.status === "Active" ? "Disabled" : "Enabled"}`,
                          description: `${service.name} is now ${service.status === "Active" ? "inactive" : "active"}.`
                        })}
                      >
                        {service.status === "Active" ? "Disable" : "Enable"}
                      </Button>
                    </CardFooter>
                  </Card>
                ))}
              </div>
            </TabsContent>
            <TabsContent value="active">
              <p className="text-muted-foreground mt-4">Showing only active services.</p>
            </TabsContent>
            <TabsContent value="inactive">
              <p className="text-muted-foreground mt-4">Showing only inactive services.</p>
            </TabsContent>
          </Tabs>

          <div className="mt-12">
            <h3 className="text-2xl font-bold mb-4">Usage Statistics</h3>
            <Card className="shadow-sm">
              <CardHeader>
                <CardTitle>Service Utilization</CardTitle>
                <CardDescription>Performance overview for the last 30 days</CardDescription>
              </CardHeader>
              <CardContent>
                <Table>
                  <TableCaption>A summary of your service usage.</TableCaption>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Service</TableHead>
                      <TableHead>Requests</TableHead>
                      <TableHead>Success Rate</TableHead>
                      <TableHead className="text-right">Avg. Latency</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    <TableRow>
                      <TableCell className="font-medium">Search Service</TableCell>
                      <TableCell>12,423</TableCell>
                      <TableCell>99.8%</TableCell>
                      <TableCell className="text-right">132ms</TableCell>
                    </TableRow>
                    <TableRow>
                      <TableCell className="font-medium">Analytics</TableCell>
                      <TableCell>5,752</TableCell>
                      <TableCell>99.4%</TableCell>
                      <TableCell className="text-right">245ms</TableCell>
                    </TableRow>
                    <TableRow>
                      <TableCell className="font-medium">User Management</TableCell>
                      <TableCell>892</TableCell>
                      <TableCell>100%</TableCell>
                      <TableCell className="text-right">89ms</TableCell>
                    </TableRow>
                  </TableBody>
                </Table>
              </CardContent>
            </Card>
          </div>
        </main>
      </div>

      {/* Configuration Dialog */}
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

      {/* Login Dialog */}
      <LoginDialog isOpen={showLoginDialog} onClose={() => setShowLoginDialog(false)} />

      <Toaster />
    </div>
  )
}

export default App
