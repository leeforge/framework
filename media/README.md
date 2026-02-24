# media — 媒体处理

提供媒体文件的上传存储、图片处理（缩放/裁剪/格式转换）与异步处理队列能力，支持本地存储和 OSS（阿里云）。

## 子包

| 包 | 路径 | 功能 |
|---|---|---|
| `storage` | `media/storage` | 存储提供者（本地 / OSS） |
| `processor` | `media/processor` | 图片处理管道（缩放、裁剪、水印）|
| `queue` | `media/queue` | 异步媒体处理任务队列 |

---

## storage — 文件存储

```go
import "github.com/leeforge/framework/media/storage"

// 本地存储
local := storage.NewLocalStorage(storage.LocalConfig{
    BasePath: "/var/app/uploads",
    BaseURL:  "https://cdn.example.com/uploads",
})

// 上传文件
output, err := local.Upload(ctx, storage.UploadInput{
    FileName:    "avatar.jpg",
    ContentType: "image/jpeg",
    Body:        fileReader,
    Size:        fileSize,
})
// output.URL: 可访问的公开 URL
// output.Key: 存储路径/Key

// 删除文件
err = local.Delete(ctx, output.Key)
```

### OSS 存储（阿里云）

```go
oss := storage.NewOSSStorage(storage.OSSConfig{
    Endpoint:        "oss-cn-hangzhou.aliyuncs.com",
    AccessKeyID:     os.Getenv("OSS_ACCESS_KEY"),
    AccessKeySecret: os.Getenv("OSS_ACCESS_SECRET"),
    BucketName:      "my-bucket",
    BaseURL:         "https://my-bucket.oss-cn-hangzhou.aliyuncs.com",
})
```

### StorageProvider 接口

```go
type StorageProvider interface {
    Upload(ctx context.Context, input UploadInput) (UploadOutput, error)
    Delete(ctx context.Context, key string) error
    GetURL(key string) string
}
```

---

## processor — 图片处理

```go
import "github.com/leeforge/framework/media/processor"

proc := processor.NewImageProcessor(processor.ImageConfig{
    MaxWidth:  4096,
    MaxHeight: 4096,
    Quality:   85,
})

// 缩放到指定尺寸
output, err := proc.Process(ctx, imageBytes)

// 预设尺寸
thumb := processor.Thumbnail // 245×156, Q80
small := processor.Small     // 500×500, Q85

// 处理管道（支持链式操作）
result, err := proc.
    Resize(processor.Thumbnail).
    Watermark("logo.png", processor.BottomRight).
    Convert("webp").
    ProcessFromFile(ctx, "/path/to/image.jpg")
```

---

## queue — 异步处理队列

```go
import "github.com/leeforge/framework/media/queue"

q := queue.NewProcessor(queue.Config{
    Workers:    4,
    BufferSize: 100,
})
q.Start()
defer q.Stop()

// 提交处理任务
q.Submit(queue.Task{
    MediaID:  "media-123",
    FilePath: "/uploads/photo.jpg",
    Actions: []queue.Action{
        {Type: "resize", Options: processor.Thumbnail},
        {Type: "resize", Options: processor.Small},
    },
    OnComplete: func(results []queue.Result) {
        // 更新数据库中的媒体格式记录
    },
})
```

---

## 典型上传流程

```
1. 接收上传请求（multipart/form-data）
2. 校验文件类型/大小
3. StorageProvider.Upload → 保存原始文件
4. 创建 Media 实体记录
5. queue.Submit → 异步生成缩略图/各种尺寸格式
6. 队列回调 → 更新 MediaFormat 记录
```

## 注意事项

- OSS AccessKey 必须通过环境变量注入，**禁止**提交到代码仓库
- 图片处理依赖 `nfnt/resize`，大图处理（>10MB）建议在独立 goroutine 中进行
- 队列处理失败后应记录到 `MediaFormat.Status = "failed"`，方便重试
