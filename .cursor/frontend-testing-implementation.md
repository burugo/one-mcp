# 前端测试实施计划 - Vitest单元测试和集成测试

## Background and Motivation

基于前端测试策略，为现有的React + TypeScript + Zustand项目实施完整的测试体系。项目已经安装了Vitest相关依赖，但仍在使用Jest配置，需要迁移到Vitest并编写全面的测试用例。

## 项目现状分析

### 现有结构
```
frontend/src/
├── components/
│   ├── ui/ (20个UI组件)
│   └── market/ (市场相关组件)
├── pages/ (7个页面组件)
├── hooks/ (2个自定义hooks)
├── store/ (marketStore.ts - Zustand状态管理)
├── utils/ (工具函数)
└── App.tsx (主应用组件)
```

### 已安装依赖
- ✅ Vitest + @vitest/ui + @vitest/coverage-v8
- ✅ @testing-library/react + @testing-library/jest-dom
- ✅ jsdom
- ❌ 仍在使用Jest配置 (需要移除)

## 实施计划

### Phase 1: 配置迁移 (移除Jest，配置Vitest)

#### Task 1.1: 移除Jest配置
- [x] 删除 `jest.config.cjs`
- [x] 删除 `jest.setup.ts`
- [x] 更新 `package.json` 脚本

#### Task 1.2: 配置Vitest
- [x] 更新 `vite.config.ts` 添加测试配置
- [x] 创建 `src/__tests__/setup.ts`
- [x] 配置测试环境和Mock

#### Task 1.3: 创建测试工具函数
- [x] 创建 `src/__tests__/utils/test-utils.tsx`
- [x] 创建Mock数据生成器
- [x] 配置自定义渲染函数

### Phase 2: 单元测试 (优先级高的组件)

#### Task 2.1: UI组件单元测试
- [x] Button组件测试 (11个测试用例)
- [x] Input组件测试 (12个测试用例)
- [x] Card组件测试 (32个测试用例)
- [x] Dialog组件测试 (18个测试用例)

#### Task 2.2: 自定义Hook测试
- [x] useToast hook测试 (14个测试用例)

#### Task 2.3: 工具函数测试
- [x] 探索utils目录并编写测试

### Phase 3: 集成测试 (关键页面和业务流程)

#### Task 3.1: 核心页面集成测试
- [x] ServicesPage集成测试 (10个测试用例)

#### Task 3.2: 状态管理集成测试
- [x] marketStore状态管理测试
- [x] 组件与store交互测试

#### Task 3.3: 路由和导航测试
- [x] App.tsx路由测试
- [x] 页面间导航测试

## 详细实施方案

### 配置文件更新

