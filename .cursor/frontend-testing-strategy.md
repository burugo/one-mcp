# 前端测试体系构建方案 - Vitest + Playwright

## Background and Motivation

构建一套完整的前端测试体系，使用现代化的测试工具栈（Vitest + React Testing Library + Playwright），通过测试金字塔模型有效减少bug，提高代码质量和开发效率。

## 测试金字塔架构

```
    /\
   /  \     E2E Tests (Playwright)
  /____\    - 用户流程测试
 /      \   - 跨页面功能测试
/________\  Integration Tests (Vitest + RTL)
          \ - 组件协作测试
           \- API交互测试
____________\
            Unit Tests (Vitest + RTL)
            - 组件单元测试
            - 工具函数测试
            - Hook测试
```

### 测试比例建议
- **单元测试**: 70% (快速、稳定、易维护)
- **集成测试**: 20% (关键业务流程)
- **E2E测试**: 10% (核心用户路径)

## 工具栈选择

### 1. Vitest (替代Jest)
**优势**:
- 与Vite生态完美集成
- 启动速度快，HMR支持
- 原生ESM和TypeScript支持
- 兼容Jest API，迁移成本低

### 2. React Testing Library
**优势**:
- 鼓励测试用户行为而非实现细节
- 良好的可访问性支持
- 社区活跃，最佳实践丰富

### 3. Playwright
**优势**:
- 支持多浏览器(Chrome, Firefox, Safari)
- 强大的调试工具
- 自动等待机制
- 并行执行能力

## 项目结构设计

```
src/
├── components/
│   ├── Button/
│   │   ├── Button.tsx
│   │   ├── Button.test.tsx          # 单元测试
│   │   └── Button.stories.tsx       # Storybook (可选)
│   └── UserProfile/
│       ├── UserProfile.tsx
│       ├── UserProfile.test.tsx     # 单元测试
│       └── UserProfile.integration.test.tsx # 集成测试
├── hooks/
│   ├── useAuth.ts
│   └── useAuth.test.ts              # Hook测试
├── utils/
│   ├── formatters.ts
│   └── formatters.test.ts           # 工具函数测试
├── pages/
│   ├── LoginPage/
│   │   ├── LoginPage.tsx
│   │   └── LoginPage.integration.test.tsx
│   └── ServicesPage/
│       ├── ServicesPage.tsx
│       └── ServicesPage.integration.test.tsx
├── __tests__/
│   ├── setup.ts                     # 测试环境配置
│   └── utils/                       # 测试工具函数
└── e2e/                            # E2E测试目录
    ├── auth.spec.ts
    ├── services.spec.ts
    └── fixtures/                    # 测试数据
```

## 配置文件

### 1. Vitest配置 (vite.config.ts)

```typescript
/// <reference types="vitest" />
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/__tests__/setup.ts'],
    css: true,
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html'],
      exclude: [
        'node_modules/',
        'src/__tests__/',
        'src/**/*.test.{ts,tsx}',
        'src/**/*.spec.{ts,tsx}',
        'src/**/*.stories.{ts,tsx}',
        'e2e/',
        '**/*.d.ts',
      ],
      thresholds: {
        global: {
          branches: 80,
          functions: 80,
          lines: 80,
          statements: 80,
        },
      },
    },
  },
})
```

### 2. 测试环境配置 (src/__tests__/setup.ts)

```typescript
import '@testing-library/jest-dom/vitest'
import { cleanup } from '@testing-library/react'
import { afterEach, beforeAll, vi } from 'vitest'

// 每个测试后清理
afterEach(() => {
  cleanup()
})

// Mock全局对象
beforeAll(() => {
  // Mock window.matchMedia
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    value: vi.fn().mockImplementation(query => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  })

  // Mock IntersectionObserver
  global.IntersectionObserver = vi.fn().mockImplementation(() => ({
    observe: vi.fn(),
    unobserve: vi.fn(),
    disconnect: vi.fn(),
  }))

  // Mock ResizeObserver
  global.ResizeObserver = vi.fn().mockImplementation(() => ({
    observe: vi.fn(),
    unobserve: vi.fn(),
    disconnect: vi.fn(),
  }))
})
```

