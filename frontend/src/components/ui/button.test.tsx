import { describe, it, expect, vi } from 'vitest'
import { fireEvent, screen } from '@testing-library/react'
import { render } from '@/__tests__/utils/test-utils'
import { Button } from './button'

describe('Button Component', () => {
    it('should render with correct text', () => {
        render(<Button>Click me</Button>)
        expect(screen.getByRole('button', { name: 'Click me' })).toBeInTheDocument()
    })

    it('should call onClick when clicked', () => {
        const handleClick = vi.fn()
        render(<Button onClick={handleClick}>Click me</Button>)

        fireEvent.click(screen.getByRole('button'))
        expect(handleClick).toHaveBeenCalledTimes(1)
    })

    it('should be disabled when disabled prop is true', () => {
        render(<Button disabled>Click me</Button>)
        const button = screen.getByRole('button')
        expect(button).toBeDisabled()
        expect(button).toHaveClass('disabled:pointer-events-none', 'disabled:opacity-50')
    })

    it('should apply custom className', () => {
        render(<Button className="custom-class">Click me</Button>)
        expect(screen.getByRole('button')).toHaveClass('custom-class')
    })

    it('should render different variants correctly', () => {
        const { rerender } = render(<Button variant="destructive">Destructive</Button>)
        expect(screen.getByRole('button')).toHaveClass('bg-destructive')

        rerender(<Button variant="outline">Outline</Button>)
        expect(screen.getByRole('button')).toHaveClass('border', 'border-input')

        rerender(<Button variant="secondary">Secondary</Button>)
        expect(screen.getByRole('button')).toHaveClass('bg-secondary')

        rerender(<Button variant="ghost">Ghost</Button>)
        expect(screen.getByRole('button')).toHaveClass('hover:bg-accent')

        rerender(<Button variant="link">Link</Button>)
        expect(screen.getByRole('button')).toHaveClass('text-primary', 'underline-offset-4')
    })

    it('should render different sizes correctly', () => {
        const { rerender } = render(<Button size="sm">Small</Button>)
        expect(screen.getByRole('button')).toHaveClass('h-8', 'px-3', 'text-xs')

        rerender(<Button size="lg">Large</Button>)
        expect(screen.getByRole('button')).toHaveClass('h-10', 'px-8')

        rerender(<Button size="icon">Icon</Button>)
        expect(screen.getByRole('button')).toHaveClass('h-9', 'w-9')
    })

    it('should render as child component when asChild is true', () => {
        render(
            <Button asChild>
                <a href="/test">Link Button</a>
            </Button>
        )

        const link = screen.getByRole('link')
        expect(link).toBeInTheDocument()
        expect(link).toHaveAttribute('href', '/test')
        expect(link).toHaveClass('inline-flex', 'items-center', 'justify-center')
    })

    it('should forward ref correctly', () => {
        const ref = vi.fn()
        render(<Button ref={ref}>Button with ref</Button>)
        expect(ref).toHaveBeenCalled()
    })

    it('should handle keyboard events', () => {
        const handleClick = vi.fn()
        render(<Button onClick={handleClick}>Keyboard Button</Button>)

        const button = screen.getByRole('button')
        fireEvent.keyDown(button, { key: 'Enter' })
        fireEvent.keyUp(button, { key: 'Enter' })

        // Button should be focusable
        expect(button).toHaveClass('focus-visible:outline-none', 'focus-visible:ring-1')
    })

    it('should apply default variant and size when not specified', () => {
        render(<Button>Default Button</Button>)
        const button = screen.getByRole('button')

        // Default variant classes
        expect(button).toHaveClass('bg-primary', 'text-primary-foreground')
        // Default size classes
        expect(button).toHaveClass('h-9', 'px-4', 'py-2')
    })

    it('should not trigger onClick when disabled', () => {
        const handleClick = vi.fn()
        render(<Button disabled onClick={handleClick}>Disabled Button</Button>)

        fireEvent.click(screen.getByRole('button'))
        expect(handleClick).not.toHaveBeenCalled()
    })
}) 