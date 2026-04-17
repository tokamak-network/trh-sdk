# Lessons Learned

## 2026-02-19

- 배포 컴포넌트 교체 문서는 단순 실행 절차보다 "코드 결합 지점 + 운영 영향 범위 + 롤백 전략"을 함께 제공해야 실제 적용 가능성이 높다.
- `trh-sdk`처럼 특정 컴포넌트명(`op-geth`) 하드코딩이 많은 구조에서는 1차 전환 시 리소스명 유지 전략이 리스크를 크게 줄인다.

## 2026-04-18

- 런타임 바이너리를 `~/.trh/bin` 같은 캐시 경로에 내려받는 코드라면, 다운로드 전에 parent directory 생성까지 함수 책임으로 포함해야 컨테이너 첫 기동에서 깨지지 않는다.
- Electron E2E가 `FailedToDeploy`로만 끝날 때는 teardown 전에 backend deployment payload와 container logs를 함께 확인해야 UI 문제와 runtime 문제를 빠르게 분리할 수 있다.
