#!/usr/bin/env bash
# =============================================================================
# TRH SDK — AWS vs DigitalOcean Deployment Comparison Demo
#
# 사용법: bash demo/provider-comparison.sh
#        bash demo/provider-comparison.sh --section credentials
#        bash demo/provider-comparison.sh --section infra
#        bash demo/provider-comparison.sh --section steps
#        bash demo/provider-comparison.sh --section destroy
# =============================================================================

set -euo pipefail

# ─── 색상 (NO_COLOR 또는 비-TTY stdout 환경에서 자동 비활성화) ─────────────
if [[ -t 1 && -z "${NO_COLOR:-}" ]]; then
  BOLD="\033[1m"
  RESET="\033[0m"
  CYAN="\033[36m"
  YELLOW="\033[33m"
  GREEN="\033[32m"
  ORANGE="\033[38;5;214m"
  DIM="\033[2m"
  WHITE="\033[97m"
else
  BOLD="" RESET="" CYAN="" YELLOW="" GREEN="" ORANGE="" DIM="" WHITE=""
fi

# ─── 레이아웃 상수 ─────────────────────────────────────────────────────────
DIVIDER="$(printf '%.0s─' {1..76})"
HALF="$(printf '%.0s─' {1..36})"

# ─── 헬퍼 ──────────────────────────────────────────────────────────────────
pause() {
  # non-interactive 환경(CI, 파이프)에서는 입력 대기 없이 통과한다.
  if [[ ! -t 0 ]]; then return; fi
  echo ""
  read -rp "$(echo -e "${DIM}  [ Enter 키를 눌러 계속 ]${RESET}")" _
  echo ""
}

header() {
  echo ""
  echo -e "${BOLD}${CYAN}${DIVIDER}${RESET}"
  echo -e "${BOLD}${CYAN}  $1${RESET}"
  echo -e "${BOLD}${CYAN}${DIVIDER}${RESET}"
}

side_by_side_header() {
  printf "\n${BOLD}  %-36s  %-36s${RESET}\n" "☁  AWS" "🌊 DigitalOcean"
  printf "  ${YELLOW}${HALF}${RESET}  ${ORANGE}${HALF}${RESET}\n"
}

row() {
  # row <aws_label> <aws_value> <do_label> <do_value>
  local aws_label="$1" aws_value="$2" do_label="$3" do_value="$4"
  printf "  ${DIM}%-16s${RESET} ${YELLOW}%-19s${RESET}  ${DIM}%-16s${RESET} ${ORANGE}%-19s${RESET}\n" \
    "${aws_label}:" "${aws_value}" "${do_label}:" "${do_value}"
}

# ─── SECTIONS ──────────────────────────────────────────────────────────────

section_intro() {
  # clear은 TTY에서만 실행 (파이프/리다이렉트 환경에서 제어코드 출력 방지)
  [[ -t 1 ]] && clear
  echo ""
  echo -e "${BOLD}${WHITE}"
  cat << 'EOF'
  ████████╗██████╗ ██╗  ██╗    ██████╗ ███████╗███╗   ███╗ ██████╗
     ██╔══╝██╔══██╗██║  ██║    ██╔══██╗██╔════╝████╗ ████║██╔═══██╗
     ██║   ██████╔╝███████║    ██║  ██║█████╗  ██╔████╔██║██║   ██║
     ██║   ██╔══██╗██╔══██║    ██║  ██║██╔══╝  ██║╚██╔╝██║██║   ██║
     ██║   ██║  ██║██║  ██║    ██████╔╝███████╗██║ ╚═╝ ██║╚██████╔╝
     ╚═╝   ╚═╝  ╚═╝╚═╝  ╚═╝   ╚═════╝ ╚══════╝╚═╝     ╚═╝ ╚═════╝
EOF
  echo -e "${RESET}"
  echo -e "  ${BOLD}Tokamak Rollup Hub SDK${RESET} — Cloud Provider Comparison Demo"
  echo ""
  echo -e "  ${DIM}L2 체인을 AWS와 DigitalOcean 중 어디에 배포하시겠습니까?${RESET}"
  echo -e "  ${DIM}이 데모는 두 경로의 절차적 차이를 단계별로 보여줍니다.${RESET}"
  echo ""
  echo -e "  ${BOLD}섹션 목록:${RESET}"
  echo -e "    1. 사전 요구사항 및 자격증명"
  echo -e "    2. 인프라 구성 비교"
  echo -e "    3. 배포 단계 흐름"
  echo -e "    4. 파괴(Destroy) 흐름"
  echo -e "    5. 선택 가이드 (비용·복잡도 비교)"
  pause
}

