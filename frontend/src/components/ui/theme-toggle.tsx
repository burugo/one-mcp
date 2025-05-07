import { Moon, Sun } from "lucide-react";
import { useEffect, useState } from "react";
import { Button } from "./button";

type Theme = "light" | "dark" | "system";

export function ThemeToggle() {
    const [theme, setTheme] = useState<Theme>(
        () => (localStorage.getItem("theme") as Theme) || "system"
    );

    useEffect(() => {
        const root = window.document.documentElement;

        // 移除之前的类
        root.classList.remove("light", "dark");

        // 根据主题设置添加类
        if (theme === "system") {
            const systemTheme = window.matchMedia("(prefers-color-scheme: dark)").matches
                ? "dark"
                : "light";
            root.classList.add(systemTheme);
            localStorage.removeItem("theme");
        } else {
            root.classList.add(theme);
            localStorage.setItem("theme", theme);
        }
    }, [theme]);

    // 监听系统主题变化
    useEffect(() => {
        const mediaQuery = window.matchMedia("(prefers-color-scheme: dark)");
        const handleChange = () => {
            if (theme === "system") {
                const root = window.document.documentElement;
                root.classList.remove("light", "dark");
                root.classList.add(mediaQuery.matches ? "dark" : "light");
            }
        };

        mediaQuery.addEventListener("change", handleChange);
        return () => mediaQuery.removeEventListener("change", handleChange);
    }, [theme]);

    function toggleTheme() {
        setTheme(prevTheme => {
            if (prevTheme === "light") return "dark";
            if (prevTheme === "dark") return "system";
            return "light";
        });
    }

    return (
        <Button
            variant="ghost"
            size="icon"
            onClick={toggleTheme}
            className="w-9 h-9 rounded-full"
            aria-label="Toggle theme"
        >
            {theme === "light" && <Sun className="h-4 w-4" />}
            {theme === "dark" && <Moon className="h-4 w-4" />}
            {theme === "system" && (
                <>
                    <Sun className="h-4 w-4 absolute transition-opacity dark:opacity-0" />
                    <Moon className="h-4 w-4 absolute transition-opacity opacity-0 dark:opacity-100" />
                </>
            )}
        </Button>
    );
} 