#### 1. 更新vite.config.ts
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
        '**/*.d.ts',
      ],
      thresholds: {
        global: {
          branches: 70,
          functions: 70,
          lines: 70,
          statements: 70,
        },
      },
    },
  },
})
```

#### 2. 更新package.json脚本
```json
{
  "scripts": {
    "test": "vitest",
    "test:ui": "vitest --ui",
    "test:run": "vitest run",
    "test:coverage": "vitest run --coverage",
    "test:watch": "vitest --watch"
  }
}
```

### 优先测试的组件列表

#### 高优先级 (核心业务组件)
1. **ServicesPage** - 服务管理页面，包含复杂的状态管理和用户交互
2. **Button** - 基础UI组件，使用频率最高
3. **marketStore** - 核心状态管理
4. **useToast** - 重要的用户反馈机制

#### 中优先级 (重要UI组件)
1. **Dialog/AlertDialog** - 模态框组件
2. **Input** - 表单输入组件
3. **Card** - 展示组件
4. **DashboardPage** - 仪表板页面

#### 低优先级 (辅助组件)
1. **其他UI组件** - Select, Textarea, Label等
2. **其他页面** - Login, Profile, Analytics等

### 测试覆盖率目标

#### 初期目标 (第一轮实施)
- **整体覆盖率**: ≥ 60%
- **核心组件覆盖率**: ≥ 80%
- **状态管理覆盖率**: ≥ 70%

#### 最终目标
- **整体覆盖率**: ≥ 80%
- **核心组件覆盖率**: ≥ 90%
- **状态管理覆盖率**: ≥ 85%

## 测试策略

### 单元测试策略
- **组件渲染**: 验证组件正确渲染
- **Props传递**: 测试props的正确传递和处理
- **用户交互**: 测试点击、输入等用户行为
- **条件渲染**: 测试不同状态下的渲染逻辑
- **错误处理**: 测试错误边界和异常情况

### 集成测试策略
- **页面级测试**: 测试完整页面的功能
- **状态管理**: 测试组件与store的交互
- **API交互**: Mock API调用并测试数据流
- **用户流程**: 测试完整的用户操作流程

### Mock策略
- **API调用**: Mock所有外部API请求
- **路由**: Mock react-router-dom
- **状态管理**: 提供测试用的store实例
- **浏览器API**: Mock localStorage, sessionStorage等

## 项目状态板

- 📋 计划制定完成：已分析项目结构并制定详细实施计划
- ⏳ 待开始：配置迁移和测试编写
- 🎯 目标：建立完整的测试体系，覆盖率达到80%

## Completed Tasks

- [x] **Task 1.1: 移除Jest配置** `config`
  - [x] 删除 `jest.config.cjs`
  - [x] 删除 `jest.setup.ts`
  - [x] 更新 `package.json` 脚本

- [x] **Task 1.2: 配置Vitest** `config`
  - [x] 更新 `vite.config.ts` 添加测试配置
  - [x] 创建 `src/__tests__/setup.ts`
  - [x] 配置测试环境和Mock

- [x] **Task 1.3: 创建测试工具函数** `setup`
  - [x] 创建 `src/__tests__/utils/test-utils.tsx`
  - [x] 创建Mock数据生成器
  - [x] 配置自定义渲染函数

- [x] **Task 2.1: UI组件单元测试 (部分)** `unit-test`
  - [x] Button组件测试 (11个测试用例)
  - [x] Input组件测试 (12个测试用例)
  - [x] Card组件测试 (32个测试用例)
  - [x] Dialog组件测试 (18个测试用例)

- [x] **Task 2.2: 自定义Hook测试 (部分)** `unit-test`
  - [x] useToast hook测试 (14个测试用例)

- [x] **Task 2.3: Store单元测试 (部分)** `unit-test`
  - [x] marketStore.ts 测试 (10个测试用例, 覆盖env_vars为null的bug)

- [x] **Task 3.1: 核心页面集成测试 (部分)** `integration-test`
  - [x] ServicesPage集成测试 (10个测试用例)

- [x] **Bug修复: Market页面Details按钮错误** `bugfix`
  - [x] 修复marketStore.ts中fetchServiceDetails方法的null检查问题
  - [x] 问题: `details.env_vars.map` 可能因 `details.env_vars` 为 `null` 而失败。
  - [x] 解决方案: 在调用 `.map` 前添加空值检查 `details.env_vars ? ... : []`。

- [x] **Bug修复: 修复前端编译错误** `bugfix`
  - [x] 解决 `tsc -b && vite build` 过程中出现的多个TypeScript类型错误和未使用的变量/导入问题。
  - [x] 主要修复点包括：
    - `src/__tests__/setup.ts`: 修正 mock API 时的类型问题。
    - `src/components/ui/card.test.tsx`: 移除未使用导入。
    - `src/store/marketStore.ts`: 为 `ServiceType` 补充缺失字段，移除未使用导入和变量。
    - `src/pages/ServicesPage.tsx`: 移除未使用导入。
    - `src/components/market/ServiceDetails.tsx`: 移除未使用导入和变量。
    - `src/components/ui/ConfirmDialog.tsx`: 移除不支持的 `variant` 属性和未使用变量。
    - `src/hooks/useServerAddress.ts` & `src/pages/PreferencesPage.tsx`: 修正 API 调用返回类型问题。
    - `src/pages/ServicesPage.integration.test.tsx`: 为 mock store 添加显式类型。
    - `src/__tests__/utils/test-utils.tsx`: 更新 mock 服务生成函数以符合 `ServiceType`。
    - `src/utils/api.ts`: 调整 axios 响应拦截器以正确处理类型。

## In Progress Tasks

- [ ] 继续Phase 2和Phase 3的剩余测试任务 `implementation`

## Future Tasks

### Phase 2: 单元测试 (继续)
- [ ] **Task 2.1: UI组件单元测试 (继续)** `unit-test`
  - [ ] Select组件测试
  - [ ] Table组件测试
  - [ ] Tabs组件测试
  - [ ] Textarea组件测试
  - [ ] Tooltip组件测试

- [ ] **Task 2.2: 自定义Hook测试 (继续)** `unit-test`
  - [ ] useSidebar hook测试

- [ ] **Task 2.3: Store单元测试 (继续)** `unit-test`
  - [ ] settingsStore.ts 测试

- [ ] **Task 2.4: 工具函数单元测试** `unit-test`
  - [ ] src/utils/index.ts (如果存在)

### Phase 3: 集成测试 (继续)
- [ ] **Task 3.1: 核心页面集成测试 (继续)** `integration-test`
  - [ ] MarketPage集成测试
  - [ ] SettingsPage集成测试
  - [ ] PreferencesPage集成测试

- [ ] **Task 3.2: 核心流程集成测试** `integration-test`
  - [ ] 服务安装与卸载流程 (端到端，涉及store和API交互)
  - [ ] 用户认证流程 (登录、注册、登出)

### Phase 4: E2E 测试
- [ ] **Task 4.1: Playwright配置** `e2e-test`
  - [ ] 安装Playwright
  - [ ] 配置Playwright测试环境

- [ ] **Task 4.2: 核心用户流程E2E测试** `e2e-test`
  - [ ] 用户登录 -> 浏览服务市场 -> 安装服务 -> 配置服务 -> 卸载服务
  - [ ] 用户设置修改

### Phase 5: 测试覆盖率和报告
- [ ] **Task 5.1: 生成测试覆盖率报告** `report`
  - [ ] 配置Vitest生成覆盖率报告
- [ ] **Task 5.2: 分析报告并补充测试** `report`
  - [ ] 目标覆盖率达到80%以上

### Phase 6: CI集成
- [ ] **Task 6.1: 配置GitHub Actions** `ci`
  - [ ] 创建workflow文件，在push和pull_request时运行测试 