section_credentials() {
  header "1. 사전 요구사항 & 자격증명 입력"

  echo ""
  echo -e "  ${BOLD}CLI 도구 요구사항${RESET}"
  side_by_side_header
  printf "  ${YELLOW}%-36s${RESET}  ${ORANGE}%-36s${RESET}\n" \
    "terraform, helm, aws, kubectl" \
    "terraform, helm, doctl, kubectl"
  echo ""
  echo -e "  ${DIM}  차이: aws CLI → doctl (DigitalOcean 전용 CLI)${RESET}"

  echo ""
  echo -e "  ${BOLD}자격증명 입력 프롬프트${RESET}  ${DIM}(trh deploy 실행 시)${RESET}"

  side_by_side_header

  printf "\n  ${DIM}[1/3]${RESET} ${YELLOW}AWS Access Key ID      ${RESET}  ${DIM}[1/4]${RESET} ${ORANGE}DigitalOcean API Token ${RESET}\n"
  printf "  ${YELLOW}      ▸ IAM 사용자 액세스 키  ${RESET}  ${ORANGE}      ▸ PAT (개인 액세스 토큰)${RESET}\n"
  printf "  ${YELLOW}      ▸ 형식: AKIA...         ${RESET}  ${ORANGE}      ▸ 토큰 검증: doctl API  ${RESET}\n"

  printf "\n  ${DIM}[2/3]${RESET} ${YELLOW}AWS Secret Access Key  ${RESET}  ${DIM}[2/4]${RESET} ${ORANGE}Region (기본: nyc3)    ${RESET}\n"
  printf "  ${YELLOW}      ▸ IAM 시크릿 키         ${RESET}  ${ORANGE}      ▸ 유효 리전 자동 검증  ${RESET}\n"

  printf "\n  ${DIM}[3/3]${RESET} ${YELLOW}AWS Region             ${RESET}  ${DIM}[3/4]${RESET} ${ORANGE}Spaces Access Key      ${RESET}\n"
  printf "  ${YELLOW}      ▸ 기본: ap-northeast-2  ${RESET}  ${ORANGE}      ▸ HMAC 키 (S3 호환)    ${RESET}\n"
  printf "  ${YELLOW}      ▸ 리전별 AZ 자동 조회   ${RESET}  ${ORANGE}      ▸ DO API 토큰과 별개   ${RESET}\n"

  printf "\n  ${DIM}     (없음)               ${RESET}  ${DIM}[4/4]${RESET} ${ORANGE}Spaces Secret Key      ${RESET}\n"
  printf "  ${YELLOW}                             ${RESET}  ${ORANGE}      ▸ 터미널 에코 마스킹   ${RESET}\n"
  printf "  ${YELLOW}                             ${RESET}  ${ORANGE}      ▸ 디스크 저장 안 함    ${RESET}\n"

  echo ""
  echo -e "  ${BOLD}⚠  핵심 차이점${RESET}"
  echo ""
  echo -e "  ${YELLOW}AWS${RESET}  : IAM 자격증명 1쌍으로 배포+상태 저장 모두 처리"
  echo -e "  ${ORANGE}DO${RESET}   : API 토큰(배포용) + Spaces HMAC 키(상태 저장용) — 2종 필요"
  echo ""
  echo -e "  ${DIM}  Terraform 상태를 DO Spaces(S3 호환)에 저장하기 위해 별도의"
  echo -e "  HMAC 자격증명이 필요합니다. DO API 토큰과 혼용하면 인증 실패.${RESET}"
  pause
}

