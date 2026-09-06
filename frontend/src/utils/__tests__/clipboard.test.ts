import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { copyToClipboard, isClipboardSupported, getClipboardErrorMessage } from '../clipboard';

const mockWriteText = vi.fn();
const mockExecCommand = vi.fn();
const restorers: Array<() => void> = [];

// Mock only browser capabilities; keep real DOM nodes and restore descriptors.
function mockProperty(target: object, key: string, value: unknown) {
    const descriptor = Object.getOwnPropertyDescriptor(target, key);
    Object.defineProperty(target, key, { configurable: true, writable: true, value });
    restorers.push(() => {
        if (descriptor) Object.defineProperty(target, key, descriptor);
        else Reflect.deleteProperty(target, key);
    });
}

describe('clipboard utils', () => {
    beforeEach(() => {
        mockWriteText.mockReset();
        mockExecCommand.mockReset();
        mockProperty(navigator, 'clipboard', undefined);
        mockProperty(window, 'isSecureContext', false);
        mockProperty(document, 'execCommand', mockExecCommand);
    });

    afterEach(() => {
        while (restorers.length) restorers.pop()!();
        vi.restoreAllMocks();
    });

    describe('copyToClipboard', () => {
        it('should use modern clipboard API when available', async () => {
            mockProperty(navigator, 'clipboard', { writeText: mockWriteText });
            mockProperty(window, 'isSecureContext', true);
            mockWriteText.mockResolvedValue(undefined);

            expect(await copyToClipboard('test text')).toEqual({ success: true, method: 'modern' });
            expect(mockWriteText).toHaveBeenCalledWith('test text');
            expect(mockExecCommand).not.toHaveBeenCalled();
        });

        it('should fallback to legacy method when modern API fails', async () => {
            mockProperty(navigator, 'clipboard', { writeText: mockWriteText });
            mockProperty(window, 'isSecureContext', true);
            mockWriteText.mockRejectedValue(new Error('Permission denied'));
            mockExecCommand.mockImplementation(() => {
                expect(document.activeElement).toBeInstanceOf(HTMLTextAreaElement);
                expect((document.activeElement as HTMLTextAreaElement).value).toBe('test text');
                return true;
            });

            expect(await copyToClipboard('test text')).toEqual({ success: true, method: 'legacy' });
            expect(mockWriteText).toHaveBeenCalledWith('test text');
            expect(mockExecCommand).toHaveBeenCalledWith('copy');
            expect(document.querySelector('textarea')).toBeNull();
        });

        it('should use legacy method when modern API is not available', async () => {
            mockExecCommand.mockReturnValue(true);

            expect(await copyToClipboard('test text')).toEqual({ success: true, method: 'legacy' });
            expect(mockExecCommand).toHaveBeenCalledWith('copy');
            expect(document.querySelector('textarea')).toBeNull();
        });

        it.each([false, true])('should report legacy failure (modern API available: %s)', async (modernAvailable) => {
            if (modernAvailable) {
                mockProperty(navigator, 'clipboard', { writeText: mockWriteText });
                mockProperty(window, 'isSecureContext', true);
                mockWriteText.mockRejectedValue(new Error('Permission denied'));
            }
            mockExecCommand.mockReturnValue(false);

            expect(await copyToClipboard('test text')).toEqual({
                success: false, error: 'execCommand_failed', method: 'manual',
            });
            expect(mockExecCommand).toHaveBeenCalledWith('copy');
            expect(document.querySelector('textarea')).toBeNull();
        });
    });

    describe('isClipboardSupported', () => {
        it('should return true when modern clipboard API is available', () => {
            mockProperty(navigator, 'clipboard', { writeText: mockWriteText });
            mockProperty(window, 'isSecureContext', true);
            mockProperty(document, 'execCommand', undefined);
            expect(isClipboardSupported()).toBe(true);
        });

        it('should return true when legacy method is supported', () => {
            expect(isClipboardSupported()).toBe(true);
        });

        it('should return false when no clipboard support is available', () => {
            mockProperty(document, 'execCommand', undefined);
            expect(isClipboardSupported()).toBe(false);
        });
    });

    describe('getClipboardErrorMessage', () => {
        it('should return correct error message for execCommand_failed', () => {
            expect(getClipboardErrorMessage('execCommand_failed')).toBe('clipboardError.execCommandFailed');
        });

        it('should return correct error message for clipboard_not_supported', () => {
            expect(getClipboardErrorMessage('clipboard_not_supported')).toBe('clipboardError.notSupported');
        });

        it('should return default error message for unknown error', () => {
            expect(getClipboardErrorMessage('unknown_error')).toBe('clipboardError.accessDenied');
            expect(getClipboardErrorMessage()).toBe('clipboardError.accessDenied');
        });
    });
});
