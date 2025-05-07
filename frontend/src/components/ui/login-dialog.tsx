import { useState } from "react";
import { Dialog, DialogContent } from "./dialog";
import { Input } from "./input";
import { Button } from "./button";
import { Eye, EyeOff } from "lucide-react";

interface LoginDialogProps {
    isOpen: boolean;
    onClose: () => void;
}

export function LoginDialog({ isOpen, onClose }: LoginDialogProps) {
    const [showPassword, setShowPassword] = useState(false);
    const [email, setEmail] = useState("");
    const [password, setPassword] = useState("");

    return (
        <Dialog open={isOpen} onOpenChange={onClose}>
            <DialogContent className="sm:max-w-md p-0 gap-0 border-none">
                <div className="p-6 pt-8 flex flex-col items-center">
                    <h2 className="text-2xl font-semibold text-center mb-2">Sign in to One MCP</h2>
                    <p className="text-muted-foreground text-center mb-8">Welcome back! Please sign in to continue</p>

                    {/* Social Login Options */}
                    <div className="grid grid-cols-3 gap-4 w-full mb-8">
                        <Button variant="outline" className="flex items-center justify-center py-6 hover:bg-muted/50">
                            <svg viewBox="0 0 24 24" width="24" height="24" aria-hidden="true" className="text-foreground">
                                <path fill="currentColor" d="M12 .297c-6.63 0-12 5.373-12 12 0 5.303 3.438 9.8 8.205 11.385.6.113.82-.258.82-.577 0-.285-.01-1.04-.015-2.04-3.338.724-4.042-1.61-4.042-1.61C4.422 18.07 3.633 17.7 3.633 17.7c-1.087-.744.084-.729.084-.729 1.205.084 1.838 1.236 1.838 1.236 1.07 1.835 2.809 1.305 3.495.998.108-.776.417-1.305.76-1.605-2.665-.3-5.466-1.332-5.466-5.93 0-1.31.465-2.38 1.235-3.22-.135-.303-.54-1.523.105-3.176 0 0 1.005-.322 3.3 1.23.96-.267 1.98-.399 3-.405 1.02.006 2.04.138 3 .405 2.28-1.552 3.285-1.23 3.285-1.23.645 1.653.24 2.873.12 3.176.765.84 1.23 1.91 1.23 3.22 0 4.61-2.805 5.625-5.475 5.92.42.36.81 1.096.81 2.22 0 1.606-.015 2.896-.015 3.286 0 .315.21.69.825.57C20.565 22.092 24 17.592 24 12.297c0-6.627-5.373-12-12-12" />
                            </svg>
                        </Button>
                        <Button variant="outline" className="flex items-center justify-center py-6 hover:bg-muted/50">
                            <svg viewBox="0 0 24 24" width="24" height="24" aria-hidden="true" className="text-foreground">
                                <path fill="#4285F4" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z" />
                                <path fill="#34A853" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" />
                                <path fill="#FBBC05" d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z" />
                                <path fill="#EA4335" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" />
                            </svg>
                        </Button>
                        <Button variant="outline" className="flex items-center justify-center py-6 hover:bg-muted/50">
                            <svg viewBox="0 0 24 24" width="24" height="24" aria-hidden="true" fill="none" stroke="#ff9c0c" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                                <path d="M18.5 8c-3.5 0-4 3.5-6.5 3.5S9 8 5.5 8 2 11.5 2 11.5 5.5 16 9 16s5-3.5 7.5-3.5 3.5 3.5 7 3.5c0 0 .5-4.5-5-8Z" />
                                <circle cx="8.5" cy="10.5" r="1.5" />
                                <circle cx="15.5" cy="13.5" r="1.5" />
                            </svg>
                        </Button>
                    </div>

                    <div className="flex items-center w-full mb-6">
                        <div className="h-px bg-border flex-grow"></div>
                        <span className="px-4 text-muted-foreground">or</span>
                        <div className="h-px bg-border flex-grow"></div>
                    </div>

                    <div className="w-full mb-4">
                        <label htmlFor="email" className="block text-sm font-medium mb-2">
                            Email address
                        </label>
                        <Input
                            id="email"
                            type="email"
                            value={email}
                            onChange={(e) => setEmail(e.target.value)}
                            className="bg-sky-50 border-sky-100 px-4 py-6 text-base"
                            placeholder="your.email@example.com"
                        />
                    </div>

                    <div className="w-full mb-8">
                        <label htmlFor="password" className="block text-sm font-medium mb-2">
                            Password
                        </label>
                        <div className="relative">
                            <Input
                                id="password"
                                type={showPassword ? "text" : "password"}
                                value={password}
                                onChange={(e) => setPassword(e.target.value)}
                                className="bg-sky-50 border-sky-100 px-4 py-6 text-base pr-10"
                            />
                            <button
                                type="button"
                                onClick={() => setShowPassword(!showPassword)}
                                className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-500 hover:text-gray-700"
                            >
                                {showPassword ? <EyeOff size={20} /> : <Eye size={20} />}
                            </button>
                        </div>
                    </div>

                    <Button className="w-full py-6 text-base font-medium rounded-md">
                        Continue <span className="ml-2">➔</span>
                    </Button>

                    <p className="mt-8 text-center text-muted-foreground">
                        Don't have an account?{" "}
                        <a href="#" className="text-primary hover:text-primary/80 transition-colors font-medium">
                            Sign up
                        </a>
                    </p>
                </div>
            </DialogContent>
        </Dialog>
    );
} 