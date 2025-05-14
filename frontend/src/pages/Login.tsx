import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { useToast } from '@/hooks/use-toast';
import api from '@/utils/api';

interface LoginForm {
    username: string;
    password: string;
}

const Login: React.FC = () => {
    const navigate = useNavigate();
    const { toast } = useToast();
    const [loading, setLoading] = useState(false);
    const [formData, setFormData] = useState<LoginForm>({
        username: '',
        password: ''
    });

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        try {
            setLoading(true);
            const response = await api.post('/auth/login', formData);

            if (response.success) {
                // 保存token和用户信息
                localStorage.setItem('token', response.data.access_token);
                localStorage.setItem('refresh_token', response.data.refresh_token);
                localStorage.setItem('user', JSON.stringify(response.data.user));

                toast({
                    title: "登录成功",
                    description: "欢迎回来！"
                });
                navigate('/');
            } else {
                toast({
                    variant: "destructive",
                    title: "登录失败",
                    description: response.message || "请检查用户名和密码"
                });
            }
        } catch (error: any) {
            toast({
                variant: "destructive",
                title: "登录失败",
                description: error.message || "请检查用户名和密码"
            });
        } finally {
            setLoading(false);
        }
    };

    const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        const { name, value } = e.target;
        setFormData(prev => ({
            ...prev,
            [name]: value
        }));
    };

    return (
        <div className="min-h-screen flex items-center justify-center bg-background">
            <div className="w-full max-w-md p-8 space-y-8">
                <div className="text-center">
                    <h1 className="text-2xl font-bold tracking-tight">登录</h1>
                    <p className="text-sm text-muted-foreground mt-2">
                        欢迎使用 One MCP
                    </p>
                </div>

                <form onSubmit={handleSubmit} className="space-y-6">
                    <div className="space-y-2">
                        <label htmlFor="username" className="text-sm font-medium">
                            用户名
                        </label>
                        <Input
                            id="username"
                            name="username"
                            type="text"
                            required
                            placeholder="请输入用户名"
                            value={formData.username}
                            onChange={handleInputChange}
                            className="w-full"
                        />
                    </div>

                    <div className="space-y-2">
                        <label htmlFor="password" className="text-sm font-medium">
                            密码
                        </label>
                        <Input
                            id="password"
                            name="password"
                            type="password"
                            required
                            placeholder="请输入密码"
                            value={formData.password}
                            onChange={handleInputChange}
                            className="w-full"
                        />
                    </div>

                    <Button
                        type="submit"
                        className="w-full"
                        disabled={loading}
                    >
                        {loading ? "登录中..." : "登录"}
                    </Button>
                </form>
            </div>
        </div>
    );
};

export default Login; 