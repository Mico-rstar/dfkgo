#!/usr/bin/env bash
# =============================================================================
# 模拟前端 STS 直传 OSS 集成测试脚本
#
# 完整流程：注册 → 登录 → 上传初始化(获取 STS) → STS 直传 OSS → 上传回调 → 创建检测任务
#
# 用法：
#   chmod +x scripts/test_sts_upload.sh
#   ./scripts/test_sts_upload.sh [文件路径]
#
# 示例：
#   ./scripts/test_sts_upload.sh ./testdata/sample.jpg
#   ./scripts/test_sts_upload.sh ~/Downloads/test.mp4
#
# 前置条件：
#   1. dfkgo 服务已启动（默认 localhost:8080）
#   2. app.env 中 OSS 配置正确（真实 STS 凭证，非 Mock）
#   3. OSS Bucket 已配置 CORS 规则
#   4. 安装了 curl, jq, md5sum/md5, file 命令
# =============================================================================

set -euo pipefail

# ======================== 配置 ========================
BASE_URL="${BASE_URL:-http://localhost:8080/api}"
TEST_EMAIL="${TEST_EMAIL:-testuser_$(date +%s)@example.com}"
TEST_PASSWORD="${TEST_PASSWORD:-Test1234567}"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

info()    { echo -e "${CYAN}[INFO]${NC} $*"; }
success() { echo -e "${GREEN}[OK]${NC}   $*"; }
warn()    { echo -e "${YELLOW}[WARN]${NC} $*"; }
fail()    { echo -e "${RED}[FAIL]${NC} $*"; exit 1; }

# ======================== 工具函数 ========================
check_deps() {
    for cmd in curl jq file; do
        command -v "$cmd" >/dev/null 2>&1 || fail "缺少依赖命令: $cmd"
    done
    # md5sum (Linux) 或 md5 (macOS)
    if command -v md5sum >/dev/null 2>&1; then
        MD5_CMD="md5sum"
    elif command -v md5 >/dev/null 2>&1; then
        MD5_CMD="md5 -r"
    else
        fail "缺少 md5sum 或 md5 命令"
    fi
}

# 计算文件 MD5
calc_md5() {
    $MD5_CMD "$1" | awk '{print $1}'
}

# 获取文件 MIME 类型
get_mime() {
    file --mime-type -b "$1"
}

# 从 JSON 响应中提取字段，同时检查 code==0
extract() {
    local resp="$1"
    local field="$2"
    local code
    code=$(echo "$resp" | jq -r '.code')
    if [ "$code" != "0" ]; then
        local msg
        msg=$(echo "$resp" | jq -r '.message // "unknown error"')
        fail "API 返回错误 code=$code: $msg"
    fi
    echo "$resp" | jq -r ".data.$field"
}