section_infra() {
  header "2. 생성되는 인프라 비교"

  echo ""
  side_by_side_header
  echo ""

  row "Kubernetes"  "EKS (AWS 관리형)"       "Kubernetes"  "DOKS (DO 관리형)"
  row "네트워크"    "VPC + Subnets + IGW"    "네트워크"    "DigitalOcean VPC"
  row "데이터베이스" "RDS PostgreSQL"        "데이터베이스" "Managed PostgreSQL"
  row "스토리지"    "EFS (NFS 마운트)"       "스토리지"    "DOKS CSI 드라이버"
  row "Helm 상태"   "S3 버킷"               "Helm 상태"   "DO Spaces 버킷"
  row "시크릿 관리" "AWS Secrets Manager"   "시크릿 관리" "K8s Secret (직접)"
  row "백업"        "EFS + DynamoDB"        "백업"        "미지원"
  row "모니터링"    "지원"                  "모니터링"    "지원"

  echo ""
  echo -e "  ${BOLD}Terraform 모듈 구조${RESET}"
  echo ""
  printf "  ${YELLOW}AWS${RESET} ${DIM}(terraform/aws/thanos-stack/)${RESET}\n"
  printf "  ${YELLOW}├── modules/vpc/${RESET}          VPC, 서브넷, 라우팅\n"
  printf "  ${YELLOW}├── modules/eks/${RESET}          EKS 클러스터 + 노드 그룹\n"
  printf "  ${YELLOW}├── modules/rds/${RESET}          PostgreSQL 인스턴스\n"
  printf "  ${YELLOW}├── modules/efs/${RESET}          EFS 파일시스템 + 마운트 타겟\n"
  printf "  ${YELLOW}├── modules/kubernetes/${RESET}   K8s 네임스페이스, RBAC\n"
  printf "  ${YELLOW}├── modules/secretsmanager/${RESET} 프라이빗 키 저장\n"
  printf "  ${YELLOW}└── modules/chain-config/${RESET} 체인 설정 ConfigMap\n"
  echo ""
  printf "  ${ORANGE}DigitalOcean${RESET} ${DIM}(terraform/digitalocean/thanos-stack/)${RESET}\n"
  printf "  ${ORANGE}└── main.tf${RESET}               VPC + DOKS + PostgreSQL 통합\n"
  printf "  ${ORANGE}                          ${RESET}${DIM}(모듈 없음 — 플랫 구조)${RESET}\n"
  echo ""
  echo -e "  ${BOLD}⚠  핵심 차이점${RESET}"
  echo ""
  echo -e "  ${YELLOW}AWS${RESET}  : 7개 독립 모듈 — 컴포넌트별 세밀한 제어 가능"
  echo -e "  ${ORANGE}DO${RESET}   : 단일 main.tf — 단순하지만 컴포넌트 분리 불가"
  echo ""
  echo -e "  ${YELLOW}AWS${RESET}  : EFS로 노드 간 공유 스토리지 + 백업 자동화 지원"
  echo -e "  ${ORANGE}DO${RESET}   : DOKS 내장 CSI 드라이버 사용 (EFS 상당 기능 없음)"
  pause
}

