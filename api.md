# 智能招聘系统 - 接口文档

## 统一返回格式

```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

错误时 code 非 0，message 为错误信息。

---

## 一、认证接口（无需 JWT）

### 1. HR 注册

- **POST** `/api/hr/register`
- **请求体**：
```json
{
  "username": "hr01",
  "password": "123456",
  "name": "张HR"
}
```
- **返回**：
```json
{ "code": 0, "message": "success", "data": { "id": 1 } }
```

### 2. HR 登录

- **POST** `/api/hr/login`
- **请求体**：
```json
{
  "username": "hr01",
  "password": "123456"
}
```
- **返回**：
```json
{
  "code": 0, "message": "success",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "id": 1,
    "username": "hr01",
    "name": "张HR"
  }
}
```

### 3. 候选人注册

- **POST** `/api/candidate/register`
- **请求体**：
```json
{
  "username": "candidate01",
  "password": "123456"
}
```
- **返回**：
```json
{ "code": 0, "message": "success", "data": { "id": 1 } }
```

### 4. 候选人登录

- **POST** `/api/candidate/login`
- **请求体**：
```json
{
  "username": "candidate01",
  "password": "123456"
}
```
- **返回**：
```json
{
  "code": 0, "message": "success",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "id": 1,
    "username": "candidate01"
  }
}
```

### 5. 公开岗位列表（游客可访问）

- **GET** `/api/jobs?page=1&pageSize=10`
- **权限**：无需登录
- **返回**：
```json
{
  "code": 0, "message": "success",
  "data": {
    "jobs": [
      {
        "id": 1, "hr_id": 1, "title": "Go 开发工程师",
        "description": "...", "requirement": "...",
        "salary": "15k-25k", "location": "北京",
        "status": 1, "created_at": "2024-01-01 10:00:00"
      }
    ],
    "total": 1
  }
}
```

---

## 二、HR 接口（需 JWT，role=hr）

所有请求需携带 Header：`Authorization: Bearer <token>`

### 6. 新增岗位

- **POST** `/api/hr/jobs`
- **请求体**：
```json
{
  "title": "Go 开发工程师",
  "description": "负责后端开发",
  "requirement": "3年以上经验",
  "salary": "15k-25k",
  "location": "北京"
}
```
- **返回**：
```json
{ "code": 0, "message": "success", "data": { "id": 1 } }
```

### 7. 编辑岗位

- **PUT** `/api/hr/jobs/:id`
- **权限**：仅能编辑自己创建的岗位
- **请求体**：同新增
- **返回**：
```json
{ "code": 0, "message": "success", "data": null }
```

### 8. 下架岗位

- **PUT** `/api/hr/jobs/:id/offline`
- **权限**：仅能下架自己创建的岗位
- **返回**：
```json
{ "code": 0, "message": "success", "data": null }
```

### 9. HR 岗位列表（仅自己的）

- **GET** `/api/hr/jobs?page=1&pageSize=10`
- **返回**：
```json
{
  "code": 0, "message": "success",
  "data": { "jobs": [...], "total": 5 }
}
```

### 10. 查看岗位投递列表

- **GET** `/api/hr/jobs/:id/applications?page=1&pageSize=10`
- **权限**：仅能查看自己岗位的投递
- **返回**：
```json
{
  "code": 0, "message": "success",
  "data": {
    "applications": [
      {
        "id": 1, "jobId": 1, "candidateId": 1,
        "status": 0, "created_at": "...",
        "job_title": "Go 开发", "candidate_name": "张三"
      }
    ],
    "total": 1
  }
}
```

### 11. 查看投递详情（候选人档案+简历）

- **GET** `/api/hr/applications/:id/detail`
- **权限**：仅能查看自己岗位下的投递详情
- **返回**：
```json
{
  "code": 0, "message": "success",
  "data": {
    "application": { "id": 1, "jobId": 1, "status": 0, "job_title": "Go 开发" },
    "profile": {
      "candidateId": 1, "name": "张三", "phone": "13800138000",
      "education": "本科", "university": "北京大学",
      "experience": "...", "skills": "Go,MySQL", "isComplete": true
    },
    "resume": {
      "id": 1, "fileName": "resume.pdf", "fileType": "pdf",
      "url": "https://xxx.oss-cn-hangzhou.aliyuncs.com/..."
    }
  }
}
```

### 12. AI 聊天

- **POST** `/api/hr/chat`
- **请求体**：
```json
{ "content": "目前有多少人投递了岗位？" }
```
- **返回**：
```json
{
  "code": 0, "message": "success",
  "data": { "reply": "目前系统共有 15 位候选人提交了岗位投递..." }
}
```

### 13. 获取聊天历史

- **GET** `/api/hr/chat/history`
- **返回**：
```json
{
  "code": 0, "message": "success",
  "data": {
    "messages": [
      { "id": 1, "hr_id": 1, "role": "user", "content": "...", "created_at": "..." },
      { "id": 2, "hr_id": 1, "role": "assistant", "content": "...", "created_at": "..." }
    ]
  }
}
```

---

## 三、候选人接口（需 JWT，role=candidate）

所有请求需携带 Header：`Authorization: Bearer <token>`

### 14. 获取个人资料

- **GET** `/api/candidate/profile`
- **返回**：
```json
{
  "code": 0, "message": "success",
  "data": {
    "candidateId": 1, "name": "张三", "phone": "13800138000",
    "education": "本科", "university": "北京大学",
    "experience": "...", "skills": "Go,MySQL",
    "isComplete": true
  }
}
```

### 15. 更新个人资料

- **PUT** `/api/candidate/profile`
- **请求体**：
```json
{
  "name": "张三",
  "phone": "13800138000",
  "education": "本科",
  "university": "北京大学",
  "experience": "3年Go开发经验...",
  "skills": "Go,MySQL,Redis"
}
```
- **返回**：
```json
{ "code": 0, "message": "success", "data": { "isComplete": true } }
```

### 16. 上传简历

- **POST** `/api/candidate/resume/upload`
- **Content-Type**：`multipart/form-data`
- **参数**：`file` (PDF/DOC/DOCX)
- **后端校验**：文件后缀 + 文件头魔数
- **返回**：
```json
{ "code": 0, "message": "success", "data": { "id": 1, "ossKey": "resumes/1/..." } }
```

### 17. 获取简历信息

- **GET** `/api/candidate/resume`
- **返回**：
```json
{
  "code": 0, "message": "success",
  "data": {
    "id": 1, "candidateId": 1, "fileName": "resume.pdf",
    "ossKey": "resumes/1/...", "fileSize": 102400,
    "fileType": "pdf",
    "url": "https://xxx.oss-cn-hangzhou.aliyuncs.com/...(签名URL)",
    "createdAt": "2024-01-01 10:00:00"
  }
}
```

### 18. 投递岗位

- **POST** `/api/candidate/apply/:jobId`
- **前置条件**：已完善资料 + 已上传简历
- **返回**：
```json
{ "code": 0, "message": "success", "data": { "id": 1 } }
```

### 19. 我的投递列表

- **GET** `/api/candidate/applications?page=1&pageSize=10`
- **返回**：
```json
{
  "code": 0, "message": "success",
  "data": {
    "applications": [
      {
        "id": 1, "jobId": 1, "candidateId": 1,
        "status": 0, "created_at": "...",
        "job_title": "Go 开发", "job_location": "北京", "job_salary": "15k-25k"
      }
    ],
    "total": 1
  }
}
```
