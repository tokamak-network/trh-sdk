#!/usr/bin/env bash
# demo/live-comparison.sh — Split-screen AWS vs DigitalOcean deployment simulation
#
# 사용법: bash demo/live-comparison.sh

set -uo pipefail

# ── Terminal guard ────────────────────────────────────────────────────────────
if [[ ! -t 1 ]]; then
  echo "이 데모는 인터랙티브 터미널이 필요합니다." >&2; exit 1
fi
COLS=$(tput cols)
ROWS=$(tput lines)
if (( COLS < 88 )); then
  echo "터미널 너비가 최소 88칸 필요합니다 (현재: ${COLS}칸)." >&2; exit 1
fi

clear

# ── Colors ────────────────────────────────────────────────────────────────────
if [[ -z "${NO_COLOR:-}" ]]; then
  B="\033[1m"   RST="\033[0m"  CY="\033[36m"   YE="\033[38;5;220m"
  GR="\033[32m" DIM="\033[2m"  WH="\033[97m"   OR="\033[38;5;208m"
  RE="\033[31m" AQ="\033[96m"  MA="\033[35m"   BL="\033[34m"
else
  B="" RST="" CY="" YE="" GR="" DIM="" WH="" OR="" RE="" AQ="" MA="" BL=""
fi

# ── Layout ────────────────────────────────────────────────────────────────────
LC=2          # AWS (left) column x-start
RC=46         # DO  (right) column x-start

# ── Primitives ────────────────────────────────────────────────────────────────
at()    { tput cup "$1" "$2"; }
sp()    { sleep "$1"; }
printl(){ at "$1" $LC;  printf "%b" "$2"; printf "%b" "$RST"; }
printr(){ at "$1" $RC;  printf "%b" "$2"; printf "%b" "$RST"; }
ok()    { printf "%b✅ %b" "$GR" "$RST"; }
run()   { printf "%b⟳  %b" "$YE" "$RST"; }