section_steps() {
  header "3. 배포 단계 흐름 (trh deploy)"

  echo ""
  printf "  ${YELLOW}%-3s %-30s${RESET}  ${ORANGE}%-3s %-30s${RESET}\n" \
    "AWS" "단계" "DO" "단계"
  printf "  ${YELLOW}${HALF}${RESET}  ${ORANGE}${HALF}${RESET}\n\n"

  # Steps comparison
  printf "  ${YELLOW}1.${RESET}  의존성 확인                   ${ORANGE}1.${RESET}  의존성 확인\n"
  printf "  ${DIM}    terraform,helm,aws,kubectl  ${RESET}  ${DIM}    terraform,helm,doctl,kubectl${RESET}\n\n"

  printf "  ${YELLOW}2.${RESET}  .envrc 생성 (20+ 변수)        ${ORANGE}2.${RESET}  DO 설정 저장 (settings.json)\n"
  printf "  ${DIM}    IAM ARN, AZ, 백업 크론 등  ${RESET}  ${DIM}    토큰, 리전, Spaces 키${RESET}\n\n"

  printf "  ${YELLOW}3.${RESET}  config 파일 복사              ${ORANGE}3.${RESET}  config 파일 복사\n"
  printf "  ${DIM}    rollup.json, genesis.json   ${RESET}  ${DIM}    rollup.json, genesis.json${RESET}\n\n"

  printf "  ${YELLOW}4.${RESET}  Terraform backend 초기화      ${ORANGE}4.${RESET}  DO Spaces 버킷 생성\n"
  printf "  ${DIM}    S3 버킷 → 상태 저장         ${RESET}  ${DIM}    terraform apply (local state)${RESET}\n\n"

  printf "  ${YELLOW}5.${RESET}  EKS + VPC + RDS + EFS 배포   ${ORANGE}5.${RESET}  DOKS + VPC + PostgreSQL 배포\n"
  printf "  ${DIM}    terraform apply (S3 backend)${RESET}  ${DIM}    terraform apply (Spaces backend)${RESET}\n\n"

  printf "  ${YELLOW}6.${RESET}  EKS kubeconfig 설정           ${ORANGE}6.${RESET}  doctl kubeconfig 저장\n"
  printf "  ${DIM}    aws eks update-kubeconfig    ${RESET}  ${DIM}    doctl k8s cluster kubeconfig save${RESET}\n\n"

  printf "  ${YELLOW}7.${RESET}  K8s 클러스터 Ready 대기       ${ORANGE}7.${RESET}  K8s 클러스터 Ready 대기\n\n"

  printf "  ${YELLOW}8.${RESET}  Helm 차트 설치                ${ORANGE}8.${RESET}  Helm 차트 설치\n"
  printf "  ${DIM}    PVC 전용 → Deployment 순서  ${RESET}  ${DIM}    PVC 전용 → Deployment 순서${RESET}\n\n"

  printf "  ${DIM}    (없음)                      ${RESET}  ${ORANGE}9.${RESET}  L2 RPC 엔드포인트 대기\n"
  printf "  ${YELLOW}8.2${RESET} L2 RPC 엔드포인트 대기        ${DIM}    (10분 타임아웃, 15초 간격)${RESET}\n\n"

  printf "  ${YELLOW}8.3${RESET} 브릿지 설치 (선택)            ${ORANGE}10.${RESET} 브릿지 설치 (선택)\n\n"

  printf "  ${DIM}    백업 초기화 (EFS + DynamoDB) ${RESET}  ${DIM}    (백업 미지원)${RESET}\n\n"

  echo -e "  ${BOLD}⚠  핵심 차이점${RESET}"
  echo ""
  echo -e "  ${YELLOW}AWS${RESET}  : .envrc 파일로 20+ Terraform 변수 관리"
  echo -e "  ${ORANGE}DO${RESET}   : 환경 변수를 Go에서 직접 cmd.Env로 주입 (ps aux 노출 없음)"
  echo ""
  echo -e "  ${YELLOW}AWS${RESET}  : 단일 terraform apply로 전체 인프라"
  echo -e "  ${ORANGE}DO${RESET}   : 2단계 — ① Spaces 버킷 먼저 생성 ② 메인 인프라 배포"
  pause
}

section_destroy() {
  header "4. 파괴(Destroy) 흐름 (trh destroy)"

  echo ""
  printf "  ${YELLOW}AWS 파괴 순서${RESET}                        ${ORANGE}DigitalOcean 파괴 순서${RESET}\n"
  printf "  ${YELLOW}${HALF}${RESET}  ${ORANGE}${HALF}${RESET}\n\n"

  printf "  ${YELLOW}1.${RESET} 백업 리소스 정리                ${ORANGE}1.${RESET} Helm 릴리즈 언인스톨\n"
  printf "  ${DIM}   EFS 백업, DynamoDB 항목 삭제   ${RESET}  ${DIM}   네임스페이스 내 릴리즈 제거${RESET}\n\n"

  printf "  ${YELLOW}2.${RESET} Helm 릴리즈 언인스톨            ${ORANGE}2.${RESET} K8s 네임스페이스 삭제\n"
  printf "  ${DIM}   네임스페이스 + 모니터링 포함    ${RESET}  ${DIM}   (5분 타임아웃)${RESET}\n\n"

  printf "  ${YELLOW}3.${RESET} 모니터링 언인스톨\n"
  printf "  ${DIM}   uptime-service 포함${RESET}\n\n"

  printf "  ${YELLOW}4.${RESET} K8s 네임스페이스 삭제           ${ORANGE}3.${RESET} terraform destroy\n"
  printf "  ${DIM}   (5분 타임아웃)                  ${RESET}  ${DIM}   Spaces 인증: AWS_ACCESS_KEY_ID${RESET}\n\n"

  printf "  ${YELLOW}5.${RESET} EFS 마운트 타겟 삭제\n"
  printf "  ${DIM}   ⚠ 서브넷 삭제 전 필수!${RESET}\n\n"

  printf "  ${YELLOW}6.${RESET} terraform destroy\n"
  printf "  ${DIM}   EFS 삭제 후 30초 대기 필요${RESET}\n\n"

  printf "  ${YELLOW}7.${RESET} 고아 리소스 검증 + 정리\n"
  printf "  ${DIM}   남은 AWS 리소스 스캔${RESET}\n\n"

  echo -e "  ${BOLD}⚠  핵심 차이점${RESET}"
  echo ""
  echo -e "  ${YELLOW}AWS${RESET}  : 7단계 — EFS 마운트 타겟 선삭제 필수 (서브넷 의존성)"
  echo -e "  ${ORANGE}DO${RESET}   : 3단계 — Helm → 네임스페이스 → terraform destroy"
  echo ""
  echo -e "  ${YELLOW}AWS${RESET}  : 백업, 모니터링, 고아 리소스 별도 정리 필요"
  echo -e "  ${ORANGE}DO${RESET}   : terraform destroy가 모든 리소스 일괄 제거"
  echo ""
  echo -e "  ${ORANGE}DO${RESET}  : Spaces Secret Key는 저장 안 함 → destroy 시 재입력"
  pause
}