# ======================== 主流程 ========================
main() {
    local file_path="${1:-}"

    if [ -z "$file_path" ]; then
        # 没有提供文件，创建一个测试图片
        warn "未提供文件路径，将创建临时测试文件"
        file_path="/tmp/dfkgo_test_$(date +%s).jpg"
        # 生成一个最小的合法 JPEG 文件
        printf '\xff\xd8\xff\xe0\x00\x10JFIF\x00\x01\x01\x00\x00\x01\x00\x01\x00\x00\xff\xd9' > "$file_path"
        CREATED_TMP=true
        info "创建临时测试文件: $file_path"
    fi

    if [ ! -f "$file_path" ]; then
        fail "文件不存在: $file_path"
    fi

    check_deps

    local file_name
    file_name=$(basename "$file_path")
    local file_size
    file_size=$(stat -f%z "$file_path" 2>/dev/null || stat -c%s "$file_path" 2>/dev/null)
    local mime_type
    mime_type=$(get_mime "$file_path")
    local md5
    md5=$(calc_md5 "$file_path")

    info "=========================================="
    info "文件信息"
    info "=========================================="
    info "文件名:    $file_name"
    info "大小:      $file_size bytes"
    info "MIME:      $mime_type"
    info "MD5:       $md5"
    echo ""

    # ---------- Step 1: 注册 ----------
    info "[Step 1/6] 注册用户: $TEST_EMAIL"
    local reg_resp
    reg_resp=$(curl -s -X POST "$BASE_URL/auth/register" \
        -H "Content-Type: application/json" \
        -d "{\"email\":\"$TEST_EMAIL\",\"password\":\"$TEST_PASSWORD\"}")
    local reg_code
    reg_code=$(echo "$reg_resp" | jq -r '.code')
    if [ "$reg_code" = "0" ]; then
        success "注册成功"
    else
        warn "注册跳过 (code=$reg_code): $(echo "$reg_resp" | jq -r '.message')，继续后续流程"
    fi

    # ---------- Step 2: 登录 ----------
    info "[Step 2/6] 登录获取 Token"
    local login_resp
    login_resp=$(curl -s -X POST "$BASE_URL/auth/login" \
        -H "Content-Type: application/json" \
        -d "{\"email\":\"$TEST_EMAIL\",\"password\":\"$TEST_PASSWORD\"}")
    local token
    token=$(extract "$login_resp" "token")
    if [ -z "$token" ] || [ "$token" = "null" ]; then
        fail "登录失败，未获取到 token"
    fi
    success "登录成功，Token: ${token:0:20}..."

    local AUTH="Authorization: Bearer $token"

    # ---------- Step 3: 上传初始化 ----------
    info "[Step 3/6] 上传初始化（获取 STS 凭证）"
    local init_resp
    init_resp=$(curl -s -X POST "$BASE_URL/upload/init" \
        -H "Content-Type: application/json" \
        -H "$AUTH" \
        -d "{\"fileName\":\"$file_name\",\"fileSize\":$file_size,\"mimeType\":\"$mime_type\",\"md5\":\"$md5\"}")

    local file_id hit
    file_id=$(extract "$init_resp" "fileId")
    hit=$(extract "$init_resp" "hit")

    if [ "$hit" = "true" ]; then
        success "秒传命中! fileId=$file_id"
        info "文件已存在于 OSS，跳过上传步骤"
        local oss_url
        oss_url=$(extract "$init_resp" "ossUrl")
        info "OSS URL: $oss_url"
        skip_upload=true
    else
        success "获取 STS 凭证成功，fileId=$file_id"
        skip_upload=false

        # 提取 STS 信息
        local ak_id ak_secret sec_token bucket region endpoint object_key
        ak_id=$(echo "$init_resp" | jq -r '.data.sts.accessKeyId')
        ak_secret=$(echo "$init_resp" | jq -r '.data.sts.accessKeySecret')
        sec_token=$(echo "$init_resp" | jq -r '.data.sts.securityToken')
        bucket=$(echo "$init_resp" | jq -r '.data.sts.bucket')
        region=$(echo "$init_resp" | jq -r '.data.sts.region')
        endpoint=$(echo "$init_resp" | jq -r '.data.sts.endpoint')
        object_key=$(echo "$init_resp" | jq -r '.data.sts.objectKey')

        info "  AK ID:      ${ak_id:0:10}..."
        info "  Bucket:     $bucket"
        info "  Region:     $region"
        info "  Endpoint:   $endpoint"
        info "  ObjectKey:  $object_key"
    fi

    # ---------- Step 4: STS 直传 OSS ----------
    if [ "$skip_upload" = "false" ]; then
        info "[Step 4/6] STS 直传文件到 OSS"

        # 构造签名所需参数
        local oss_host="https://${bucket}.${endpoint}"
        local date_str
        date_str=$(date -u +"%a, %d %b %Y %H:%M:%S GMT")
        local content_type="$mime_type"

        # 使用 PUT 方式上传到 OSS（V1 签名）
        local string_to_sign="PUT\n\n${content_type}\n${date_str}\nx-oss-security-token:${sec_token}\n/${bucket}/${object_key}"
        local signature
        signature=$(printf '%b' "$string_to_sign" | openssl dgst -sha1 -hmac "$ak_secret" -binary | base64)

        local put_resp_code
        put_resp_code=$(curl -s -o /dev/null -w "%{http_code}" \
            -X PUT "${oss_host}/${object_key}" \
            -H "Date: ${date_str}" \
            -H "Content-Type: ${content_type}" \
            -H "Authorization: OSS ${ak_id}:${signature}" \
            -H "x-oss-security-token: ${sec_token}" \
            --data-binary "@${file_path}")

        if [ "$put_resp_code" = "200" ]; then
            success "OSS 上传成功 (HTTP $put_resp_code)"
        else
            fail "OSS 上传失败 (HTTP $put_resp_code)，请检查 STS 凭证和 Bucket CORS 配置"
        fi
    else
        info "[Step 4/6] 跳过（秒传命中）"
    fi

    # ---------- Step 5: 上传回调 ----------
    if [ "$skip_upload" = "false" ]; then
        info "[Step 5/6] 上传回调确认"
        local cb_resp
        cb_resp=$(curl -s -X POST "$BASE_URL/upload/callback" \
            -H "Content-Type: application/json" \
            -H "$AUTH" \
            -d "{\"fileId\":\"$file_id\"}")
        local cb_oss_url
        cb_oss_url=$(extract "$cb_resp" "ossUrl")
        success "回调成功，ossUrl=$cb_oss_url"
    else
        info "[Step 5/6] 跳过（秒传命中）"
    fi

    # ---------- Step 6: 创建检测任务 ----------
    # 根据 MIME 类型推断 modality
    local modality
    case "$mime_type" in
        image/*) modality="image" ;;
        video/*) modality="video" ;;
        audio/*) modality="audio" ;;
        *)       modality="image"; warn "无法推断 modality，默认使用 image" ;;
    esac

    info "[Step 6/6] 创建检测任务 (modality=$modality)"
    local task_resp
    task_resp=$(curl -s -X POST "$BASE_URL/tasks" \
        -H "Content-Type: application/json" \
        -H "$AUTH" \
        -d "{\"fileId\":\"$file_id\",\"modality\":\"$modality\"}")
    local task_id task_status
    task_id=$(extract "$task_resp" "taskId")
    task_status=$(extract "$task_resp" "status")
    success "任务创建成功，taskId=$task_id, status=$task_status"

    # ---------- 轮询任务状态 ----------
    echo ""
    info "=========================================="
    info "轮询任务状态（最多等待 60 秒）"
    info "=========================================="
    local max_wait=60
    local elapsed=0
    local interval=3
    while [ $elapsed -lt $max_wait ]; do
        local status_resp
        status_resp=$(curl -s -X GET "$BASE_URL/tasks/$task_id" \
            -H "$AUTH")
        local status
        status=$(extract "$status_resp" "status")
        info "[$elapsed s] 状态: $status"

        if [ "$status" = "completed" ] || [ "$status" = "failed" ]; then
            echo ""
            if [ "$status" = "completed" ]; then
                success "任务完成!"
            else
                warn "任务失败"
            fi
            # 获取结果
            local result_resp
            result_resp=$(curl -s -X GET "$BASE_URL/tasks/$task_id/result" \
                -H "$AUTH")
            info "任务结果:"
            echo "$result_resp" | jq '.data'
            break
        fi

        sleep $interval
        elapsed=$((elapsed + interval))
    done

    if [ $elapsed -ge $max_wait ]; then
        warn "等待超时，任务仍在处理中。可手动查询: curl -H \"$AUTH\" $BASE_URL/tasks/$task_id"
    fi

    # 清理临时文件
    if [ "${CREATED_TMP:-false}" = "true" ]; then
        rm -f "$file_path"
        info "已清理临时文件"
    fi

    echo ""
    info "=========================================="
    success "集成测试完成!"
    info "=========================================="
    info "fileId:  $file_id"
    info "taskId:  $task_id"
    info ""
    info "后续手动测试命令:"
    info "  查询状态: curl -s -H \"$AUTH\" $BASE_URL/tasks/$task_id | jq"
    info "  查询结果: curl -s -H \"$AUTH\" $BASE_URL/tasks/$task_id/result | jq"
    info "  取消任务: curl -s -X POST -H \"$AUTH\" $BASE_URL/tasks/$task_id/cancel | jq"
    info "  查看历史: curl -s -H \"$AUTH\" \"$BASE_URL/history?page=1&pageSize=10\" | jq"
}

main "$@"
