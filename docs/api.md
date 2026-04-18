文档概述 
本文档描述了基于多模态大模型技术的Deepfake检测平台的RESTful API接口规范，涵盖用户认证、文件上传、检测任务管理、历史记录和用户管理等功能模块。
基础信息 
- 基础URL: https://api.deepfake-detection.com/v1
- 数据格式: JSON
- 认证方式: Bearer Token (JWT)
统一响应格式 
成功响应 
{
  "code": 200,
  "message": "Success",
  "data": {...},
  "timestamp": 1620000000
}
错误响应 
{
  "code": 400,
  "message": "错误描述",
  "details": "可选错误详情",
  "timestamp": 1620000000
}
认证接口 
1. 用户注册 
- URL: /auth/register
- 方法: POST
- 描述: 创建新用户账户并发送验证邮件
请求体:
{
  "email": "user@example.com",
  "password": "Password123"
}
响应:
{
  "code": 0
}
错误响应
{
"code": 401009,
"message": "用户已存在"
}

1. 邮箱验证 *
- URL: /auth/verify-email
- 方法: GET
- 描述: 验证邮箱地址
参数:
暂时无法在飞书文档外展示此内容
响应:
{
  "code": 200,
  "message": "邮箱验证成功",
  "data": null
}
1. 用户登录 
- URL: /auth/login
- 方法: POST
- 描述: 用户登录获取访问令牌
请求体:
{
  "email": "user@example.com",
  "password": "Password123",
}
响应:
{
  "code": 0,
  "message": "登录成功"
}
密码错误
{
"code": 401005,
"message": "用户密码错误, 请重新输入!"
}
用户不存在
{
"code": 40104,
"message": "用户不存在"
}

1. 令牌刷新 
- URL: /auth/refresh-token
- 方法: POST
- 描述: 使用刷新令牌获取新的访问令牌
响应:
{
  "code": 0,
  "message": "令牌刷新成功"
}
1. 用户登出 
- URL: /auth/logout
- 方法: POST
- 认证: 需要
- 描述: 用户登出系统
响应:
{
  "code": 0,
  "message": "登出成功"
}
文件上传接口 
1. 初始化上传 
- URL: /file/upload/init
- 方法: POST
- 认证: 需要
- 描述: 初始化分块上传任务
请求体:
{
  "fileName": "video.mp4",
  "fileSize": 104857600,
  "mimeType": "video/mp4",
  "md5": ""
}
成功响应:
{
  "code": 0,
  "message": "上传初始化成功",
  "data": {
    "uploadId": "upload_123456",
    "chunkSize": 5242880
  }
}
已存在
{
    "code": 401018,
    "message": "文件已存在"
}

