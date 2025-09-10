import React, { useState, useEffect, useRef } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Badge } from '@/components/ui/badge'
import { Loader2, Search, FileText, AlertCircle, Info, AlertTriangle } from 'lucide-react'
import { useToast } from '@/hooks/use-toast'
import { useTranslation } from 'react-i18next'
import api from '@/utils/api'

interface MCPLog {
    id: number
    service_id: number
    service_name: string
    phase: 'install' | 'run'
    level: 'info' | 'warn' | 'error'
    message: string
    created_at: string
    updated_at: string
}

interface LogsResponse {
    logs: MCPLog[];
    total: number;
    page: number;
    page_size: number;
}

const LogsPage: React.FC = () => {
    const { t } = useTranslation()
    const { toast } = useToast()

    // State
    const [logs, setLogs] = useState<MCPLog[]>([])
    const [loading, setLoading] = useState(false)
    const [total, setTotal] = useState(0)
    const [page, setPage] = useState(1)
    const [pageSize] = useState(20)

    // Filters
    const [serviceIdFilter, setServiceIdFilter] = useState('')
    const [serviceNameFilter, setServiceNameFilter] = useState('')
    const [phaseFilter, setPhaseFilter] = useState<string>('all')
    const [levelFilter, setLevelFilter] = useState<string>('all')

    // Load logs
    const loadLogs = async (currentPage = page) => {
        setLoading(true)
        try {
            const params = new URLSearchParams({
                page: currentPage.toString(),
                page_size: pageSize.toString(),
            })

            if (serviceIdFilter.trim()) {
                params.append('service_id', serviceIdFilter.trim())
            }
            if (serviceNameFilter.trim()) {
                params.append('service_name', serviceNameFilter.trim())
            }
            if (phaseFilter && phaseFilter !== 'all') {
                params.append('phase', phaseFilter)
            }
            if (levelFilter && levelFilter !== 'all') {
                params.append('level', levelFilter)
            }

            const response = await api.get<LogsResponse>(`/mcp_logs?${params.toString()}`)

            if (response.success) {
                const { logs, total, page } = response.data!
                setLogs(logs || [])
                setTotal(total || 0)
                setPage(page || 1)
            } else {
                throw new Error(response.message || 'Failed to load logs')
            }
        } catch (error: any) {
            console.error('Failed to load logs:', error)
            toast({
                variant: 'destructive',
                title: t('logs.loadError'),
                description: error.message || 'Unknown error'
            })
            setLogs([])
            setTotal(0)
        } finally {
            setLoading(false)
        }
    }

    // Avoid duplicate load in React StrictMode by using a ref guard
    const didInitRef = useRef(false)
    useEffect(() => {
        if (didInitRef.current) return
        didInitRef.current = true
        loadLogs(1)
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [])

    // Handle search
    const handleSearch = () => {
        loadLogs(1)
    }

    // Handle pagination
    const handlePageChange = (newPage: number) => {
        setPage(newPage)
        loadLogs(newPage)
    }

    // Clear filters
    const clearFilters = () => {
        setServiceIdFilter('')
        setServiceNameFilter('')
        setPhaseFilter('all')
        setLevelFilter('all')
    }

    // Format timestamp
    const formatTimestamp = (timestamp: string) => {
        return new Date(timestamp).toLocaleString()
    }

    // Get badge variant for log level
    const getLevelBadgeVariant = (level: string) => {
        switch (level) {
            case 'error':
                return 'destructive'
            case 'warn':
                return 'secondary'
            case 'info':
                return 'default'
            default:
                return 'outline'
        }
    }

    // Get icon for log level
    const getLevelIcon = (level: string) => {
        switch (level) {
            case 'error':
                return <AlertCircle className="h-4 w-4" />
            case 'warn':
                return <AlertTriangle className="h-4 w-4" />
            case 'info':
                return <Info className="h-4 w-4" />
            default:
                return <FileText className="h-4 w-4" />
        }
    }

    // Get badge variant for phase
    const getPhaseBadgeVariant = (phase: string) => {
        switch (phase) {
            case 'install':
                return 'secondary'
            case 'run':
                return 'default'
            default:
                return 'outline'
        }
    }

    const totalPages = Math.ceil(total / pageSize)

    return (
        <div className="container mx-auto py-6 space-y-6">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-3xl font-bold tracking-tight">{t('logs.title')}</h1>
                    <p className="text-muted-foreground">{t('logs.description')}</p>
                </div>
            </div>

            {/* Filters */}
            <Card>
                <CardHeader>
                    <CardTitle className="flex items-center gap-2">
                        <Search className="h-5 w-5" />
                        {t('logs.filters')}
                    </CardTitle>
                    <CardDescription>{t('logs.filtersDescription')}</CardDescription>
                </CardHeader>
                <CardContent>
                    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-4">
                        <div className="space-y-2">
                            <label className="text-sm font-medium">{t('logs.serviceId')}</label>
                            <Input
                                placeholder={t('logs.serviceIdPlaceholder')}
                                value={serviceIdFilter}
                                onChange={(e) => setServiceIdFilter(e.target.value)}
                                type="number"
                            />
                        </div>

                        <div className="space-y-2">
                            <label className="text-sm font-medium">{t('logs.serviceName')}</label>
                            <Input
                                placeholder={t('logs.serviceNamePlaceholder')}
                                value={serviceNameFilter}
                                onChange={(e) => setServiceNameFilter(e.target.value)}
                            />
                        </div>

                        <div className="space-y-2">
                            <label className="text-sm font-medium">{t('logs.phase')}</label>
                            <Select value={phaseFilter} onValueChange={setPhaseFilter}>
                                <SelectTrigger>
                                    <SelectValue placeholder={t('logs.allPhases')} />
                                </SelectTrigger>
                                <SelectContent>
                                    <SelectItem value="all">{t('logs.allPhases')}</SelectItem>
                                    <SelectItem value="install">{t('logs.install')}</SelectItem>
                                    <SelectItem value="run">{t('logs.run')}</SelectItem>
                                </SelectContent>
                            </Select>
                        </div>

                        <div className="space-y-2">
                            <label className="text-sm font-medium">{t('logs.level')}</label>
                            <Select value={levelFilter} onValueChange={setLevelFilter}>
                                <SelectTrigger>
                                    <SelectValue placeholder={t('logs.allLevels')} />
                                </SelectTrigger>
                                <SelectContent>
                                    <SelectItem value="all">{t('logs.allLevels')}</SelectItem>
                                    <SelectItem value="info">{t('logs.info')}</SelectItem>
                                    <SelectItem value="warn">{t('logs.warn')}</SelectItem>
                                    <SelectItem value="error">{t('logs.error')}</SelectItem>
                                </SelectContent>
                            </Select>
                        </div>
                    </div>

                    <div className="flex gap-2">
                        <Button onClick={handleSearch} variant="default">
                            <Search className="h-4 w-4 mr-2" />
                            {t('logs.search')}
                        </Button>
                        <Button onClick={clearFilters} variant="outline">
                            {t('logs.clearFilters')}
                        </Button>
                    </div>
                </CardContent>
            </Card>

            {/* Logs Table */}
            <Card>
                <CardHeader>
                    <CardTitle className="flex items-center gap-2">
                        <FileText className="h-5 w-5" />
                        {t('logs.logEntries')}
                    </CardTitle>
                    <CardDescription>
                        {total > 0
                            ? t('logs.totalEntries', { total, page, totalPages })
                            : t('logs.noEntries')
                        }
                    </CardDescription>
                </CardHeader>
                <CardContent>
                    {loading ? (
                        <div className="flex items-center justify-center py-12">
                            <Loader2 className="h-6 w-6 animate-spin mr-2" />
                            {t('logs.loading')}
                        </div>
                    ) : logs.length === 0 ? (
                        <div className="text-center py-12 text-muted-foreground">
                            <FileText className="h-12 w-12 mx-auto mb-4 opacity-50" />
                            <p className="text-lg font-medium">{t('logs.noLogsFound')}</p>
                            <p className="text-sm">{t('logs.tryDifferentFilters')}</p>
                        </div>
                    ) : (
                        <>
                            <div className="rounded-md border">
                                <Table>
                                    <TableHeader>
                                        <TableRow>
                                            <TableHead>{t('logs.timestamp')}</TableHead>
                                            <TableHead>{t('logs.service')}</TableHead>
                                            <TableHead>{t('logs.phase')}</TableHead>
                                            <TableHead>{t('logs.level')}</TableHead>
                                            <TableHead>{t('logs.message')}</TableHead>
                                        </TableRow>
                                    </TableHeader>
                                    <TableBody>
                                        {logs.map((log) => (
                                            <TableRow key={log.id}>
                                                <TableCell className="font-mono text-sm">
                                                    {formatTimestamp(log.created_at)}
                                                </TableCell>
                                                <TableCell>
                                                    <div>
                                                        <div className="font-medium">{log.service_name}</div>
                                                        <div className="text-sm text-muted-foreground">ID: {log.service_id}</div>
                                                    </div>
                                                </TableCell>
                                                <TableCell>
                                                    <Badge variant={getPhaseBadgeVariant(log.phase)}>
                                                        {t(`logs.${log.phase}`)}
                                                    </Badge>
                                                </TableCell>
                                                <TableCell>
                                                    <Badge variant={getLevelBadgeVariant(log.level)} className="flex items-center gap-1 w-fit">
                                                        {getLevelIcon(log.level)}
                                                        {t(`logs.${log.level}`)}
                                                    </Badge>
                                                </TableCell>
                                                <TableCell className="max-w-md">
                                                    <div className="break-words text-sm">
                                                        {log.message}
                                                    </div>
                                                </TableCell>
                                            </TableRow>
                                        ))}
                                    </TableBody>
                                </Table>
                            </div>

                            {/* Pagination */}
                            {totalPages > 1 && (
                                <div className="flex items-center justify-between mt-4">
                                    <div className="text-sm text-muted-foreground">
                                        {t('logs.showing', {
                                            start: (page - 1) * pageSize + 1,
                                            end: Math.min(page * pageSize, total),
                                            total
                                        })}
                                    </div>
                                    <div className="flex gap-2">
                                        <Button
                                            variant="outline"
                                            size="sm"
                                            onClick={() => handlePageChange(page - 1)}
                                            disabled={page <= 1}
                                        >
                                            {t('logs.previous')}
                                        </Button>
                                        <div className="flex items-center gap-1">
                                            {Array.from({ length: Math.min(5, totalPages) }, (_, i) => {
                                                const pageNum = Math.max(1, Math.min(totalPages - 4, page - 2)) + i
                                                return (
                                                    <Button
                                                        key={pageNum}
                                                        variant={pageNum === page ? 'default' : 'outline'}
                                                        size="sm"
                                                        onClick={() => handlePageChange(pageNum)}
                                                    >
                                                        {pageNum}
                                                    </Button>
                                                )
                                            })}
                                        </div>
                                        <Button
                                            variant="outline"
                                            size="sm"
                                            onClick={() => handlePageChange(page + 1)}
                                            disabled={page >= totalPages}
                                        >
                                            {t('logs.next')}
                                        </Button>
                                    </div>
                                </div>
                            )}
                        </>
                    )}
                </CardContent>
            </Card>
        </div>
    )
}

export default LogsPage

