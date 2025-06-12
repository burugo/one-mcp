import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import HttpBackend from 'i18next-http-backend';
import LanguageDetector from 'i18next-browser-languagedetector';

i18n
    // 使用 HTTP 后端加载翻译文件
    .use(HttpBackend)
    // 使用浏览器语言检测器
    .use(LanguageDetector)
    // 与 React 集成
    .use(initReactI18next)
    // 初始化 i18next
    .init({
        // 语言检测配置
        detection: {
            // 检测顺序：localStorage -> navigator -> 默认语言
            order: ['localStorage', 'navigator'],
            // 缓存用户选择到 localStorage
            caches: ['localStorage'],
            // localStorage 中的键名
            lookupLocalStorage: 'i18nextLng',
        },

        // 支持的语言列表（BCP 47 标准）
        supportedLngs: ['en', 'zh-CN'],

        // 后备语言
        fallbackLng: 'en',

        // 默认命名空间
        defaultNS: 'translation',

        // HTTP 后端配置
        backend: {
            // 翻译文件的加载路径
            loadPath: '/locales/{{lng}}/{{ns}}.json',
        },

        // 调试模式（开发环境）
        debug: import.meta.env.DEV,

        // 插值配置
        interpolation: {
            // React 已经安全地转义了，所以不需要额外转义
            escapeValue: false,
        },
    });

export default i18n; 