### 3. Playwright配置 (playwright.config.ts)

```typescript
import { defineConfig, devices } from '@playwright/test'

export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: 'html',
  use: {
    baseURL: 'http://localhost:3000',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
    {
      name: 'firefox',
      use: { ...devices['Desktop Firefox'] },
    },
    {
      name: 'webkit',
      use: { ...devices['Desktop Safari'] },
    },
    // Mobile测试
    {
      name: 'Mobile Chrome',
      use: { ...devices['Pixel 5'] },
    },
  ],
  webServer: {
    command: 'npm run dev',
    url: 'http://localhost:3000',
    reuseExistingServer: !process.env.CI,
  },
})
```

## 测试策略详解

### 1. 单元测试策略

**测试内容**:
- 组件渲染和props传递
- 用户交互事件
- 条件渲染逻辑
- 工具函数和Hook

**示例 - 组件单元测试**:
```typescript
// src/components/Button/Button.test.tsx
import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import Button from './Button'

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
    expect(screen.getByRole('button')).toBeDisabled()
  })

  it('should apply custom className', () => {
    render(<Button className="custom-class">Click me</Button>)
    expect(screen.getByRole('button')).toHaveClass('custom-class')
  })
})
```

**示例 - Hook测试**:
```typescript
// src/hooks/useAuth.test.ts
import { renderHook, act } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { useAuth } from './useAuth'

// Mock API
vi.mock('@/utils/api', () => ({
  login: vi.fn(),
  logout: vi.fn(),
}))

describe('useAuth Hook', () => {
  it('should initialize with logged out state', () => {
    const { result } = renderHook(() => useAuth())
    
    expect(result.current.isAuthenticated).toBe(false)
    expect(result.current.user).toBeNull()
  })

  it('should login successfully', async () => {
    const mockUser = { id: 1, name: 'John Doe' }
    const { login } = await import('@/utils/api')
    vi.mocked(login).mockResolvedValue(mockUser)

    const { result } = renderHook(() => useAuth())

    await act(async () => {
      await result.current.login('user@example.com', 'password')
    })

    expect(result.current.isAuthenticated).toBe(true)
    expect(result.current.user).toEqual(mockUser)
  })
})
```

### 2. 集成测试策略

**测试内容**:
- 多组件协作
- 状态管理交互
- API调用和数据流
- 路由导航

**示例 - 页面集成测试**:
```typescript
// src/pages/ServicesPage/ServicesPage.integration.test.tsx
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { BrowserRouter } from 'react-router-dom'
import ServicesPage from './ServicesPage'

// Mock API
vi.mock('@/utils/api', () => ({
  get: vi.fn(),
  post: vi.fn(),
}))

// Mock store
vi.mock('@/store/marketStore', () => ({
  useMarketStore: vi.fn(),
}))

const renderWithRouter = (component: React.ReactElement) => {
  return render(<BrowserRouter>{component}</BrowserRouter>)
}

describe('ServicesPage Integration', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('should display services and handle uninstall flow', async () => {
    const mockServices = [
      { id: '1', name: 'Service 1', health_status: 'active' },
      { id: '2', name: 'Service 2', health_status: 'inactive' },
    ]

    const mockStore = {
      installedServices: mockServices,
      fetchInstalledServices: vi.fn(),
      uninstallService: vi.fn().mockResolvedValue(undefined),
    }

    vi.mocked(useMarketStore).mockReturnValue(mockStore)

    renderWithRouter(<ServicesPage />)

    // 验证服务列表显示
    expect(screen.getByText('Service 1')).toBeInTheDocument()
    expect(screen.getByText('Service 2')).toBeInTheDocument()

    // 测试卸载流程
    const uninstallButtons = screen.getAllByTitle('卸载服务')
    fireEvent.click(uninstallButtons[0])

    // 确认对话框出现
    expect(screen.getByText('确认卸载')).toBeInTheDocument()
    
    // 确认卸载
    fireEvent.click(screen.getByText('卸载'))

    // 验证API调用
    await waitFor(() => {
      expect(mockStore.uninstallService).toHaveBeenCalledWith('1')
    })
  })
})
```

