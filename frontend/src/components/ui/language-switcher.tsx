import { useTranslation } from 'react-i18next';
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from '@/components/ui/select';
import { Globe } from 'lucide-react';

// 语言配置 - 支持未来扩展
const languages = [
    { code: 'en', name: 'English', nativeName: 'EN' },
    { code: 'zh-CN', name: 'Chinese (Simplified)', nativeName: '中文' },
    // 未来可以轻松添加更多语言：
    // { code: 'ja', name: 'Japanese', nativeName: '日本語' },
    // { code: 'ko', name: 'Korean', nativeName: '한국어' },
    // { code: 'fr', name: 'French', nativeName: 'Français' },
    // { code: 'de', name: 'German', nativeName: 'Deutsch' },
    // { code: 'es', name: 'Spanish', nativeName: 'Español' },
] as const;

interface LanguageSwitcherProps {
    className?: string;
}

export function LanguageSwitcher({ className }: LanguageSwitcherProps) {
    const { i18n } = useTranslation();

    const currentLanguage = languages.find(lang => lang.code === i18n.language) || languages[0];

    const handleLanguageChange = (languageCode: string) => {
        i18n.changeLanguage(languageCode);
    };

    return (
        <Select value={i18n.language} onValueChange={handleLanguageChange}>
            <SelectTrigger className={`w-[120px] h-8 ${className}`}>
                <div className="flex items-center gap-2">
                    <Globe className="h-3.5 w-3.5" />
                    <SelectValue placeholder="Language">
                        {currentLanguage.nativeName}
                    </SelectValue>
                </div>
            </SelectTrigger>
            <SelectContent>
                {languages.map((language) => (
                    <SelectItem key={language.code} value={language.code}>
                        <div className="flex items-center justify-between w-full">
                            <span className="font-medium">{language.nativeName}</span>
                            <span className="text-muted-foreground text-xs ml-2">{language.name}</span>
                        </div>
                    </SelectItem>
                ))}
            </SelectContent>
        </Select>
    );
} 