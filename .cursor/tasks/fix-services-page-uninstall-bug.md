# 修复服务页面连续卸载服务时的显示bug

## Background and Motivation

在ServicesPage中，当用户连续卸载两个服务时，第一个已经消失的服务会重新显示出来。这是由于前端状态管理的竞态条件和数据同步问题导致的。

## Key Challenges and Analysis

### 问题根本原因分析

1. **双重状态管理**：
   - `globalInstalledServices` (来自useMarketStore)
   - `localInstalledServices` (组件本地状态)
   - 两者之间存在同步问题

2. **竞态条件**：
   ```typescript
   // 第一次卸载
   await uninstallService(serviceId1);  // 更新全局状态
   setLocalInstalledServices(prev => prev.filter(s => s.id !== serviceId1)); // 手动更新本地状态
   
   // 第二次卸载可能触发fetchInstalledServices()
   await uninstallService(serviceId2);  // 这可能导致globalInstalledServices更新
   
   // useEffect会重新同步，可能包含已卸载的服务
   useEffect(() => {
       setLocalInstalledServices(globalInstalledServices);
   }, [globalInstalledServices]);
   ```

3. **时序问题**：
   - `uninstallService`在marketStore中已经正确更新了`installedServices`状态
   - 但ServicesPage的`useEffect`可能在不合适的时机重新同步数据
   - 手动的本地状态更新和全局状态更新之间存在不一致

### 具体流程分析

1. **第一次卸载**：
   - 调用`uninstallService(serviceId1)`
   - marketStore正确更新`installedServices`，移除serviceId1
   - 手动调用`setLocalInstalledServices`移除serviceId1
   - UI正确显示serviceId1已被移除

2. **第二次卸载**：
   - 调用`uninstallService(serviceId2)`
   - 如果此时`globalInstalledServices`发生变化（可能由于某种原因包含了serviceId1）
   - `useEffect`触发，`setLocalInstalledServices(globalInstalledServices)`
   - serviceId1重新出现在UI中

## High-level Task Breakdown

- 移除冗余的本地状态管理
- 统一使用全局状态
- 确保卸载操作的原子性
- 优化状态同步逻辑

## Project Status Board

- ✅ 问题已识别：双重状态管理导致的竞态条件
- ✅ 计划制定完成：已制定详细修复方案
- ✅ 核心修复已完成：移除了冗余的本地状态管理
- ⏳ 待测试：需要验证修复效果

## Completed Tasks

- [x] **Task 1: 移除冗余的本地状态管理** `refactor`
  - [x] 1.1 移除`localInstalledServices`状态
  - [x] 1.2 直接使用`globalInstalledServices`
  - [x] 1.3 移除相关的`useEffect`同步逻辑
  - [x] 1.4 更新所有引用`localInstalledServices`的地方

- [x] **Task 2: 优化卸载操作逻辑** `bug-fix`
  - [x] 2.1 移除`handleUninstallConfirm`中的手动状态更新
  - [x] 2.2 确保`uninstallService`方法正确更新全局状态（已验证marketStore实现正确）
  - [x] 2.3 验证marketStore中的状态更新逻辑（已确认正确）

## In Progress Tasks

无

## Future Tasks

- [ ] **Task 3: 测试和验证修复效果** `test`
  - [ ] 3.1 测试连续卸载多个服务
  - [ ] 3.2 验证UI状态与实际数据一致性
  - [ ] 3.3 测试各种边界情况

## Implementation Plan

### 修复方案

#### 方案1：移除本地状态（推荐）
```typescript
// 移除这些状态和逻辑
const [localInstalledServices, setLocalInstalledServices] = useState<ServiceType[]>([]);

useEffect(() => {
    setLocalInstalledServices(globalInstalledServices);
}, [globalInstalledServices]);

// 直接使用
const allServices = globalInstalledServices;
const activeServices = globalInstalledServices.filter(s => s.health_status === 'active' || s.health_status === 'Active');
const inactiveServices = globalInstalledServices.filter(s => s.health_status === 'inactive' || s.health_status === 'Inactive');

// 简化卸载逻辑
const handleUninstallConfirm = async () => {
    if (!pendingUninstallId) return;
    const serviceToUninstallId = pendingUninstallId;
    setUninstallDialogOpen(false);
    setPendingUninstallId(null);

    try {
        await uninstallService(serviceToUninstallId);
        toast({
            title: 'Uninstall Complete',
            description: 'Service has been successfully uninstalled.'
        });
        // 移除手动状态更新，依赖marketStore的状态管理
    } catch (e: any) {
        toast({
            title: 'Uninstall Failed',
            description: e?.message || 'Unknown error',
            variant: 'destructive'
        });
    }
};
```

#### 方案2：修复同步逻辑（备选）
如果需要保留本地状态，则需要：
1. 在`handleUninstallConfirm`中移除手动状态更新
2. 确保`useEffect`只在必要时触发
3. 添加状态版本控制或时间戳来避免过期数据覆盖

### 相关文件

- `frontend/src/pages/ServicesPage.tsx` - 主要修改文件
- `frontend/src/store/marketStore.ts` - 验证状态管理逻辑

## Lessons

- 避免双重状态管理，特别是在有异步操作的情况下
- 手动状态更新和自动同步之间容易产生竞态条件
- 应该有单一的数据源（Single Source of Truth）
- 状态管理应该集中化，避免分散在多个组件中
- 修复实施经验：
  - 移除冗余的本地状态是最直接有效的解决方案
  - marketStore中的uninstallService方法已经正确处理了状态更新
  - 简化状态管理逻辑可以避免很多潜在的竞态条件问题
  - 直接使用全局状态比本地状态同步更可靠

## ACT mode Feedback or Assistance Requests

无

## User Specified Lessons

无 