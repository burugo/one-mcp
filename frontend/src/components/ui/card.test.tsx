import { describe, it, expect, vi } from 'vitest'
import { screen } from '@testing-library/react'
import { render } from '@/__tests__/utils/test-utils'
import {
    Card,
    CardHeader,
    CardTitle,
    CardDescription,
    CardContent,
    CardFooter
} from './card'

describe('Card Components', () => {
    describe('Card', () => {
        it('should render card element', () => {
            render(<Card data-testid="card">Card content</Card>)
            const card = screen.getByTestId('card')

            expect(card).toBeInTheDocument()
            expect(card).toHaveTextContent('Card content')
        })

        it('should apply default classes', () => {
            render(<Card data-testid="card">Card content</Card>)
            const card = screen.getByTestId('card')

            expect(card).toHaveClass(
                'rounded-xl',
                'border',
                'bg-card',
                'text-card-foreground',
                'shadow'
            )
        })

        it('should apply custom className', () => {
            render(<Card className="custom-card" data-testid="card">Card content</Card>)
            const card = screen.getByTestId('card')

            expect(card).toHaveClass('custom-card')
        })

        it('should forward ref correctly', () => {
            const ref = vi.fn()
            render(<Card ref={ref} data-testid="card">Card content</Card>)

            expect(ref).toHaveBeenCalled()
        })

        it('should pass through HTML attributes', () => {
            render(
                <Card
                    data-testid="card"
                    id="test-card"
                    role="region"
                    aria-label="Test card"
                >
                    Card content
                </Card>
            )
            const card = screen.getByTestId('card')

            expect(card).toHaveAttribute('id', 'test-card')
            expect(card).toHaveAttribute('role', 'region')
            expect(card).toHaveAttribute('aria-label', 'Test card')
        })
    })

    describe('CardHeader', () => {
        it('should render card header', () => {
            render(<CardHeader data-testid="card-header">Header content</CardHeader>)
            const header = screen.getByTestId('card-header')

            expect(header).toBeInTheDocument()
            expect(header).toHaveTextContent('Header content')
        })

        it('should apply default classes', () => {
            render(<CardHeader data-testid="card-header">Header content</CardHeader>)
            const header = screen.getByTestId('card-header')

            expect(header).toHaveClass(
                'flex',
                'flex-col',
                'space-y-1.5',
                'p-6'
            )
        })

        it('should apply custom className', () => {
            render(<CardHeader className="custom-header" data-testid="card-header">Header</CardHeader>)
            const header = screen.getByTestId('card-header')

            expect(header).toHaveClass('custom-header')
        })

        it('should forward ref correctly', () => {
            const ref = vi.fn()
            render(<CardHeader ref={ref} data-testid="card-header">Header</CardHeader>)

            expect(ref).toHaveBeenCalled()
        })
    })

    describe('CardTitle', () => {
        it('should render card title', () => {
            render(<CardTitle data-testid="card-title">Title content</CardTitle>)
            const title = screen.getByTestId('card-title')

            expect(title).toBeInTheDocument()
            expect(title).toHaveTextContent('Title content')
        })

        it('should apply default classes', () => {
            render(<CardTitle data-testid="card-title">Title content</CardTitle>)
            const title = screen.getByTestId('card-title')

            expect(title).toHaveClass(
                'font-semibold',
                'leading-none',
                'tracking-tight'
            )
        })

        it('should apply custom className', () => {
            render(<CardTitle className="custom-title" data-testid="card-title">Title</CardTitle>)
            const title = screen.getByTestId('card-title')

            expect(title).toHaveClass('custom-title')
        })

        it('should forward ref correctly', () => {
            const ref = vi.fn()
            render(<CardTitle ref={ref} data-testid="card-title">Title</CardTitle>)

            expect(ref).toHaveBeenCalled()
        })
    })

    describe('CardDescription', () => {
        it('should render card description', () => {
            render(<CardDescription data-testid="card-description">Description content</CardDescription>)
            const description = screen.getByTestId('card-description')

            expect(description).toBeInTheDocument()
            expect(description).toHaveTextContent('Description content')
        })

        it('should apply default classes', () => {
            render(<CardDescription data-testid="card-description">Description</CardDescription>)
            const description = screen.getByTestId('card-description')

            expect(description).toHaveClass(
                'text-sm',
                'text-muted-foreground'
            )
        })

        it('should apply custom className', () => {
            render(<CardDescription className="custom-description" data-testid="card-description">Description</CardDescription>)
            const description = screen.getByTestId('card-description')

            expect(description).toHaveClass('custom-description')
        })

        it('should forward ref correctly', () => {
            const ref = vi.fn()
            render(<CardDescription ref={ref} data-testid="card-description">Description</CardDescription>)

            expect(ref).toHaveBeenCalled()
        })
    })

    describe('CardContent', () => {
        it('should render card content', () => {
            render(<CardContent data-testid="card-content">Content area</CardContent>)
            const content = screen.getByTestId('card-content')

            expect(content).toBeInTheDocument()
            expect(content).toHaveTextContent('Content area')
        })

        it('should apply default classes', () => {
            render(<CardContent data-testid="card-content">Content</CardContent>)
            const content = screen.getByTestId('card-content')

            expect(content).toHaveClass('p-6', 'pt-0')
        })

        it('should apply custom className', () => {
            render(<CardContent className="custom-content" data-testid="card-content">Content</CardContent>)
            const content = screen.getByTestId('card-content')

            expect(content).toHaveClass('custom-content')
        })

        it('should forward ref correctly', () => {
            const ref = vi.fn()
            render(<CardContent ref={ref} data-testid="card-content">Content</CardContent>)

            expect(ref).toHaveBeenCalled()
        })
    })

    describe('CardFooter', () => {
        it('should render card footer', () => {
            render(<CardFooter data-testid="card-footer">Footer content</CardFooter>)
            const footer = screen.getByTestId('card-footer')

            expect(footer).toBeInTheDocument()
            expect(footer).toHaveTextContent('Footer content')
        })

        it('should apply default classes', () => {
            render(<CardFooter data-testid="card-footer">Footer</CardFooter>)
            const footer = screen.getByTestId('card-footer')

            expect(footer).toHaveClass(
                'flex',
                'items-center',
                'p-6',
                'pt-0'
            )
        })

        it('should apply custom className', () => {
            render(<CardFooter className="custom-footer" data-testid="card-footer">Footer</CardFooter>)
            const footer = screen.getByTestId('card-footer')

            expect(footer).toHaveClass('custom-footer')
        })

        it('should forward ref correctly', () => {
            const ref = vi.fn()
            render(<CardFooter ref={ref} data-testid="card-footer">Footer</CardFooter>)

            expect(ref).toHaveBeenCalled()
        })
    })

    describe('Card Composition', () => {
        it('should render complete card structure', () => {
            render(
                <Card data-testid="complete-card">
                    <CardHeader data-testid="header">
                        <CardTitle data-testid="title">Card Title</CardTitle>
                        <CardDescription data-testid="description">Card Description</CardDescription>
                    </CardHeader>
                    <CardContent data-testid="content">
                        <p>This is the main content of the card.</p>
                    </CardContent>
                    <CardFooter data-testid="footer">
                        <button>Action Button</button>
                    </CardFooter>
                </Card>
            )

            expect(screen.getByTestId('complete-card')).toBeInTheDocument()
            expect(screen.getByTestId('header')).toBeInTheDocument()
            expect(screen.getByTestId('title')).toHaveTextContent('Card Title')
            expect(screen.getByTestId('description')).toHaveTextContent('Card Description')
            expect(screen.getByTestId('content')).toHaveTextContent('This is the main content of the card.')
            expect(screen.getByTestId('footer')).toBeInTheDocument()
            expect(screen.getByRole('button', { name: 'Action Button' })).toBeInTheDocument()
        })

        it('should maintain proper hierarchy and styling', () => {
            render(
                <Card data-testid="styled-card" className="w-96">
                    <CardHeader>
                        <CardTitle>Styled Card</CardTitle>
                        <CardDescription>This card has custom styling</CardDescription>
                    </CardHeader>
                    <CardContent>
                        Content with proper spacing
                    </CardContent>
                    <CardFooter className="justify-end">
                        <button>Save</button>
                        <button>Cancel</button>
                    </CardFooter>
                </Card>
            )

            const card = screen.getByTestId('styled-card')
            expect(card).toHaveClass('w-96')

            const footer = screen.getByText('Save').closest('div')
            expect(footer).toHaveClass('justify-end')
        })
    })
}) 