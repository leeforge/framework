# HTTP Responder - 前端使用文档

## 📋 响应数据结构

所有 API 响应都遵循统一的结构：

```typescript
interface Response<T> {
  data: T | null;        // 响应数据
  error: Error | null;   // 错误信息
  meta: Meta;           // 元数据
}

interface Error {
  code: number;          // 错误码
  message: string;       // 错误消息
  details?: any;         // 详细信息（可选）
}

interface Meta {
  traceId?: string;      // 请求追踪 ID
  took?: number;         // 处理耗时（毫秒）
  pagination?: Pagination; // 分页信息
}

interface Pagination {
  page: number;          // 当前页码
  pageSize: number;      // 每页条数
  total: number;         // 总条数
  totalPages: number;    // 总页数
  hasMore: boolean;      // 是否还有更多
}
```

## 📝 成功响应示例

### 1. 单条数据查询

**请求：** `GET /api/posts/1`

**响应：**
```json
{
  "data": {
    "id": 1,
    "title": "Hello World",
    "content": "这是文章内容",
    "createdAt": "2024-01-12T10:00:00Z"
  },
  "error": null,
  "meta": {
    "traceId": "abc-123-def",
    "took": 15
  }
}
```

**前端处理：**
```typescript
const response = await fetch('/api/posts/1');
const result = await response.json();

if (result.error) {
  // 处理错误
  console.error(result.error.message);
  return;
}

// 使用数据
const post = result.data;
console.log(post.title);
```

### 2. 列表查询（带分页）

**请求：** `GET /api/posts?page=1&pageSize=10`

**响应：**
```json
{
  "data": [
    { "id": 1, "title": "文章1" },
    { "id": 2, "title": "文章2" }
  ],
  "error": null,
  "meta": {
    "traceId": "xyz-456-uvw",
    "took": 23,
    "pagination": {
      "page": 1,
      "pageSize": 10,
      "total": 100,
      "totalPages": 10,
      "hasMore": true
    }
  }
}
```

**前端处理：**
```typescript
const response = await fetch('/api/posts?page=1&pageSize=10');
const result = await response.json();

if (result.error) {
  console.error(result.error.message);
  return;
}

const posts = result.data; // 数组
const pagination = result.meta.pagination;

// 渲染列表
posts.forEach(post => {
  // ...
});

// 分页控件
console.log(`第 ${pagination.page}/${pagination.totalPages} 页`);
console.log(`总共 ${pagination.total} 条`);
```

### 3. 创建/更新操作

**请求：** `POST /api/posts`

**响应：**
```json
{
  "data": {
    "id": 101,
    "title": "新文章",
    "createdAt": "2024-01-12T10:05:00Z"
  },
  "error": null,
  "meta": {
    "traceId": "req-789",
    "took": 45
  }
}
```

## ❌ 错误响应示例

### 1. 参数验证错误

**响应：** `400 Bad Request`
```json
{
  "data": null,
  "error": {
    "code": 4001,
    "message": "invalid request body",
    "details": {
      "title": "required field",
      "content": "min length 10 required"
    }
  },
  "meta": {
    "traceId": "err-001",
    "took": 5
  }
}
```

### 2. 资源不存在

**响应：** `404 Not Found`
```json
{
  "data": null,
  "error": {
    "code": 4041,
    "message": "post not found"
  },
  "meta": {
    "traceId": "err-002",
    "took": 8
  }
}
```

### 3. 服务器错误

**响应：** `500 Internal Server Error`
```json
{
  "data": null,
  "error": {
    "code": 5000,
    "message": "internal server error",
    "details": "database connection failed"
  },
  "meta": {
    "traceId": "err-003",
    "took": 2
  }
}
```

## 🎨 前端封装示例

### 通用请求封装

```typescript
interface ApiResponse<T> {
  data: T | null;
  error: Error | null;
  meta: Meta;
}

interface Error {
  code: number;
  message: string;
  details?: any;
}

interface Meta {
  traceId?: string;
  took?: number;
  pagination?: Pagination;
}

class ApiClient {
  private baseUrl: string;

  constructor(baseUrl: string = '/api') {
    this.baseUrl = baseUrl;
  }

  async get<T>(path: string, params?: Record<string, any>): Promise<T> {
    const url = this.buildUrl(path, params);
    const response = await fetch(url);
    return this.handleResponse<T>(response);
  }

  async post<T>(path: string, body?: any): Promise<T> {
    const url = `${this.baseUrl}${path}`;
    const response = await fetch(url, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body)
    });
    return this.handleResponse<T>(response);
  }

  async put<T>(path: string, body?: any): Promise<T> {
    const url = `${this.baseUrl}${path}`;
    const response = await fetch(url, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body)
    });
    return this.handleResponse<T>(response);
  }

  async delete<T>(path: string): Promise<T> {
    const url = `${this.baseUrl}${path}`;
    const response = await fetch(url, { method: 'DELETE' });
    return this.handleResponse<T>(response);
  }

  private async handleResponse<T>(response: Response): Promise<T> {
    const result: ApiResponse<T> = await response.json();

    if (result.error) {
      // 可以在这里统一处理错误提示
      throw new ApiError(result.error.message, result.error.code, result.error.details);
    }

    return result.data as T;
  }

  private buildUrl(path: string, params?: Record<string, any>): string {
    if (!params) return `${this.baseUrl}${path}`;

    const searchParams = new URLSearchParams(params);
    return `${this.baseUrl}${path}?${searchParams.toString()}`;
  }
}

class ApiError extends Error {
  constructor(
    message: string,
    public code: number,
    public details?: any
  ) {
    super(message);
    this.name = 'ApiError';
  }
}

// 使用示例
const api = new ApiClient('http://localhost:8080/api');

// 查询单条
const post = await api.get<Post>('/posts/1');

// 查询列表
const posts = await api.get<Post[]>('/posts', { page: 1, pageSize: 10 });

// 创建
const newPost = await api.post<Post>('/posts', { title: 'New', content: '...' });

// 更新
const updated = await api.put<Post>('/posts/1', { title: 'Updated' });

// 删除
await api.delete<void>('/posts/1');
```