section_summary() {
  header "5. 선택 가이드"

  echo ""
  printf "  %-30s  ${YELLOW}%-20s${RESET}  ${ORANGE}%-20s${RESET}\n" \
    "기준" "AWS" "DigitalOcean"
  printf "  ${DIVIDER}\n"

  printf "  %-30s  ${YELLOW}%-20s${RESET}  ${ORANGE}%-20s${RESET}\n" \
    "설정 복잡도" "높음 (20+ 변수)" "낮음 (4개 입력)"
  printf "  %-30s  ${YELLOW}%-20s${RESET}  ${ORANGE}%-20s${RESET}\n" \
    "배포 시간" "~25분" "~20분"
  printf "  %-30s  ${YELLOW}%-20s${RESET}  ${ORANGE}%-20s${RESET}\n" \
    "파괴 복잡도" "높음 (7단계)" "낮음 (3단계)"
  printf "  %-30s  ${YELLOW}%-20s${RESET}  ${ORANGE}%-20s${RESET}\n" \
    "백업 지원" "✅ EFS + DynamoDB" "❌ 미지원"
  printf "  %-30s  ${YELLOW}%-20s${RESET}  ${ORANGE}%-20s${RESET}\n" \
    "모니터링" "✅ 지원" "✅ 지원"
  printf "  %-30s  ${YELLOW}%-20s${RESET}  ${ORANGE}%-20s${RESET}\n" \
    "인프라 커스터마이징" "높음 (모듈 단위)" "낮음 (단일 파일)"
  printf "  %-30s  ${YELLOW}%-20s${RESET}  ${ORANGE}%-20s${RESET}\n" \
    "자격증명 개수" "1쌍 (IAM)" "2종 (토큰 + Spaces)"
  printf "  %-30s  ${YELLOW}%-20s${RESET}  ${ORANGE}%-20s${RESET}\n" \
    "예상 월 비용 (소형)" "~\$150+" "~\$80+"
  printf "  %-30s  ${YELLOW}%-20s${RESET}  ${ORANGE}%-20s${RESET}\n" \
    "권장 대상" "프로덕션, 기업" "개발, 테스트, 스타트업"

  echo ""
  echo -e "  ${BOLD}실제 배포 명령어${RESET}"
  echo ""
  echo -e "  ${DIM}# 두 경우 모두 동일한 CLI 진입점${RESET}"
  echo -e "  ${GREEN}\$ trh deploy${RESET}"
  echo -e "  ${DIM}  → \"Which infrastructure provider?\" 선택 화면${RESET}"
  echo -e "  ${DIM}     1) aws${RESET}"
  echo -e "  ${DIM}     2) digitalocean${RESET}"
  echo ""
  echo -e "  ${DIM}# 파괴도 동일${RESET}"
  echo -e "  ${GREEN}\$ trh destroy${RESET}"
  echo ""
  echo -e "  ${BOLD}${GREEN}  ✅ 사용자는 같은 CLI로 두 클라우드를 동일하게 다룹니다.${RESET}"
  echo -e "  ${BOLD}${GREEN}     추상화가 복잡성을 숨겨줍니다.${RESET}"
  echo ""
}

# ─── MAIN ──────────────────────────────────────────────────────────────────

SECTION="${1:-all}"

case "$SECTION" in
  --section)
    SECTION="${2:-all}"
    ;;
esac

case "$SECTION" in
  credentials)
    section_credentials
    ;;
  infra)
    section_infra
    ;;
  steps)
    section_steps
    ;;
  destroy)
    section_destroy
    ;;
  summary)
    section_summary
    ;;
  all|*)
    section_intro
    section_credentials
    section_infra
    section_steps
    section_destroy
    section_summary
    ;;
esac
