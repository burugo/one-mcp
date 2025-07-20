import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button } from '@/components/ui/button';
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogFooter,
} from '@/components/ui/dialog';
import { Textarea } from '@/components/ui/textarea';

interface BatchImportModalProps {
    open: boolean;
    onClose: () => void;
    onImportSuccess: () => void;
}

type ViewState = 'input' | 'progress' | 'report';

// Placeholder for real progress update objects
interface ProgressUpdate {
    name: string;
    status: 'success' | 'skipped' | 'failed';
    message: string;
}

const BatchImportModal: React.FC<BatchImportModalProps> = ({
    open,
    onClose,
    onImportSuccess,
}) => {
    const { t } = useTranslation();
    const [view, setView] = useState<ViewState>('input');
    const [jsonInput, setJsonInput] = useState('');
    const [progressLogs, setProgressLogs] = useState<ProgressUpdate[]>([]);
    const [summary, setSummary] = useState({ success: 0, skipped: 0, failed: 0 });
    const [isImporting, setIsImporting] = useState(false);

    const handleImport = async () => {
        // Basic JSON validation
        try {
            JSON.parse(jsonInput);
        } catch (error) {
            alert(t('batchImport.invalidJsonError'));
            return;
        }

        setIsImporting(true);
        setView('progress');
        setProgressLogs([]);
        setSummary({ success: 0, skipped: 0, failed: 0 });

        try {
            // Call the backend API to start batch import
            const response = await fetch('/api/mcp_market/batch-import', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${localStorage.getItem('token')}`,
                },
                body: jsonInput,
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const data = await response.json();
            const taskId = data.task_id;

            // Establish SSE connection to receive progress updates
            const token = localStorage.getItem('token');
            const eventSource = new EventSource(`/api/mcp_market/batch-import/progress/${taskId}?token=${encodeURIComponent(token || '')}`);

            eventSource.onmessage = (event) => {
                try {
                    const update = JSON.parse(event.data);

                    if (update.status === 'done') {
                        // Final summary received
                        setSummary(update.summary);
                        setView('report');
                        setIsImporting(false);
                        eventSource.close();
                    } else {
                        // Progress update received
                        setProgressLogs(prev => [...prev, update]);

                        // Update running totals (optional, for real-time feedback)
                        setSummary(prev => ({
                            success: prev.success + (update.status === 'success' ? 1 : 0),
                            skipped: prev.skipped + (update.status === 'skipped' ? 1 : 0),
                            failed: prev.failed + (update.status === 'failed' ? 1 : 0),
                        }));
                    }
                } catch (parseError) {
                    console.error('Error parsing SSE data:', parseError);
                }
            };

            eventSource.onerror = (error) => {
                console.error('SSE connection error:', error);
                eventSource.close();
                setIsImporting(false);
                alert('连接错误，请重试');
                setView('input');
            };

        } catch (error) {
            console.error('Error starting batch import:', error);
            setIsImporting(false);
            alert('启动导入失败：' + (error instanceof Error ? error.message : String(error)));
            setView('input');
        }
    };

    const handleClose = () => {
        if (view === 'report') {
            onImportSuccess();
        }
        onClose();
        // Reset state for next time
        setTimeout(() => {
            setView('input');
            setJsonInput('');
            setProgressLogs([]);
            setSummary({ success: 0, skipped: 0, failed: 0 });
            setIsImporting(false);
        }, 300);
    };

    const renderInputView = () => (
        <>
            <DialogHeader>
                <DialogTitle>{t('batchImport.title')}</DialogTitle>
            </DialogHeader>
            <div className="py-4">
                <Textarea
                    value={jsonInput}
                    onChange={(e) => setJsonInput(e.target.value)}
                    placeholder={t('batchImport.placeholder')}
                    className="min-h-[300px] font-mono"
                />
            </div>
            <DialogFooter>
                <Button variant="outline" onClick={handleClose}>{t('common.cancel')}</Button>
                <Button onClick={handleImport} disabled={isImporting || !jsonInput.trim()}>
                    {isImporting ? t('batchImport.importing') : t('batchImport.import')}
                </Button>
            </DialogFooter>
        </>
    );

    const renderProgressView = () => (
        <>
            <DialogHeader>
                <DialogTitle>{t('batchImport.importing')}</DialogTitle>
            </DialogHeader>
            <div className="py-4">
                <div className="mb-4 text-center">
                    <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-primary mx-auto mb-2"></div>
                    <p className="text-sm text-muted-foreground">{t('batchImport.processingServices')}</p>
                    <p className="text-xs text-muted-foreground mt-1">
                        {t('batchImport.importSummary', {
                            success: summary.success,
                            skipped: summary.skipped,
                            failed: summary.failed
                        })}
                    </p>
                </div>

                {/* Real-time progress logs */}
                <div className="bg-muted rounded-md p-3 max-h-[300px] overflow-y-auto">
                    <div className="space-y-1 font-mono text-sm">
                        {progressLogs.map((log, index) => (
                            <div key={index} className="flex items-center space-x-2 py-1">
                                <span className={`
                                    inline-block w-2 h-2 rounded-full flex-shrink-0
                                    ${log.status === 'success' ? 'bg-green-500' :
                                        log.status === 'skipped' ? 'bg-yellow-500' : 'bg-red-500'}
                                `} style={{ minWidth: '8px', minHeight: '8px' }} />
                                <span className="font-medium">{log.name}</span>
                                <span className="text-muted-foreground text-xs">{log.message}</span>
                            </div>
                        ))}
                        {/* Cursor indicator */}
                        <div className="flex items-center space-x-2 py-1 animate-pulse">
                            <span className="inline-block w-2 h-2 rounded-full bg-primary flex-shrink-0" style={{ minWidth: '8px', minHeight: '8px' }}></span>
                            <span className="text-primary">_</span>
                        </div>
                    </div>
                </div>
            </div>
        </>
    );

    const renderReportView = () => (
        <>
            <DialogHeader>
                <DialogTitle>{t('batchImport.importComplete')}</DialogTitle>
            </DialogHeader>
            <div className="py-4">
                <div className="mb-4">
                    <p className="text-sm text-muted-foreground">
                        {t('batchImport.importSummary', {
                            success: summary.success,
                            skipped: summary.skipped,
                            failed: summary.failed
                        })}
                    </p>
                </div>
                <div className="space-y-2 max-h-[200px] overflow-y-auto">
                    {progressLogs.map((log, index) => (
                        <div key={index} className="flex items-center space-x-2 text-sm">
                            <span className={`
                                inline-block w-2 h-2 rounded-full
                                ${log.status === 'success' ? 'bg-green-500' :
                                    log.status === 'skipped' ? 'bg-yellow-500' : 'bg-red-500'}
                            `} />
                            <span className="font-mono">{log.name}</span>
                            <span className="text-muted-foreground">{log.message}</span>
                        </div>
                    ))}
                </div>
            </div>
            <DialogFooter>
                <Button onClick={handleClose}>{t('common.close')}</Button>
            </DialogFooter>
        </>
    );

    const renderContent = () => {
        switch (view) {
            case 'progress':
                return renderProgressView();
            case 'report':
                return renderReportView();
            case 'input':
            default:
                return renderInputView();
        }
    };

    return (
        <Dialog open={open} onOpenChange={(isOpen) => !isOpen && handleClose()}>
            <DialogContent className="sm:max-w-[600px]">
                {renderContent()}
            </DialogContent>
        </Dialog>
    );
};

export default BatchImportModal; 