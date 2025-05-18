import React, { useState } from 'react';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogDescription } from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';

interface EnvVarInputModalProps {
    open: boolean;
    missingVars: string[];
    onSubmit: (envVars: Record<string, string>) => void;
    onCancel: () => void;
}

const EnvVarInputModal: React.FC<EnvVarInputModalProps> = ({ open, missingVars, onSubmit, onCancel }) => {
    const [envValues, setEnvValues] = useState<Record<string, string>>({});
    const [error, setError] = useState<string | null>(null);

    const handleChange = (name: string, value: string) => {
        setEnvValues((prev) => ({ ...prev, [name]: value }));
    };

    const handleSubmit = () => {
        // 检查所有必填项
        for (const varName of missingVars) {
            if (!envValues[varName] || envValues[varName].trim() === '') {
                setError(`请填写 ${varName}`);
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
                    <DialogTitle>填写所需环境变量</DialogTitle>
                    <DialogDescription>
                        安装该服务需要以下环境变量，请补充完整后继续。
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
                                placeholder={`请输入 ${varName}`}
                                autoFocus={missingVars[0] === varName}
                            />
                        </div>
                    ))}
                    {error && <div className="text-red-500 text-sm mt-2">{error}</div>}
                </div>
                <DialogFooter className="mt-4">
                    <Button variant="outline" onClick={onCancel} type="button">取消</Button>
                    <Button onClick={handleSubmit} type="button">确认</Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
};

export default EnvVarInputModal; 