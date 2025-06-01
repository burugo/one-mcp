import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { useToast, toast, reducer } from './use-toast'

// Mock setTimeout and clearTimeout
vi.useFakeTimers()

describe('useToast Hook', () => {
    beforeEach(() => {
        vi.clearAllTimers()
    })

    afterEach(() => {
        vi.clearAllTimers()
    })

    it('should initialize with empty toasts array', () => {
        const { result } = renderHook(() => useToast())
        expect(result.current.toasts).toEqual([])
    })

    it('should add a toast', () => {
        const { result } = renderHook(() => useToast())

        act(() => {
            result.current.toast({
                title: 'Test Toast',
                description: 'This is a test toast',
            })
        })

        expect(result.current.toasts).toHaveLength(1)
        expect(result.current.toasts[0]).toMatchObject({
            title: 'Test Toast',
            description: 'This is a test toast',
            open: true,
        })
        expect(result.current.toasts[0].id).toBeDefined()
    })

    it('should dismiss a specific toast', () => {
        const { result } = renderHook(() => useToast())

        let toastId: string

        act(() => {
            const toastResult = result.current.toast({
                title: 'Test Toast',
            })
            toastId = toastResult.id
        })

        expect(result.current.toasts[0].open).toBe(true)

        act(() => {
            result.current.dismiss(toastId)
        })

        expect(result.current.toasts[0].open).toBe(false)
    })

    it('should dismiss all toasts when no id provided', () => {
        const { result } = renderHook(() => useToast())

        act(() => {
            result.current.toast({ title: 'Toast 1' })
            result.current.toast({ title: 'Toast 2' })
        })

        expect(result.current.toasts).toHaveLength(1) // TOAST_LIMIT = 1

        act(() => {
            result.current.dismiss()
        })

        expect(result.current.toasts[0].open).toBe(false)
    })

    it('should limit toasts to TOAST_LIMIT', () => {
        const { result } = renderHook(() => useToast())

        act(() => {
            result.current.toast({ title: 'Toast 1' })
            result.current.toast({ title: 'Toast 2' })
            result.current.toast({ title: 'Toast 3' })
        })

        // Should only keep the latest toast due to TOAST_LIMIT = 1
        expect(result.current.toasts).toHaveLength(1)
        expect(result.current.toasts[0].title).toBe('Toast 3')
    })

    it('should handle toast variants', () => {
        const { result } = renderHook(() => useToast())

        act(() => {
            result.current.toast({
                title: 'Error Toast',
                variant: 'destructive',
            })
        })

        expect(result.current.toasts[0].variant).toBe('destructive')
    })
})

describe('toast function', () => {
    it('should return toast control functions', () => {
        const toastResult = toast({
            title: 'Test Toast',
        })

        expect(toastResult).toHaveProperty('id')
        expect(toastResult).toHaveProperty('dismiss')
        expect(toastResult).toHaveProperty('update')
        expect(typeof toastResult.dismiss).toBe('function')
        expect(typeof toastResult.update).toBe('function')
    })

    it('should update toast', () => {
        const { result } = renderHook(() => useToast())

        let toastResult: ReturnType<typeof toast>

        act(() => {
            toastResult = toast({
                title: 'Original Title',
            })
        })

        act(() => {
            toastResult.update({
                id: toastResult.id,
                title: 'Updated Title',
                description: 'Updated Description',
            })
        })

        expect(result.current.toasts[0].title).toBe('Updated Title')
        expect(result.current.toasts[0].description).toBe('Updated Description')
    })

    it('should dismiss toast using returned dismiss function', () => {
        const { result } = renderHook(() => useToast())

        let toastResult: ReturnType<typeof toast>

        act(() => {
            toastResult = toast({
                title: 'Test Toast',
            })
        })

        expect(result.current.toasts[0].open).toBe(true)

        act(() => {
            toastResult.dismiss()
        })

        expect(result.current.toasts[0].open).toBe(false)
    })
})

describe('reducer', () => {
    const initialState = { toasts: [] }

    it('should add toast', () => {
        const toast = {
            id: '1',
            title: 'Test Toast',
            open: true,
        }

        const newState = reducer(initialState, {
            type: 'ADD_TOAST',
            toast,
        })

        expect(newState.toasts).toHaveLength(1)
        expect(newState.toasts[0]).toEqual(toast)
    })

    it('should update toast', () => {
        const initialStateWithToast = {
            toasts: [
                {
                    id: '1',
                    title: 'Original Title',
                    open: true,
                },
            ],
        }

        const newState = reducer(initialStateWithToast, {
            type: 'UPDATE_TOAST',
            toast: {
                id: '1',
                title: 'Updated Title',
            },
        })

        expect(newState.toasts[0].title).toBe('Updated Title')
        expect(newState.toasts[0].open).toBe(true) // Should preserve other properties
    })

    it('should dismiss toast', () => {
        const initialStateWithToast = {
            toasts: [
                {
                    id: '1',
                    title: 'Test Toast',
                    open: true,
                },
            ],
        }

        const newState = reducer(initialStateWithToast, {
            type: 'DISMISS_TOAST',
            toastId: '1',
        })

        expect(newState.toasts[0].open).toBe(false)
    })

    it('should remove toast', () => {
        const initialStateWithToast = {
            toasts: [
                {
                    id: '1',
                    title: 'Test Toast',
                    open: true,
                },
                {
                    id: '2',
                    title: 'Another Toast',
                    open: true,
                },
            ],
        }

        const newState = reducer(initialStateWithToast, {
            type: 'REMOVE_TOAST',
            toastId: '1',
        })

        expect(newState.toasts).toHaveLength(1)
        expect(newState.toasts[0].id).toBe('2')
    })

    it('should remove all toasts when no toastId provided', () => {
        const initialStateWithToasts = {
            toasts: [
                { id: '1', title: 'Toast 1', open: true },
                { id: '2', title: 'Toast 2', open: true },
            ],
        }

        const newState = reducer(initialStateWithToasts, {
            type: 'REMOVE_TOAST',
            toastId: undefined,
        })

        expect(newState.toasts).toHaveLength(0)
    })
}) 