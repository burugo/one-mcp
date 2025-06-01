import React from 'react'
import { describe, it, expect, vi } from 'vitest'
import { fireEvent, screen } from '@testing-library/react'
import { render } from '@/__tests__/utils/test-utils'
import { Input } from './input'

describe('Input Component', () => {
    it('should render input element', () => {
        render(<Input placeholder="Enter text" />)
        expect(screen.getByPlaceholderText('Enter text')).toBeInTheDocument()
    })

    it('should handle text input', () => {
        render(<Input placeholder="Enter text" />)
        const input = screen.getByPlaceholderText('Enter text')

        fireEvent.change(input, { target: { value: 'Hello World' } })
        expect(input).toHaveValue('Hello World')
    })

    it('should call onChange when value changes', () => {
        const handleChange = vi.fn()
        render(<Input onChange={handleChange} placeholder="Enter text" />)

        const input = screen.getByPlaceholderText('Enter text')
        fireEvent.change(input, { target: { value: 'test' } })

        expect(handleChange).toHaveBeenCalledTimes(1)
    })

    it('should be disabled when disabled prop is true', () => {
        render(<Input disabled placeholder="Disabled input" />)
        const input = screen.getByPlaceholderText('Disabled input')

        expect(input).toBeDisabled()
        expect(input).toHaveClass('disabled:cursor-not-allowed', 'disabled:opacity-50')
    })

    it('should apply custom className', () => {
        render(<Input className="custom-input" placeholder="Custom input" />)
        expect(screen.getByPlaceholderText('Custom input')).toHaveClass('custom-input')
    })

    it('should render different input types', () => {
        const { rerender } = render(<Input type="email" placeholder="Email" />)
        expect(screen.getByPlaceholderText('Email')).toHaveAttribute('type', 'email')

        rerender(<Input type="password" placeholder="Password" />)
        expect(screen.getByPlaceholderText('Password')).toHaveAttribute('type', 'password')

        rerender(<Input type="number" placeholder="Number" />)
        expect(screen.getByPlaceholderText('Number')).toHaveAttribute('type', 'number')
    })

    it('should forward ref correctly', () => {
        const ref = vi.fn()
        render(<Input ref={ref} placeholder="Ref input" />)
        expect(ref).toHaveBeenCalled()
    })

    it('should handle focus and blur events', () => {
        const handleFocus = vi.fn()
        const handleBlur = vi.fn()

        render(
            <Input
                onFocus={handleFocus}
                onBlur={handleBlur}
                placeholder="Focus test"
            />
        )

        const input = screen.getByPlaceholderText('Focus test')

        fireEvent.focus(input)
        expect(handleFocus).toHaveBeenCalledTimes(1)

        fireEvent.blur(input)
        expect(handleBlur).toHaveBeenCalledTimes(1)
    })

    it('should have correct default classes', () => {
        render(<Input placeholder="Default classes" />)
        const input = screen.getByPlaceholderText('Default classes')

        expect(input).toHaveClass(
            'flex',
            'h-9',
            'w-full',
            'rounded-md',
            'border',
            'border-input',
            'bg-transparent',
            'px-3',
            'py-1',
            'text-base',
            'shadow-sm',
            'transition-colors'
        )
    })

    it('should handle keyboard events', () => {
        const handleKeyDown = vi.fn()
        render(<Input onKeyDown={handleKeyDown} placeholder="Keyboard test" />)

        const input = screen.getByPlaceholderText('Keyboard test')
        fireEvent.keyDown(input, { key: 'Enter' })

        expect(handleKeyDown).toHaveBeenCalledTimes(1)
    })

    it('should support controlled input', () => {
        const TestComponent = () => {
            const [value, setValue] = React.useState('')
            return (
                <Input
                    value={value}
                    onChange={(e) => setValue(e.target.value)}
                    placeholder="Controlled input"
                />
            )
        }

        render(<TestComponent />)
        const input = screen.getByPlaceholderText('Controlled input')

        fireEvent.change(input, { target: { value: 'controlled' } })
        expect(input).toHaveValue('controlled')
    })

    it('should handle file input type', () => {
        render(<Input type="file" data-testid="file-input" />)
        const input = screen.getByTestId('file-input')

        expect(input).toHaveAttribute('type', 'file')
        expect(input).toHaveClass('file:border-0', 'file:bg-transparent', 'file:text-sm')
    })
}) 