type_at() {   # typewriter: row col text [delay]
  at "$1" "$2"
  local t="$3" d=${4:-0.038} i
  for ((i=0; i<${#t}; i++)); do printf "%s" "${t:$i:1}"; sleep "$d"; done
}

pbar() {   # progress bar: row col pct [width]
  local r=$1 c=$2 pct=$3 w=${4:-20}
  local f=$(( w * pct / 100 )) e=$(( w - w * pct / 100 ))
  local filled empty
  filled=$(printf "%${f}s" 2>/dev/null | tr ' ' '█') || filled=""
  empty=$(printf "%${e}s" 2>/dev/null  | tr ' ' '░') || empty=""
  at "$r" "$c"
  printf "%b[%b%s%b%s%b] %b%3d%%%b" \
    "$CY" "$GR" "$filled" "$DIM" "$empty" "$CY" "$WH" "$pct" "$RST"
}

apbar() {   # animate progress bar: row col from to [delay] [width]
  local r=$1 c=$2 from=$3 to=$4 d=${5:-0.07} w=${6:-20} p
  for ((p=from; p<=to; p+=5)); do pbar "$r" "$c" "$p" "$w"; sp "$d"; done
  pbar "$r" "$c" "$to" "$w"
}

mask() {   # masked typing: row col text
  at "$1" "$2"; local t="$3" i
  for ((i=0; i<${#t}; i++)); do printf "●"; sp 0.045; done
}

hr() {   # horizontal rule at row
  at "$1" 0; printf "%b" "$DIM"
  printf '%*s' "$COLS" '' | tr ' ' '─'
  printf "%b" "$RST"
}

banner() {
  at "$1" 0; printf "%b" "$B$CY"
  printf '%*s' "$COLS" '' | tr ' ' '═'
  printf "%b" "$RST"
}

center() {   # centered text: row text color
  local r=$1 txt="$2" c=${3:-$B$WH}
  local pad=$(( (COLS - ${#txt}) / 2 ))
  at "$r" "$pad"; printf "%b%s%b" "$c" "$txt" "$RST"
}

col_header() {   # column headers: row left_text right_text
  at "$1" $LC;  printf "%b%s%b" "$B$YE" "$2" "$RST"
  at "$1" $RC;  printf "%b%s%b" "$B$AQ" "$3" "$RST"
}

step() {   # step label both columns: row aws_label aws_step do_label do_step
  at "$1" $LC; printf "%b[%s/%s]%b %s" "$DIM" "$2" "$3" "$RST" "$4"
  at "$1" $RC; printf "%b[%s/%s]%b %s" "$DIM" "$5" "$6" "$RST" "$7"
}

pause_exit() {
  at $(( ROWS - 1 )) 0
  printf "%b  [ Enter 키를 눌러 종료 ]%b" "$DIM" "$RST"
  read -r _
}

# ══════════════════════════════════════════════════════════════════════════════
# SCENE 0 — Header
# ══════════════════════════════════════════════════════════════════════════════
banner 0
center 1 "🚀  TRH SDK — Live Deployment Simulation"
center 2 "aws  vs  digitalocean  (동시 진행)"
banner 3
col_header 4 "☁  AWS  (EKS · S3 · RDS · EFS)" "🌊 DigitalOcean  (DOKS · Spaces · PostgreSQL)"
hr 5
sp 0.8

# ══════════════════════════════════════════════════════════════════════════════
# SCENE 1 — Command + Provider Selection
# ══════════════════════════════════════════════════════════════════════════════
type_at 6 $LC "${DIM}$${RST} " 0.01
type_at 6 $(( LC + 2 )) "${YE}trh deploy${RST}" 0.07

type_at 6 $RC "${DIM}$${RST} " 0.01
type_at 6 $(( RC + 2 )) "${AQ}trh deploy${RST}" 0.07
sp 0.5

at 7 $LC; printf "%bWhich provider?%b [aws/digitalocean]: %baws%b" "$DIM" "$RST" "$YE" "$RST"
at 7 $RC; printf "%bWhich provider?%b [aws/digitalocean]: %bdigitalocean%b" "$DIM" "$RST" "$AQ" "$RST"
sp 0.9

# ══════════════════════════════════════════════════════════════════════════════
# SCENE 2 — Credentials
# ══════════════════════════════════════════════════════════════════════════════
hr 8
at 9 $LC; printf "%b📋 자격증명 입력%b  %b(3개)%b" "$B$WH" "$RST" "$DIM" "$RST"
at 9 $RC; printf "%b📋 자격증명 입력%b  %b(4개 — API 토큰 + Spaces HMAC)%b" "$B$WH" "$RST" "$DIM" "$RST"
sp 0.4

# Row 10: cred 1
at 10 $LC;  printf "%b[1/3]%b Access Key ID   " "$DIM" "$RST"; type_at 10 $(( LC+21 )) "AKIA" 0.07; mask 10 $(( LC+25 )) "XXXXXXXXXXXX"
at 10 $RC;  printf "%b[1/4]%b DO API Token    " "$DIM" "$RST"; mask 10 $(( RC+20 )) "dop_v1_xxxxxxxxxxxxxxxxxxx"
sp 0.3

# Row 11: cred 2
at 11 $LC;  printf "%b[2/3]%b Secret Key      " "$DIM" "$RST"; mask 11 $(( LC+19 )) "XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"
at 11 $RC;  printf "%b[2/4]%b Region          " "$DIM" "$RST"; type_at 11 $(( RC+19 )) "${GR}nyc3${RST}" 0.06; at 11 $(( RC+24 )); printf " %b(기본값)%b" "$DIM" "$RST"
sp 0.3

# Row 12: cred 3
at 12 $LC;  printf "%b[3/3]%b Region          " "$DIM" "$RST"; type_at 12 $(( LC+19 )) "${GR}ap-northeast-2${RST}" 0.05
at 12 $RC;  printf "%b[2/4]%b Spaces Key      " "$DIM" "$RST"; mask 12 $(( RC+19 )) "DO00XXXXXXXXXXXXXXXXXXXX"
sp 0.3

# Row 13: AWS done, DO cred 4
at 13 $LC;  printf "     %b✅ AWS 인증 완료%b" "$GR" "$RST"
at 13 $RC;  printf "%b[4/4]%b Spaces Secret   " "$DIM" "$RST"; mask 13 $(( RC+19 )) "XXXXXXXXXXXXXXXXXXXXXXXXX"; printf " %b← 디스크 저장 안 함%b" "$DIM" "$RST"
sp 0.3

at 14 $RC;  printf "     %b✅ DO 인증 완료%b  %b(doctl token 검증)%b" "$GR" "$RST" "$DIM" "$RST"
sp 0.7

# ══════════════════════════════════════════════════════════════════════════════
# SCENE 3 — Dependency Check
# ══════════════════════════════════════════════════════════════════════════════
hr 15
step 16  "1" "8" "의존성 확인  terraform · helm · aws · kubectl" \
             "1" "9" "의존성 확인  terraform · helm · doctl · kubectl"
sp 0.3

declare -a DEPS_AWS=("terraform" "helm" "aws" "kubectl")
declare -a DEPS_DO=("terraform" "helm" "doctl" "kubectl")
ROW=17
for i in 0 1 2 3; do
  at $ROW $LC;  run; printf "%b%s%b" "$WH" "${DEPS_AWS[$i]}" "$RST"
  at $ROW $RC;  run; printf "%b%s%b" "$WH" "${DEPS_DO[$i]}"  "$RST"
  sp 0.25
  at $ROW $LC;  ok;  printf "%b%s%b" "$GR" "${DEPS_AWS[$i]}" "$RST"
  at $ROW $RC;  ok;  printf "%b%s%b" "$GR" "${DEPS_DO[$i]}"  "$RST"
  sp 0.15
  (( ROW++ ))
done
sp 0.5

# ══════════════════════════════════════════════════════════════════════════════
# SCENE 4 — Setup
# ══════════════════════════════════════════════════════════════════════════════
hr 21
step 22  "2" "8" ".envrc 생성 (Terraform 변수 20개)" \
             "2" "9" "settings.json 저장 (4개 입력)"
sp 0.3
at 23 $LC; run; printf "%bAWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY,...%b" "$DIM" "$RST"
at 23 $RC; run; printf "%bdo_token, region, spaces_access_key%b" "$DIM" "$RST"
sp 0.5
at 23 $LC; ok;  printf "%b20개 변수 .envrc 기록 완료%b" "$GR" "$RST"
at 23 $RC; ok;  printf "%bsettings.json 저장 완료%b" "$GR" "$RST"
sp 0.4

# ── DO 전용: Spaces 버킷 먼저 생성 ──────────────────────────────────────────
at 24 $LC;  printf "%b      (단계 없음)%b" "$DIM" "$RST"
at 24 $RC;  printf "%b[3/9]%b DO Spaces 버킷 생성 %b(Terraform 상태용)%b" "$DIM" "$RST" "$DIM" "$RST"
sp 0.3
at 25 $RC;  run; printf "terraform apply -target=digitalocean_spaces_bucket"
sp 0.4
pbar 25 $RC 0 20; sp 0.1
apbar 25 $RC 0 100 0.04 20
at 25 $(( RC + 23 )); printf " %b완료%b" "$GR" "$RST"
sp 0.4

# ══════════════════════════════════════════════════════════════════════════════
# SCENE 5 — Terraform Init
# ══════════════════════════════════════════════════════════════════════════════
hr 26
step 27  "3" "8" "Terraform init  (S3 backend)" \
             "4" "9" "Terraform init  (Spaces backend)"
sp 0.3
at 28 $LC;  run; printf "Initializing S3 state backend..."
at 28 $RC;  run; printf "Initializing Spaces state backend..."
sp 0.8
at 28 $LC;  ok;  printf "%bBackend initialized (S3)%b" "$GR" "$RST"
at 28 $RC;  ok;  printf "%bBackend initialized (DO Spaces)%b" "$GR" "$RST"
sp 0.5

# ══════════════════════════════════════════════════════════════════════════════
# SCENE 6 — Terraform Apply
# ══════════════════════════════════════════════════════════════════════════════
hr 29
step 30  "4" "8" "Terraform apply  EKS · VPC · RDS · EFS" \
             "5" "9" "Terraform apply  DOKS · VPC · PostgreSQL"
sp 0.3
at 31 $LC;  printf "    %bEKS 클러스터 · RDS · EFS · VPC%b" "$DIM" "$RST"
at 31 $RC;  printf "    %bDOKS 클러스터 · PostgreSQL · VPC%b" "$DIM" "$RST"

pbar 32 $LC 0 20;  pbar 32 $RC 0 20
sp 0.2

# Animate both — AWS slower (more resources), DO slightly faster
for p in 5 10 15 20 25 30 35 40 45 50 55 60 65 70 75 80 85 90 95 100; do
  pbar 32 $LC "$p" 20
  do_p=$(( p < 100 ? p + 5 : 100 ))
  pbar 32 $RC "$do_p" 20
  sp 0.11
done
pbar 32 $LC 100 20
pbar 32 $RC 100 20

at 33 $LC;  ok;  printf "%b완료  (%b7개 리소스%b 생성)%b" "$GR" "$RST" "$WH" "$GR" "$RST"
at 33 $RC;  ok;  printf "%b완료  (%b3개 리소스%b 생성)%b" "$GR" "$RST" "$WH" "$GR" "$RST"
sp 0.5

# ══════════════════════════════════════════════════════════════════════════════
# SCENE 7 — kubeconfig
# ══════════════════════════════════════════════════════════════════════════════
hr 34
step 35  "5" "8" "kubeconfig 설정" \
             "6" "9" "kubeconfig 설정"
sp 0.3
at 36 $LC;  run; printf "%baws eks update-kubeconfig --name thanos-chain%b" "$DIM" "$RST"
at 36 $RC;  run; printf "%bdoctl k8s cluster kubeconfig save thanos-chain%b" "$DIM" "$RST"
sp 0.6
at 36 $LC;  ok;  printf "%bkubeconfig updated%b" "$GR" "$RST"
at 36 $RC;  ok;  printf "%bkubeconfig saved%b" "$GR" "$RST"
sp 0.5

# ══════════════════════════════════════════════════════════════════════════════
# SCENE 8 — Helm Install
# ══════════════════════════════════════════════════════════════════════════════
hr 37
step 38  "6" "8" "Helm 차트 설치" \
             "7" "9" "Helm 차트 설치"
sp 0.3

declare -a CHARTS=("op-geth" "op-node" "op-batcher" "op-proposer")
HROW=39
for ch in "${CHARTS[@]}"; do
  at $HROW $LC; run; printf "%b%s%b" "$WH" "$ch" "$RST"
  at $HROW $RC; run; printf "%b%s%b" "$WH" "$ch" "$RST"
  sp 0.35
  at $HROW $LC; ok;  printf "%b%s%b" "$GR" "$ch" "$RST"
  at $HROW $RC; ok;  printf "%b%s%b" "$GR" "$ch" "$RST"
  sp 0.15
  (( HROW++ ))
done
sp 0.4

# ══════════════════════════════════════════════════════════════════════════════
# SCENE 9 — Done + Summary
# ══════════════════════════════════════════════════════════════════════════════
banner 43
at 44 $LC;  printf "%b🎉 AWS 배포 완료%b  %b(약 25분)%b" "$B$YE" "$RST" "$DIM" "$RST"
at 44 $RC;  printf "%b🎉 DO  배포 완료%b  %b(약 19분)%b" "$B$AQ" "$RST" "$DIM" "$RST"
banner 45
sp 0.5

# Summary table
at 46 0
COL1=3; COL2=24; COL3=46; COL4=66
printf "%b%-20s  %-20s  %-20s  %-18s%b" "$DIM" "항목" "AWS" "DigitalOcean" "비고" "$RST"
hr 47

declare -a LABELS=("자격증명 수" "Terraform 리소스" "배포 단계" "파괴 단계" "백업" "예상 월 비용")
declare -a AWS_V=("1쌍 (IAM)"    "7개 모듈"         "8단계"    "7단계"    "✅ EFS+DynamoDB"  "~\$150+")
declare -a DO_V=( "2종 (토큰+Spaces)" "3개 단일파일" "9단계"    "3단계"    "❌ 미지원"       "~\$80+")
declare -a NOTES=("DO는 Spaces HMAC 별도" "AWS가 세밀한 제어" "DO는 2단계 TF" "DO가 훨씬 단순" "" "")

ROW=48
for i in 0 1 2 3 4 5; do
  at $ROW $COL1; printf "%b%-20s%b" "$WH" "${LABELS[$i]}" "$RST"
  at $ROW $COL2; printf "%b%-20s%b" "$YE" "${AWS_V[$i]}"  "$RST"
  at $ROW $COL3; printf "%b%-20s%b" "$AQ" "${DO_V[$i]}"   "$RST"
  at $ROW $COL4; printf "%b%s%b"    "$DIM" "${NOTES[$i]}" "$RST"
  (( ROW++ ))
done

hr $ROW
(( ROW++ ))
at $ROW $COL1; printf "%b권장 대상%b" "$B$WH" "$RST"
at $ROW $COL2; printf "%b프로덕션 · 기업%b" "$YE" "$RST"
at $ROW $COL3; printf "%b개발 · 스타트업%b" "$AQ" "$RST"

sp 0.5
pause_exit
