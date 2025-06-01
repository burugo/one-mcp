import axios from 'axios';

// API响应类型
export interface APIResponse<T = any> {
    success: boolean;
    message?: string;
    data?: T;
}

// 创建一个简单的事件发布订阅系统
type ToastType = {
    variant?: "default" | "destructive";
    title: string;
    description: string;
};

type ToastCallback = (toast: ToastType) => void;

class ToastEmitter {
    private static instance: ToastEmitter;
    private callbacks: ToastCallback[] = [];

    private constructor() { }

    static getInstance(): ToastEmitter {
        if (!ToastEmitter.instance) {
            ToastEmitter.instance = new ToastEmitter();
        }
        return ToastEmitter.instance;
    }

    subscribe(callback: ToastCallback) {
        this.callbacks.push(callback);
        return () => {
            this.callbacks = this.callbacks.filter(cb => cb !== callback);
        };
    }

    emit(toast: ToastType) {
        this.callbacks.forEach(callback => callback(toast));
    }
}

export const toastEmitter = ToastEmitter.getInstance();

// 创建axios实例，统一管理API请求
const api = axios.create({
    baseURL: '/api', // 使用相对路径，将由Vite代理转发到后端
    timeout: 30000,
    headers: {
        'Content-Type': 'application/json',
    },
});

// 请求拦截器
api.interceptors.request.use(
    (config) => {
        // 从localStorage获取token
        const token = localStorage.getItem('token');
        if (token) {
            config.headers['Authorization'] = `Bearer ${token}`;
        }
        return config;
    },
    (error) => {
        return Promise.reject(error);
    }
);

// 响应拦截器
api.interceptors.response.use(
    (response) => {
        // 直接返回响应数据，使用类型断言绕过TypeScript检查但保持运行时行为不变
        return response.data as any;
    },
    async (error) => {
        const { response } = error;

        if (response) {
            // 处理特定状态码
            switch (response.status) {
                case 401: {
                    // 清除本地存储的认证信息
                    localStorage.removeItem('token');
                    localStorage.removeItem('refresh_token');
                    localStorage.removeItem('user');

                    // 重定向到登录页
                    window.location.href = '/login';
                    break;
                }
                case 403: {
                    toastEmitter.emit({
                        variant: "destructive",
                        title: "无权限",
                        description: "您没有权限执行此操作"
                    });
                    break;
                }
                case 404:
                    toastEmitter.emit({
                        variant: "destructive",
                        title: "请求失败",
                        description: "请求的资源不存在"
                    });
                    break;
                case 500:
                    toastEmitter.emit({
                        variant: "destructive",
                        title: "服务器错误",
                        description: "服务器错误，请稍后重试"
                    });
                    break;
                default: {
                    toastEmitter.emit({
                        variant: "destructive",
                        title: "请求失败",
                        description: response.data?.message || "未知错误"
                    });
                }
            }
        } else {
            toastEmitter.emit({
                variant: "destructive",
                title: "网络错误",
                description: "请检查网络连接"
            });
        }
        return Promise.reject(error);
    }
);

export default api; 