### 3. E2E测试策略

**测试内容**:
- 完整用户流程
- 跨页面导航
- 真实API交互
- 浏览器兼容性

**示例 - E2E测试**:
```typescript
// e2e/services.spec.ts
import { test, expect } from '@playwright/test'

test.describe('Services Management', () => {
  test.beforeEach(async ({ page }) => {
    // 登录
    await page.goto('/login')
    await page.fill('[data-testid="email"]', 'test@example.com')
    await page.fill('[data-testid="password"]', 'password')
    await page.click('[data-testid="login-button"]')
    await expect(page).toHaveURL('/services')
  })

  test('should install and uninstall a service', async ({ page }) => {
    // 导航到市场页面
    await page.click('[data-testid="add-service-button"]')
    await page.click('text=从市场安装')
    await expect(page).toHaveURL('/market')

    // 搜索服务
    await page.fill('[data-testid="search-input"]', 'exa-mcp-server')
    await page.press('[data-testid="search-input"]', 'Enter')

    // 等待搜索结果
    await expect(page.locator('[data-testid="search-results"]')).toBeVisible()

    // 安装服务
    await page.click('[data-testid="install-button"]').first()
    
    // 填写环境变量
    await page.fill('[data-testid="env-FIRECRAWL_API_KEY"]', 'test-api-key')
    await page.click('[data-testid="confirm-install"]')

    // 等待安装完成
    await expect(page.locator('text=安装成功')).toBeVisible()

    // 返回服务页面
    await page.goto('/services')

    // 验证服务已安装
    await expect(page.locator('text=exa-mcp-server')).toBeVisible()

    // 卸载服务
    await page.click('[data-testid="uninstall-exa-mcp-server"]')
    await page.click('text=卸载')

    // 验证卸载成功
    await expect(page.locator('text=卸载成功')).toBeVisible()
    await expect(page.locator('text=exa-mcp-server')).not.toBeVisible()
  })

  test('should handle service configuration', async ({ page }) => {
    // 假设已有服务
    await expect(page.locator('[data-testid="service-card"]').first()).toBeVisible()

    // 打开配置
    await page.click('[data-testid="configure-button"]').first()

    // 修改环境变量
    await page.fill('[data-testid="env-var-input"]', 'new-value')
    await page.click('[data-testid="save-config"]')

    // 验证保存成功
    await expect(page.locator('text=配置已保存')).toBeVisible()
  })
})
```

## 测试工具函数

```typescript
// src/__tests__/utils/test-utils.tsx
import React, { ReactElement } from 'react'
import { render, RenderOptions } from '@testing-library/react'
import { BrowserRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'

// 创建测试用的QueryClient
const createTestQueryClient = () => new QueryClient({
  defaultOptions: {
    queries: { retry: false },
    mutations: { retry: false },
  },
})

// 自定义渲染函数
interface CustomRenderOptions extends Omit<RenderOptions, 'wrapper'> {
  withRouter?: boolean
  withQueryClient?: boolean
}

export const customRender = (
  ui: ReactElement,
  options: CustomRenderOptions = {}
) => {
  const { withRouter = false, withQueryClient = false, ...renderOptions } = options

  let Wrapper: React.FC<{ children: React.ReactNode }> = ({ children }) => <>{children}</>

  if (withRouter) {
    const RouterWrapper = Wrapper
    Wrapper = ({ children }) => (
      <BrowserRouter>
        <RouterWrapper>{children}</RouterWrapper>
      </BrowserRouter>
    )
  }

  if (withQueryClient) {
    const QueryWrapper = Wrapper
    const queryClient = createTestQueryClient()
    Wrapper = ({ children }) => (
      <QueryClientProvider client={queryClient}>
        <QueryWrapper>{children}</QueryWrapper>
      </QueryClientProvider>
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
  ...overrides,
})

export const createMockUser = (overrides = {}) => ({
  id: 1,
  name: 'Test User',
  email: 'test@example.com',
  ...overrides,
})
```

