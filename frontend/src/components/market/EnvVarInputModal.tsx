import React, { useState } from 'react';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogDescription } from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { useTranslation } from 'react-i18next';

interface EnvVarInputModalProps {
    open: boolean;
    missingVars: string[];
    onSubmit: (envVars: Record<string, string>) => void;
    onCancel: () => void;
}

const EnvVarInputModal: React.FC<EnvVarInputModalProps> = ({ open, missingVars, onSubmit, onCancel }) => {
    const { t } = useTranslation();
    const [envValues, setEnvValues] = useState<Record<string, string>>({});
    const [error, setError] = useState<string | null>(null);

    const handleChange = (name: string, value: string) => {
        setEnvValues((prev) => ({ ...prev, [name]: value }));
    };

    const handleSubmit = () => {
        // 检查所有必填项
        for (const varName of missingVars) {
            if (!envValues[varName] || envValues[varName].trim() === '') {
                setError(t('envVarModal.errorRequired', { varName }));
                return;
            }
        }
        setError(null);
        onSubmit(envValues);
    };

    return (
        <Dialog open={open} onOpenChange={onCancel}>
            <DialogContent className="sm:max-w-md">
                <DialogHeader>
                    <DialogTitle>{t('envVarModal.title')}</DialogTitle>
                    <DialogDescription>
                        {t('envVarModal.description')}
                    </DialogDescription>
                </DialogHeader>
                <div className="space-y-4 mt-2">
                    {missingVars.map((varName) => (
                        <div key={varName}>
                            <label className="block text-sm font-medium mb-1">{varName}</label>
                            <Input
                                type="text"
                                value={envValues[varName] || ''}
                                onChange={(e) => handleChange(varName, e.target.value)}
                                placeholder={t('envVarModal.placeholder', { varName })}
                                autoFocus={missingVars[0] === varName}
                            />
                        </div>
                    ))}
                    {error && <div className="text-red-500 text-sm mt-2">{error}</div>}
                </div>
                <DialogFooter className="mt-4">
                    <Button variant="outline" onClick={onCancel} type="button">{t('envVarModal.cancel')}</Button>
                    <Button onClick={handleSubmit} type="button">{t('envVarModal.confirm')}</Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
};

export default EnvVarInputModal; 