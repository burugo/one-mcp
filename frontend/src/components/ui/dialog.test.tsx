import React from 'react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { fireEvent, screen, waitFor } from '@testing-library/react'
import { render } from '@/__tests__/utils/test-utils'
import {
    Dialog,
    DialogTrigger,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogDescription,
    DialogFooter,
    DialogClose,
} from './dialog'

// Mock lucide-react
vi.mock('lucide-react', () => ({
    X: () => <div data-testid="close-icon">×</div>,
}))

describe('Dialog Components', () => {
    beforeEach(() => {
        // Clear any existing modals
        document.body.innerHTML = ''
    })

    describe('Dialog Basic Functionality', () => {
        it('should render dialog trigger', () => {
            render(
                <Dialog>
                    <DialogTrigger asChild>
                        <button>Open Dialog</button>
                    </DialogTrigger>
                </Dialog>
            )

            expect(screen.getByRole('button', { name: 'Open Dialog' })).toBeInTheDocument()
        })

        it('should open dialog when trigger is clicked', async () => {
            render(
                <Dialog>
                    <DialogTrigger asChild>
                        <button>Open Dialog</button>
                    </DialogTrigger>
                    <DialogContent>
                        <DialogTitle>Test Dialog</DialogTitle>
                        <DialogDescription>This is a test dialog</DialogDescription>
                    </DialogContent>
                </Dialog>
            )

            const trigger = screen.getByRole('button', { name: 'Open Dialog' })
            fireEvent.click(trigger)

            await waitFor(() => {
                expect(screen.getByRole('dialog')).toBeInTheDocument()
                expect(screen.getByText('Test Dialog')).toBeInTheDocument()
                expect(screen.getByText('This is a test dialog')).toBeInTheDocument()
            })
        })

        it('should close dialog when close button is clicked', async () => {
            render(
                <Dialog>
                    <DialogTrigger asChild>
                        <button>Open Dialog</button>
                    </DialogTrigger>
                    <DialogContent>
                        <DialogTitle>Test Dialog</DialogTitle>
                        <DialogDescription>This is a test dialog</DialogDescription>
                    </DialogContent>
                </Dialog>
            )

            // Open dialog
            const trigger = screen.getByRole('button', { name: 'Open Dialog' })
            fireEvent.click(trigger)

            await waitFor(() => {
                expect(screen.getByRole('dialog')).toBeInTheDocument()
            })

            // Close dialog
            const closeButton = screen.getByRole('button', { name: /Close/i })
            fireEvent.click(closeButton)

            await waitFor(() => {
                expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
            })
        })

        it('should close dialog when escape key is pressed', async () => {
            render(
                <Dialog>
                    <DialogTrigger asChild>
                        <button>Open Dialog</button>
                    </DialogTrigger>
                    <DialogContent>
                        <DialogTitle>Test Dialog</DialogTitle>
                        <DialogDescription>This is a test dialog</DialogDescription>
                    </DialogContent>
                </Dialog>
            )

            // Open dialog
            const trigger = screen.getByRole('button', { name: 'Open Dialog' })
            fireEvent.click(trigger)

            await waitFor(() => {
                expect(screen.getByRole('dialog')).toBeInTheDocument()
            })

            // Press escape
            fireEvent.keyDown(document, { key: 'Escape' })

            await waitFor(() => {
                expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
            })
        })
    })

    describe('DialogContent', () => {
        it('should apply default classes', async () => {
            render(
                <Dialog defaultOpen>
                    <DialogContent data-testid="dialog-content">
                        <DialogTitle>Test Dialog</DialogTitle>
                        <DialogDescription>Some accessibility description</DialogDescription>
                    </DialogContent>
                </Dialog>
            )

            await waitFor(() => {
                const content = screen.getByTestId('dialog-content')
                expect(content).toHaveClass(
                    'fixed',
                    'left-[50%]',
                    'top-[50%]',
                    'z-50',
                    'grid',
                    'w-full',
                    'max-w-lg'
                )
            })
        })

        it('should apply custom className', async () => {
            render(
                <Dialog defaultOpen>
                    <DialogContent className="custom-dialog" data-testid="dialog-content">
                        <DialogTitle>Test Dialog</DialogTitle>
                        <DialogDescription>Some accessibility description</DialogDescription>
                    </DialogContent>
                </Dialog>
            )

            await waitFor(() => {
                const content = screen.getByTestId('dialog-content')
                expect(content).toHaveClass('custom-dialog')
            })
        })

        it('should render close icon', async () => {
            render(
                <Dialog defaultOpen>
                    <DialogContent>
                        <DialogTitle>Test Dialog</DialogTitle>
                        <DialogDescription>Some accessibility description</DialogDescription>
                    </DialogContent>
                </Dialog>
            )

            await waitFor(() => {
                expect(screen.getByTestId('close-icon')).toBeInTheDocument()
            })
        })
    })

    describe('DialogHeader', () => {
        it('should render dialog header', () => {
            render(
                <Dialog defaultOpen>
                    <DialogContent>
                        <DialogHeader data-testid="dialog-header">
                            <DialogTitle>Header Title</DialogTitle>
                            <DialogDescription>Header Description</DialogDescription>
                        </DialogHeader>
                    </DialogContent>
                </Dialog>
            )

            const header = screen.getByTestId('dialog-header')
            expect(header).toBeInTheDocument()
            expect(header).toHaveTextContent('Header Title')
            expect(header).toHaveTextContent('Header Description')
        })

        it('should apply default classes', () => {
            render(
                <DialogHeader data-testid="dialog-header">
                    Header content
                </DialogHeader>
            )

            const header = screen.getByTestId('dialog-header')
            expect(header).toHaveClass(
                'flex',
                'flex-col',
                'space-y-1.5',
                'text-center',
                'sm:text-left'
            )
        })

        it('should apply custom className', () => {
            render(
                <DialogHeader className="custom-header" data-testid="dialog-header">
                    Header content
                </DialogHeader>
            )

            const header = screen.getByTestId('dialog-header')
            expect(header).toHaveClass('custom-header')
        })
    })

    describe('DialogTitle', () => {
        it('should render dialog title', () => {
            render(
                <Dialog defaultOpen>
                    <DialogContent>
                        <DialogTitle data-testid="dialog-title">Test Title</DialogTitle>
                        <DialogDescription>Some accessibility description</DialogDescription>
                    </DialogContent>
                </Dialog>
            )

            const title = screen.getByTestId('dialog-title')
            expect(title).toBeInTheDocument()
            expect(title).toHaveTextContent('Test Title')
        })

        it('should apply default classes', () => {
            render(
                <Dialog defaultOpen>
                    <DialogContent>
                        <DialogTitle data-testid="dialog-title">Test Title</DialogTitle>
                        <DialogDescription>Some accessibility description</DialogDescription>
                    </DialogContent>
                </Dialog>
            )

            const title = screen.getByTestId('dialog-title')
            expect(title).toHaveClass(
                'text-lg',
                'font-semibold',
                'leading-none',
                'tracking-tight'
            )
        })

        it('should apply custom className', () => {
            render(
                <Dialog defaultOpen>
                    <DialogContent>
                        <DialogTitle className="custom-title" data-testid="dialog-title">
                            Test Title
                        </DialogTitle>
                        <DialogDescription>Some accessibility description</DialogDescription>
                    </DialogContent>
                </Dialog>
            )

            const title = screen.getByTestId('dialog-title')
            expect(title).toHaveClass('custom-title')
        })
    })

    describe('DialogDescription', () => {
        it('should render dialog description', () => {
            render(
                <Dialog defaultOpen>
                    <DialogContent>
                        <DialogTitle>Some accessibility title</DialogTitle>
                        <DialogDescription data-testid="dialog-description">
                            Test Description
                        </DialogDescription>
                    </DialogContent>
                </Dialog>
            )

            const description = screen.getByTestId('dialog-description')
            expect(description).toBeInTheDocument()
            expect(description).toHaveTextContent('Test Description')
        })

        it('should apply default classes', () => {
            render(
                <Dialog defaultOpen>
                    <DialogContent>
                        <DialogTitle>Some accessibility title</DialogTitle>
                        <DialogDescription data-testid="dialog-description">
                            Test Description
                        </DialogDescription>
                    </DialogContent>
                </Dialog>
            )

            const description = screen.getByTestId('dialog-description')
            expect(description).toHaveClass(
                'text-sm',
                'text-muted-foreground'
            )
        })

        it('should apply custom className', () => {
            render(
                <Dialog defaultOpen>
                    <DialogContent>
                        <DialogTitle>Some accessibility title</DialogTitle>
                        <DialogDescription
                            className="custom-description"
                            data-testid="dialog-description"
                        >
                            Test Description
                        </DialogDescription>
                    </DialogContent>
                </Dialog>
            )

            const description = screen.getByTestId('dialog-description')
            expect(description).toHaveClass('custom-description')
        })
    })

    describe('DialogFooter', () => {
        it('should render dialog footer', () => {
            render(
                <DialogFooter data-testid="dialog-footer">
                    <button>Cancel</button>
                    <button>Confirm</button>
                </DialogFooter>
            )

            const footer = screen.getByTestId('dialog-footer')
            expect(footer).toBeInTheDocument()
            expect(screen.getByRole('button', { name: 'Cancel' })).toBeInTheDocument()
            expect(screen.getByRole('button', { name: 'Confirm' })).toBeInTheDocument()
        })

        it('should apply default classes', () => {
            render(
                <DialogFooter data-testid="dialog-footer">
                    Footer content
                </DialogFooter>
            )

            const footer = screen.getByTestId('dialog-footer')
            expect(footer).toHaveClass(
                'flex',
                'flex-col-reverse',
                'sm:flex-row',
                'sm:justify-end',
                'sm:space-x-2'
            )
        })

        it('should apply custom className', () => {
            render(
                <DialogFooter className="custom-footer" data-testid="dialog-footer">
                    Footer content
                </DialogFooter>
            )

            const footer = screen.getByTestId('dialog-footer')
            expect(footer).toHaveClass('custom-footer')
        })
    })

    describe('DialogClose', () => {
        it('should close dialog when DialogClose is clicked', async () => {
            const onOpenChange = vi.fn()
            render(
                <Dialog open={true} onOpenChange={onOpenChange}>
                    <DialogContent>
                        <DialogTitle>Test Dialog</DialogTitle>
                        <DialogDescription>Content</DialogDescription>
                        <DialogFooter>
                            <DialogClose asChild>
                                <button>Close Me</button>
                            </DialogClose>
                        </DialogFooter>
                    </DialogContent>
                </Dialog>
            )

            const closeButton = screen.getByRole('button', { name: 'Close Me' })
            fireEvent.click(closeButton)

            await waitFor(() => {
                expect(onOpenChange).toHaveBeenCalledWith(false)
            })
        })
    })

    describe('Dialog Composition', () => {
        it('should render complete dialog structure', async () => {
            render(
                <Dialog>
                    <DialogTrigger asChild>
                        <button>Open Complete Dialog</button>
                    </DialogTrigger>
                    <DialogContent data-testid="complete-dialog">
                        <DialogHeader>
                            <DialogTitle>Complete Dialog Title</DialogTitle>
                            <DialogDescription>
                                This is a complete dialog with all components.
                            </DialogDescription>
                        </DialogHeader>
                        <div className="py-4">
                            <p>Dialog body content goes here.</p>
                        </div>
                        <DialogFooter>
                            <DialogClose asChild>
                                <button>Cancel</button>
                            </DialogClose>
                            <button>Save Changes</button>
                        </DialogFooter>
                    </DialogContent>
                </Dialog>
            )

            // Open dialog
            const trigger = screen.getByRole('button', { name: 'Open Complete Dialog' })
            fireEvent.click(trigger)

            await waitFor(() => {
                expect(screen.getByTestId('complete-dialog')).toBeInTheDocument()
                expect(screen.getByText('Complete Dialog Title')).toBeInTheDocument()
                expect(screen.getByText('This is a complete dialog with all components.')).toBeInTheDocument()
                expect(screen.getByText('Dialog body content goes here.')).toBeInTheDocument()
                expect(screen.getByRole('button', { name: 'Cancel' })).toBeInTheDocument()
                expect(screen.getByRole('button', { name: 'Save Changes' })).toBeInTheDocument()
            })
        })

        it('should handle controlled dialog state', async () => {
            const TestComponent = () => {
                const [open, setOpen] = React.useState(false)

                return (
                    <div>
                        <button onClick={() => setOpen(true)}>Open Controlled Dialog</button>
                        <Dialog open={open} onOpenChange={setOpen}>
                            <DialogContent>
                                <DialogTitle>Controlled Dialog</DialogTitle>
                                <DialogDescription>This dialog is controlled externally.</DialogDescription>
                                <DialogFooter>
                                    <button onClick={() => setOpen(false)}>Close Programmatically</button>
                                </DialogFooter>
                            </DialogContent>
                        </Dialog>
                    </div>
                )
            }

            render(<TestComponent />)

            // Open dialog programmatically
            const openButton = screen.getByRole('button', { name: 'Open Controlled Dialog' })
            fireEvent.click(openButton)

            await waitFor(() => {
                expect(screen.getByRole('dialog')).toBeInTheDocument()
            })

            // Close dialog programmatically
            const closeButton = screen.getByRole('button', { name: 'Close Programmatically' })
            fireEvent.click(closeButton)

            await waitFor(() => {
                expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
            })
        })
    })
}) 