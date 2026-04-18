服务信息
暂时无法在飞书文档外展示此内容

---
GET /health — 健康检查
用于服务探活。
请求示例:
curl https://u791094-633r-35cee4af.bjb1.seetacloud.com:8443/health
响应 200 OK:
{
  "status": "ready",
  "gpu": "RTX 6000 Ada Generation",
  "gpu_memory_used_gb": 15.23
}
暂时无法在飞书文档外展示此内容

---
POST /api/detect — 图片检测
上传一张图片，返回分类结果、描述文本和结果图片下载地址。
请求:
- Content-Type: multipart/form-data
暂时无法在飞书文档外展示此内容
cURL 示例:
curl -X POST \
  https://u791094-633r-35cee4af.bjb1.seetacloud.com:8443/api/detect \
  -F "file=@/path/to/image.jpg"
成功响应 200 OK:
{
  "category": "tampered",
  "description": "The tampering involves the alteration of a single object within the image, specifically the snowboard, which is depicted in a different position and angle than the rest of the scene.",
  "raw_output": "... ASSISTANT: [CLS] [SEG] Type: The image is tampered [SEG] Type: The tampering involves... [END]",
  "mask_urls": [
    "/results/figure3_a1b2c3d4_mask_0.jpg"
  ],
  "masked_image_urls": [
    "/results/figure3_a1b2c3d4_masked_img_0.jpg"
  ]
}
响应字段:
暂时无法在飞书文档外展示此内容
mask_urls 和 masked_image_urls 通常各只有一个元素（索引 0）。文件在服务端 10 分钟后自动删除。
错误响应:
暂时无法在飞书文档外展示此内容

---
GET /results/{filename} — 下载结果图片
下载推理生成的 mask 或叠加图片。
请求示例:
# mask_urls 中返回的是相对路径，拼接 Base URL 即可
curl -O https://u791094-633r-35cee4af.bjb1.seetacloud.com:8443/results/figure3_a1b2c3d4_mask_0.jpg
响应: 图片二进制流（image/jpeg）
错误响应:
暂时无法在飞书文档外展示此内容

---
Spring Boot 集成示例
// ========== 1. 调用检测接口 ==========

// 构造 multipart 请求
MultipartBody body = new MultipartBody.Builder()
    .setType(MultipartBody.FORM)
    .addFormDataPart("file", imageFile.getName(),
        RequestBody.create(imageFile, MediaType.parse("image/jpeg")))
    .build();

Request request = new Request.Builder()
    .url("https://u791094-633r-35cee4af.bjb1.seetacloud.com:8443/api/detect")
    .post(body)
    .build();

Response response = okHttpClient.newCall(request).execute();
if (!response.isSuccessful()) {
    // 处理错误：response.code() 为 400/413/503/500
    String error = new JSONObject(response.body().string()).getString("detail");
    throw new BusinessException(error);
}

JSONObject result = new JSONObject(response.body().string());

// 解析结果
String category = result.getString("category");        // "tampered"
String description = result.getString("description");  // "The tampering involves..."

// ========== 2. 下载结果图片（如有） ==========

String baseUrl = "https://u791094-633r-35cee4af.bjb1.seetacloud.com:8443";
JSONArray maskUrls = result.getJSONArray("mask_urls");

if (maskUrls.length() > 0) {
    // 取第一张 mask（通常只有一张）
    String maskRelativeUrl = maskUrls.getString(0);
    Request maskReq = new Request.Builder()
        .url(baseUrl + maskRelativeUrl)
        .get()
        .build();
    Response maskResp = okHttpClient.newCall(maskReq).execute();
    byte[] maskBytes = maskResp.body().bytes();
    // 保存或上传到 OSS...
}