## Package.json脚本配置

```json
{
  "scripts": {
    "test": "vitest",
    "test:ui": "vitest --ui",
    "test:run": "vitest run",
    "test:coverage": "vitest run --coverage",
    "test:watch": "vitest --watch",
    "test:e2e": "playwright test",
    "test:e2e:ui": "playwright test --ui",
    "test:e2e:debug": "playwright test --debug",
    "test:all": "npm run test:run && npm run test:e2e"
  },
  "devDependencies": {
    "@playwright/test": "^1.40.0",
    "@testing-library/jest-dom": "^6.1.0",
    "@testing-library/react": "^13.4.0",
    "@testing-library/user-event": "^14.5.0",
    "@vitest/coverage-v8": "^1.0.0",
    "@vitest/ui": "^1.0.0",
    "jsdom": "^23.0.0",
    "vitest": "^1.0.0"
  }
}
```

## CI/CD集成

### GitHub Actions配置

```yaml
# .github/workflows/test.yml
name: Tests

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: '18'
          cache: 'npm'
      
      - run: npm ci
      - run: npm run test:coverage
      
      - name: Upload coverage to Codecov
        uses: codecov/codecov-action@v3

  e2e-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: '18'
          cache: 'npm'
      
      - run: npm ci
      - run: npx playwright install --with-deps
      - run: npm run build
      - run: npm run test:e2e
      
      - uses: actions/upload-artifact@v3
        if: failure()
        with:
          name: playwright-report
          path: playwright-report/
```

## 实施计划

### Phase 1: 基础设施搭建 (1-2天)
- [ ] 安装和配置Vitest
- [ ] 配置React Testing Library
- [ ] 设置测试环境和工具函数
- [ ] 配置代码覆盖率

### Phase 2: 单元测试 (1周)
- [ ] 为现有组件编写单元测试
- [ ] 为工具函数编写测试
- [ ] 为自定义Hook编写测试
- [ ] 达到80%代码覆盖率

### Phase 3: 集成测试 (3-5天)
- [ ] 为关键页面编写集成测试
- [ ] 测试组件间协作
- [ ] 测试状态管理交互

### Phase 4: E2E测试 (3-5天)
- [ ] 安装和配置Playwright
- [ ] 编写核心用户流程测试
- [ ] 配置多浏览器测试

### Phase 5: CI/CD集成 (1-2天)
- [ ] 配置GitHub Actions
- [ ] 设置自动化测试流程
- [ ] 配置测试报告

## 质量指标

### 代码覆盖率目标
- **语句覆盖率**: ≥ 80%
- **分支覆盖率**: ≥ 80%
- **函数覆盖率**: ≥ 80%
- **行覆盖率**: ≥ 80%

### 测试性能指标
- **单元测试**: < 10秒
- **集成测试**: < 30秒
- **E2E测试**: < 5分钟

### 质量门禁
- 所有测试必须通过
- 代码覆盖率不能下降
- 新功能必须有对应测试

## 最佳实践

1. **测试命名**: 使用描述性的测试名称
2. **AAA模式**: Arrange-Act-Assert
3. **独立性**: 每个测试应该独立运行
4. **可读性**: 测试代码应该易于理解
5. **维护性**: 避免过度Mock，保持测试简单

这套方案将帮助您建立一个健壮的前端测试体系，有效减少bug并提高代码质量。 