### React Hook 示例

```typescript
import { useState, useEffect } from 'react';

interface UseApiResponse<T> {
  data: T | null;
  loading: boolean;
  error: Error | null;
  refetch: () => Promise<void>;
}

function useApi<T>(path: string): UseApiResponse<T> {
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  const fetch = async () => {
    setLoading(true);
    setError(null);

    try {
      const response = await fetch(path);
      const result = await response.json();

      if (result.error) {
        setError(result.error);
        setData(null);
      } else {
        setData(result.data);
      }
    } catch (err) {
      setError({ code: 5000, message: 'Network error' });
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetch();
  }, [path]);

  return { data, loading, error, refetch: fetch };
}

// 使用
function PostDetail({ id }: { id: number }) {
  const { data: post, loading, error } = useApi<Post>(`/api/posts/${id}`);

  if (loading) return <div>Loading...</div>;
  if (error) return <div>Error: {error.message}</div>;
  if (!post) return <div>Not found</div>;

  return (
    <div>
      <h1>{post.title}</h1>
      <p>{post.content}</p>
    </div>
  );
}
```

### 分页组件示例

```typescript
interface PaginationProps {
  pagination: Pagination;
  onPageChange: (page: number) => void;
}

function Pagination({ pagination, onPageChange }: PaginationProps) {
  const { page, totalPages, hasMore } = pagination;

  return (
    <div className="pagination">
      <button
        disabled={page === 1}
        onClick={() => onPageChange(page - 1)}
      >
        上一页
      </button>

      <span>第 {page} / {totalPages} 页</span>

      <button
        disabled={!hasMore}
        onClick={() => onPageChange(page + 1)}
      >
        下一页
      </button>
    </div>
  );
}

// 使用
function PostList() {
  const [page, setPage] = useState(1);
  const [posts, setPosts] = useState<Post[]>([]);
  const [pagination, setPagination] = useState<Pagination | null>(null);

  useEffect(() => {
    fetch(`/api/posts?page=${page}&pageSize=10`)
      .then(res => res.json())
      .then(result => {
        if (!result.error) {
          setPosts(result.data);
          setPagination(result.meta.pagination);
        }
      });
  }, [page]);

  return (
    <div>
      <PostGrid posts={posts} />
      {pagination && (
        <Pagination
          pagination={pagination}
          onPageChange={setPage}
        />
      )}
    </div>
  );
}
```

## 🔍 调试技巧

### 1. 使用 TraceId 追踪请求

```typescript
// 在开发环境中显示 traceId
fetch('/api/posts/1')
  .then(res => res.json())
  .then(result => {
    console.log('Trace ID:', result.meta.traceId);
    console.log('Processing time:', result.meta.took, 'ms');
  });
```

### 2. 统一错误处理

```typescript
function handleApiError(error: ApiError) {
  switch (error.code) {
    case 4001:
      // 参数验证错误
      console.error('参数错误:', error.details);
      break;
    case 4041:
      // 资源不存在
      console.error('资源不存在');
      break;
    case 5000:
      // 服务器错误
      console.error('服务器错误:', error.message);
      break;
    default:
      console.error('未知错误:', error);
  }
}
```

## 📌 注意事项

1. **所有响应都有 `data`、`error`、`meta` 三个字段**
   - 成功时：`data` 有值，`error` 为 `null`
   - 失败时：`error` 有值，`data` 为 `null`

2. **分页查询必须检查 `meta.pagination`**
   - 列表接口返回分页信息
   - 单条查询没有分页信息

3. **错误码规范**
   - `4xxx`: 客户端错误（4001: 验证失败, 4041: 未找到）
   - `5xxx`: 服务器错误（5000: 内部错误）

4. **可选字段**
   - `error.details`: 错误详细信息
   - `meta.traceId`: 请求追踪 ID
   - `meta.took`: 处理耗时
