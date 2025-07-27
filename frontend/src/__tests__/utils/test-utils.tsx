import React, { ReactElement } from 'react'
import { render, RenderOptions } from '@testing-library/react'
import { BrowserRouter } from 'react-router-dom'

// 自定义渲染函数
interface CustomRenderOptions extends Omit<RenderOptions, 'wrapper'> {
    withRouter?: boolean
}

export const customRender = (
    ui: ReactElement,
    options: CustomRenderOptions = {}
) => {
    const { withRouter = false, ...renderOptions } = options

    let Wrapper: React.FC<{ children: React.ReactNode }> = ({ children }) => <>{children}</>

    if (withRouter) {
        Wrapper = ({ children }) => (
            <BrowserRouter>{children}</BrowserRouter>
        )
    }

    return render(ui, { wrapper: Wrapper, ...renderOptions })
}

// Mock数据生成器
export const createMockService = (overrides = {}) => ({
    id: '1',
    name: 'Test Service',
    display_name: 'Test Service',
    description: 'A test service',
    health_status: 'active',
    enabled: true,
    type: 'stdio' as 'stdio' | 'sse' | 'streamableHttp',
    package_manager: 'npm',
    source_package_name: 'test-package',
    client_config_templates: '{}',
    installed_version: '1.0.0',
    version: '1.0.0',
    source: 'npm' as 'npm' | 'pypi' | 'local' | 'recommended',
    envVars: [],
    readme: 'This is a readme',
    command: '',
    headers_json: '',
    ...overrides,
})

export const createMockUser = (overrides = {}) => ({
    id: 1,
    name: 'Test User',
    email: 'test@example.com',
    ...overrides,
})

export const createMockMarketService = (overrides = {}) => ({
    id: '1',
    name: 'Test Market Service',
    display_name: 'Test Market Service',
    description: 'A test market service',
    category: 'AI',
    icon: 'https://example.com/icon.png',
    package_manager: 'npm',
    package_name: 'test-package',
    service_description: 'Test service description',
    service_icon_url: 'https://example.com/service-icon.png',
    ...overrides,
})

// 测试用的环境变量Mock
export const mockEnvVars = {
    FIRECRAWL_API_KEY: 'test-api-key',
    OPENAI_API_KEY: 'test-openai-key',
}

// 重新导出testing-library的所有工具
export * from '@testing-library/react'
export { customRender as render } 