1. 上传完成回调
- URL: /file/upload/callback/
- 方法: POST
- 认证: 需要
- 描述: 完成文件上传
请求体:
{
  "uploadId": "upload_123456"
}
响应:
{
  "code": 0,
  "message": "文件上传完成",
  "data": {
    "url": "对象存储地址"
  }
}
检测任务接口 
1. 创建检测任务 
- URL: /tasks
- 方法: POST
- 认证: 需要
- 描述: 创建新的检测任务
请求体:
{
  "fileId": "file_789012",
  "fileName": "video.mp4",
  "modalities": ["video", "audio", "image", "crossmodal"]
}
响应:
{
  "code": 200,
  "message": "检测任务创建成功",
  "data": {
    "taskId": "task_345678",
    "status": "pending"
  }
}
1. 获取任务状态 
- URL: /tasks/{taskId}
- 方法: GET
- 认证: 需要
- 描述: 获取检测任务状态和进度
响应:
{
  "code": 200,
  "message": "获取任务状态成功",
  "data": {
    "taskId": "task_345678",
    "status": "processing",
    "progress": {
      "video": 65,
      "audio": 40,
      "image": 80,
      "crossmodal": 30
    },
    "estimatedTimeRemaining": 120
  }
}
1.  获取检测结果 
- URL: /tasks/{taskId}/result
- 方法: GET
- 认证: 需要
- 描述: 获取检测任务结果
响应:
{
  "code": 200,
  "message": "获取检测结果成功",
  "data": {
    "taskId": "task_345678",
    "status": "completed",
    "result": {
      "overall": 0.87,
      "modalities": {
        "video": 0.92,
        "audio": 0.78,
        "image": 0.85,
        "crossmodal": 0.81
      },
      "details": {
        "video": {
          "score": 0.92,
          "confidence": 0.89,
          "anomalies": [
            {"time": 12.4, "region": "face", "score": 0.95},
            {"time": 24.7, "region": "mouth", "score": 0.87}
          ]
        },
        "audio": {
          "score": 0.78,
          "confidence": 0.82,
          "anomalies": [
            {"time": 10.2, "type": "spectral", "score": 0.75}
          ]
        }
      },
      "conclusion": "highly_likely_fake",
      "processedAt": 1620003600
    }
  }
}
1.  取消检测任务 
- URL: /tasks/{taskId}/cancel
- 方法: POST
- 认证: 需要
- 描述: 取消进行中的检测任务
响应:
{
  "code": 200,
  "message": "任务已取消",
  "data": null
}
历史记录接口 
1.  获取历史记录列表 
- URL: /history
- 方法: GET
- 认证: 需要
- 描述: 分页获取用户检测历史记录
查询参数:
暂时无法在飞书文档外展示此内容
响应:
{
  "code": 200,
  "message": "获取历史记录成功",
  "data": {
    "items": [
      {
        "taskId": "task_345678",
        "fileName": "video.mp4",
        "fileType": "video",
        "fileSize": 104857600,
        "status": "completed",
        "result": 0.87,
        "createdAt": 1620000000,
        "completedAt": 1620003600
      }
    ],
    "pagination": {
      "page": 1,
      "limit": 10,
      "total": 45,
      "pages": 5
    }
  }
}
1.  删除历史记录 
- URL: /history/{taskId}
- 方法: DELETE
- 认证: 需要
- 描述: 删除单条历史记录
响应:
{
  "code": 200,
  "message": "记录删除成功",
  "data": null
}
1.  批量删除历史记录 
- URL: /history/batch-delete
- 方法: POST
- 认证: 需要
- 描述: 批量删除历史记录
请求体:
{
  "taskIds": ["task_123", "task_456", "task_789"]
}
响应:
{
  "code": 200,
  "message": "批量删除成功",
  "data": {
    "deletedCount": 3
  }
}
用户管理接口 
1.  获取用户信息 
- URL: /user/get-profile
- 方法: GET
- 认证: 需要
- 描述: 获取当前用户个人信息
响应:
{
    "code": 0,
    "message": "操作成功",
    "data":{
        "email": "2114678406@qq.com",
        "nickname": "new_user",
        "avatarUrl": "http://8.138.83.109:10001/avatar/default_avatar.png",
        "phone": ""
        }
}
1.  更新用户信息 
- URL: /user/update-profile
- 方法: PUT
- 认证: 需要
- 描述: 更新用户基本信息，不需要更新的字段直接不写
请求体:
{
  "nickname": "新昵称",
  "phone": "电话号码"
}
响应:
{
  "code": 0,
  "message": "用户信息更新成功",
  "data": null
}


头像上传流程
17. 用户头像上传初始化 
- URL: /user/avatar-upload/init
- 方法: POST
- 认证: 需要
- 描述: 用户头像上传初始化 
请求体: 
{
  "mime_type": "image/jpg或image/png或image/jpeg",
  "file_size": "文件大小，单位字节"
}
响应:

{
    "code": 0,
    "message": "操作成功",
    "data":{
        "access_key_id": "STS.NXarhQXK7KQAABHkNK2oue11c",
        "access_key_secret": "FqNNhsuiwusAk38fiJa7zfpCRzjoDgP2YQbvH6JzzK5Y",
        "security_token": "CAISnAN1q6Ft5B2yfSjIr5vUOdLltZQW/JOqQ2T5j04ePuBZivSagTz2IHhMeXVhCOwetf8xmm9W6vkalrR6QpZMTEXzNcUvsMxdrF7kOtWf4pbsteRY2cf7QjBnwPgXFZSADd/iRfbxJ91IAjKtz12cqNmXXDGmWEPffv/toJV7b9MRcxClZD5dfrl/LRdjr8loUhm0Mu22YCb3nk3aDkdjpnBB6wVF5L+439eX5zfHkVT+0ZV1nYnqJYW+ZMQebfUgWtyujuttbfiDgmwC9gNH7uJ3iaVL9CyCtMyWAFRNpByYOvbd6JpmJQZwb6ENQ6Ab8qD2yKA/+M6rztiolUkQZbEICXuCFN76nPGpQr35aowLEp/gIGnI39y1MZ34jhgpe3pzNnkRI4J8di4hVUR3EW+EdfH7pwjQGg69QrSCyqg92pdu1E/v+dear4hn+1ASdkzyU75LjCNAX3Z+tQSJ+V8ddzf3WVgu8TrB+BnJwMTCa0d8h0yzMGMphzo8xIWkPCuK5fhHNdmhga5O6YCUbuxMq2NEJ52lx+nx4i98GoABVEzVc+qltR9DZE560chjOQf/TWM2W2fVwAZQFANkF4279Cd1j9fK7d4QSdj1iqNSSRmJN1+QKni2slFqDw1krtoWL5tn1057twh7pdhRO3trcqiS8gcWMSnmdXAW8tamodH5dQbLAcFb7wvJxi4RE+LjE/3DEzOjAK72J0dhiKEgAA==",
        "expiration": "2025-10-23 10:00:49",
        "bucket_name": "user-images-bucket",
        "region": "cn-guangzhou",
        "endpoint": "sts.cn-guangzhou.aliyuncs.com",
        "object_path": "/avatar/avatar-aaaaaa_0f760b557422adac.jpg",
        "max_file_size": 10485760,
        "allowed_file_types":[
        "jpg",
        "jpeg",
        "png"
        ]
    }
}
18. 用户头像上传完成回调
- URL: /user/avatar-upload/callback
- 方法: POST
- 认证: 需要
- 描述: 向oss服务上传图片时附带的回调接口，上传完成后oss服务会自动调用该接口通知应用服务器
请求体: 
{
    "access_key_id": "STS.NXarhQXK7KQAABHkNK2oue11c",
    "bucket_name": "user-images-bucket"
}
响应:
{
    "code": 0,
    "message": "操作成功",
    "data":{
        "avatar_url": "http://8.138.83.109:10001/avatar/avatar-aaaaaa_0f760b557422adac.jpg",
        "status": "success"
    }
}
19. 获取用户头像url
- URL: /user/fecth-avatar/
- 方法: GET
- 认证: 需要
- 描述: 向oss服务上传图片时附带的回调接口，上传完成后oss服务会自动调用该接口通知应用服务器
响应:
{
    "code": 0,
    "message": "操作成功",
    "data":{
    "avatar_url": "http://8.138.83.109:10001/avatar/avatar-aaaaaa_0f760b557422adac.jpg",
    "status": "success"
    }
}

20. 修改密码 
- URL: /user/password
- 方法: PUT
- 认证: 需要
- 描述: 修改用户密码
请求体:
{
  "oldPassword": "OldPassword123",
  "newPassword": "NewPassword456",
  "confirmPassword": "NewPassword456"
}
响应:
{
  "code": 200,
  "message": "密码修改成功，请重新登录",
  "data": null
}
21. 请求修改邮箱 
- URL: /user/email/request-change
- 方法: POST
- 认证: 需要
- 描述: 请求修改邮箱地址
请求体:
{
  "newEmail": "newuser@example.com"
}
响应:
{
  "code": 200,
  "message": "验证码已发送到新邮箱",
  "data": null
}
22. 确认修改邮箱 
- URL: /user/email/confirm-change
- 方法: POST
- 认证: 需要
- 描述: 确认并完成邮箱修改
请求体:
{
  "newEmail": "newuser@example.com",
  "verificationCode": "123456"
}
响应:
{
  "code": 200,
  "message": "邮箱修改成功",
  "data": null
}
报告生成接口 *
23. 生成检测报告 
- URL: /reports/{taskId}
- 方法: POST
- 认证: 需要
- 描述: 生成检测报告(PDF格式)
请求体:
{
  "format": "pdf",
  "includeDetails": true,
  "language": "zh-CN"
}
响应:
{
  "code": 200,
  "message": "报告生成成功",
  "data": {
    "reportId": "report_901234",
    "downloadUrl": "/reports/report_901234.pdf",
    "expiresAt": 1620086400
  }
}
错误代码说明 
暂时无法在飞书文档外展示此内容