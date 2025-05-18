# Feature: Service Card Display Refinement

This feature focuses on updating the `ServiceCard.tsx` component and related data mappings in `marketStore.ts` to better present service information to the user, based on new display requirements.

## Completed Tasks

- [x] **Task 1: Update `ServiceType` interface and `marketStore.ts` mapping** `ref-struct`
    - **Description**: Modify the frontend data structures and data transformation logic to accommodate new display requirements for service cards, such as including a homepage URL and refining how author and star information is sourced and stored.
    - **Sub-tasks**:
        - [x] **1.1**: Add `homepageUrl?: string;` and `npmScore?: number;` (for npm-specific score) to the `ServiceType` interface in `frontend/src/store/marketStore.ts`.
        - [x] **1.2**: Update the mapping logic in the `searchServices` action of `marketStore.ts`:
            - [x] Map backend `item.homepage` to the new `homepageUrl` field.
            - [x] Map backend `item.github_stars` (if provided by backend enhancement) to `stars`. If `item.github_stars` is not available, `stars` can be undefined or null.
            - [x] Map backend `item.score` (npm score) to `npmScore` if `item.package_manager` is 'npm'.
            - [x] Refine `author` mapping: if `item.author` (from package manager) is absent or generic, and `homepageUrl` points to GitHub, attempt to parse owner/org from `homepageUrl` to use as `author`. Otherwise, use package manager's author or a placeholder.
    - **Success Criteria**: The `searchResults` in the `marketStore` now contain objects enriched with `homepageUrl`, `stars` (ideally GitHub stars), `npmScore`, and the `author` field is populated based on the refined logic.

- [x] **Task 2: Modify `ServiceCard.tsx` Component UI** `ref-func`
    - **Description**: Update the visual presentation of individual service cards in the marketplace to reflect the new information requirements: display GitHub homepage, remove download counts, and adjust how stars and author details are shown.
    - **Sub-tasks**:
        - [x] **2.1**: Remove the UI elements that display the download count from `ServiceCard.tsx`.
        - [x] **2.2**: Update the UI to display star count:
            - [x] If `service.stars` (GitHub stars) is available, display it with a star icon.
            - [x] Else, if `service.npmScore` is available (for npm packages without GitHub stars), display this score, possibly labelled as "npm Score" or with a different visual cue.
            - [x] If neither is available, display N/A or hide the star/score section.
        - [x] **2.3**: Revise the author display section. If a valid `service.author` exists, show it. If not (e.g., placeholder or derived from GitHub URL), ensure it's clear. Consider linking to `homepageUrl` if appropriate.
        - [x] **2.4**: Add a new UI element (e.g., a GitHub icon with a link) to the service card that directs users to the `service.homepageUrl`.
    - **Success Criteria**: The `ServiceCard` component correctly and clearly displays service information as per the new requirements, including a link to the GitHub homepage, no download count, and an improved representation of stars and author/source.

- [x] **Task 3: 聚合并返回 npm 包的 GitHub stars** `new-feat`
    - **Description**: 后端自动聚合 npm 包的 GitHub stars 字段，优先用 Redis 缓存，极大提升性能和抗限流能力，接口 contract 满足前端需求。
    - **Sub-tasks**:
        - [x] 解析 npm 包 repository 字段，提取 GitHub owner/repo
        - [x] 调用 GitHub API 获取 stars，支持 token
        - [x] stars 字段与前端 contract 对齐（github_stars）
        - [x] 兜底处理 API 失败、无 stars 情况
        - [x] Redis 缓存生效，key=github_stars:owner:repo，ttl=10min
    - **Success Criteria**: 前端能稳定拿到真实 star 数，重复请求命中缓存，后端无限流风险。

## In Progress Tasks

- [x] （全部完成）

## Future Tasks

- （全部完成，无后续任务）

## Implementation Plan

**Phase 1: Data Preparation (marketStore.ts)**
1.  Extend `ServiceType` interface with `homepageUrl` and `npmScore`.
2.  In `searchServices` mapping:
    *   Populate `homepageUrl` from `item.homepage`.
    *   Populate `stars` from `item.github_stars` (if backend provides it and it's a valid number).
    *   Populate `npmScore` from `item.score` if `item.package_manager` is 'npm' (and `item.github_stars` was not available or not applicable).
    *   **Decision Point (Author)**: Based on user feedback, if `item.author` is empty or generic:
        *   Option A: Attempt to parse owner from `item.homepageUrl` if it's a GitHub URL.
        *   Option B: Keep as "Unknown Author" or a similar placeholder if parsing fails or not a GitHub URL.
        *   Option C: Replace traditional author display with a direct link to the source repository using `homepageUrl`.

**Phase 2: UI Update (ServiceCard.tsx)**
1.  Remove the HTML/JSX block responsible for rendering downloads.
2.  Modify the stars display block:
    *   If `service.stars` (GitHub stars) is present, display with Star icon.
    *   Else if `service.npmScore` is present, display as "npm Score: [value]" or similar distinction.
    *   Else, show N/A or omit.
3.  Modify the author display block as per author mapping decision.
4.  Add a new `<a>` tag, possibly styled as an icon button (e.g., using a GitHub icon from `lucide-react`), that links to `service.homepageUrl`. This could be near the service name or in the footer of the card.

### Relevant Files

- `frontend/src/store/marketStore.ts` - For `ServiceType` interface and data mapping.
- `frontend/src/components/ServiceCard.tsx` - For UI changes to the service card display.
- `frontend/src/pages/MarketPage.tsx` - Parent page, to ensure changes integrate correctly. 